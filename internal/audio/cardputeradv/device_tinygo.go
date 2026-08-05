//go:build tinygo && esp32s3

package cardputeradv

import (
	"device/esp"
	"fmt"
	"machine"
	"runtime/volatile"
	"unsafe"
)

// Set audioDebug temporarily when inspecting GDMA state on hardware.
const audioDebug = false

func audioDebugText(text string) {
	if audioDebug {
		println(text)
	}
}

func audioDebugToneQueued(frames int, remaining uint32) {
	if audioDebug {
		println("audio: tone queued frames=", frames, " remaining=", remaining)
	}
}

const (
	i2sBCLKPin = machine.GPIO41
	i2sDataPin = machine.GPIO42
	i2sWSPin   = machine.GPIO43

	i2s1BCLKOutSignal = uint32(28)
	i2s1WSOutSignal   = uint32(29)
	i2s1DataOutSignal = uint32(30)
	gdmaI2S1Trigger   = uint32(4)

	dmaOwner = uint32(1 << 31)
	dmaEOF   = uint32(1 << 30)
)

// dmaDescriptor is the 12-byte ESP32-S3 GDMA descriptor described by
// Espressif's dma_types.h. Player retains hardwareDevice for the entire DMA
// lifetime. TinyGo's GC is non-moving, so these fixed fields retain stable
// addresses and are never temporary stack slices.
type dmaDescriptor struct {
	control uint32
	buffer  unsafe.Pointer
	next    unsafe.Pointer
}

type hardwareDevice struct {
	bus             *machine.I2C
	descriptor      dmaDescriptor
	dmaBuffer       [BufferBytes]byte
	inFlight        bool
	started         bool
	dmaStartLogs    uint8
	dmaCompleteLogs uint8
}

func newPCMDevice() pcmDevice { return &hardwareDevice{bus: machine.I2C0} }

func (d *hardwareDevice) Configure() error {
	if d.started {
		_ = muteCodec(d.bus)
		esp.DMA.SetOUT_LINK_CH0_OUTLINK_STOP(1)
		esp.DMA.SetOUT_CONF0_CH0_OUT_RST(1)
		esp.DMA.SetOUT_CONF0_CH0_OUT_RST(0)
		esp.I2S1.SetTX_CONF_TX_START(0)
		d.started = false
	}
	if err := d.bus.Configure(machine.I2CConfig{
		Frequency: 100_000,
		SDA:       machine.GPIO8,
		SCL:       machine.GPIO9,
	}); err != nil {
		return fmt.Errorf("configure codec I2C: %w", err)
	}
	if err := detectCodec(d.bus); err != nil {
		return err
	}
	audioDebugText("audio: codec detected")
	if err := prepareCodec(d.bus); err != nil {
		return err
	}
	d.resetSoftwareState()
	d.configureI2S()
	d.configureDMA()
	if err := d.writeFiniteSilence(); err != nil {
		_ = muteCodec(d.bus)
		return fmt.Errorf("start initial silence: %w", err)
	}
	audioDebugText("audio: silence started")
	if err := unmuteCodec(d.bus); err != nil {
		_ = muteCodec(d.bus)
		return err
	}
	return nil
}

func (d *hardwareDevice) Ready() bool {
	if !d.inFlight {
		return true
	}
	if esp.DMA.GetOUT_INT_RAW_CH0_OUT_EOF() == 0 {
		return false
	}
	raw := esp.DMA.GetOUT_INT_RAW_CH0_OUT_EOF()
	eofDescriptor := esp.DMA.GetOUT_EOF_DES_ADDR_CH0()
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_EOF(1)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_TOTAL_EOF(1)
	d.descriptor.control &^= dmaOwner
	d.inFlight = false
	if d.dmaCompleteLogs < 2 {
		if audioDebug {
			println("audio: dma complete raw=", raw,
				" eof_descriptor=", eofDescriptor,
				" owner=", d.descriptor.control>>31)
		}
		d.dmaCompleteLogs++
	}
	return true
}

func (d *hardwareDevice) WritePCM(data []byte) error {
	if len(data) != BufferBytes {
		return fmt.Errorf("PCM buffer has %d bytes, want %d", len(data), BufferBytes)
	}
	if !d.Ready() {
		return fmt.Errorf("DMA is busy")
	}
	copy(d.dmaBuffer[:], data)
	d.descriptor = d.makeDescriptor(&d.dmaBuffer)
	d.startDMA()
	d.inFlight = true
	return nil
}

func (d *hardwareDevice) Stop() error {
	d.abortDMA()
	return d.writeFiniteSilence()
}

func (d *hardwareDevice) resetSoftwareState() {
	d.inFlight = false
	d.started = false
	d.descriptor = dmaDescriptor{}
	for i := range d.dmaBuffer {
		d.dmaBuffer[i] = 0
	}
	d.dmaStartLogs = 0
	d.dmaCompleteLogs = 0
}

func (d *hardwareDevice) makeDescriptor(buffer *[BufferBytes]byte) dmaDescriptor {
	return dmaDescriptor{
		control: uint32(BufferBytes) | uint32(BufferBytes)<<12 | dmaEOF | dmaOwner,
		buffer:  unsafe.Pointer(&buffer[0]),
		next:    nil,
	}
}

func (d *hardwareDevice) writeFiniteSilence() error {
	for i := range d.dmaBuffer {
		d.dmaBuffer[i] = 0
	}
	d.descriptor = d.makeDescriptor(&d.dmaBuffer)
	d.startDMA()
	d.inFlight = true
	if !d.started {
		if err := d.startI2S(); err != nil {
			d.abortDMA()
			return err
		}
		d.started = true
	}
	return nil
}

// startDMA submits exactly one descriptor. The nil next pointer makes OUT_EOF
// observable and returns ownership before this fixed buffer is reused.
func (d *hardwareDevice) startDMA() {
	esp.DMA.SetOUT_LINK_CH0_OUTLINK_STOP(1)
	esp.DMA.SetOUT_CONF0_CH0_OUT_RST(1)
	esp.DMA.SetOUT_CONF0_CH0_OUT_RST(0)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_EOF(1)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_TOTAL_EOF(1)
	esp.DMA.SetOUT_LINK_CH0_OUTLINK_ADDR(uint32(uintptr(unsafe.Pointer(&d.descriptor))))
	esp.DMA.SetOUT_LINK_CH0_OUTLINK_START(1)
	if d.dmaStartLogs < 2 {
		if audioDebug {
			println("audio: dma started out_link=", esp.DMA.GetOUT_LINK_CH0_OUTLINK_ADDR(),
				" start=", esp.DMA.GetOUT_LINK_CH0_OUTLINK_START(),
				" stop=", esp.DMA.GetOUT_LINK_CH0_OUTLINK_STOP(),
				" park=", esp.DMA.GetOUT_LINK_CH0_OUTLINK_PARK(),
				" raw=", esp.DMA.GetOUT_INT_RAW_CH0_OUT_EOF(),
				" owner=", d.descriptor.control>>31,
				" length=", (d.descriptor.control>>12)&0xfff,
				" next=", uintptr(d.descriptor.next))
		}
		d.dmaStartLogs++
	}
}

func (d *hardwareDevice) abortDMA() {
	esp.DMA.SetOUT_LINK_CH0_OUTLINK_STOP(1)
	esp.DMA.SetOUT_CONF0_CH0_OUT_RST(1)
	esp.DMA.SetOUT_CONF0_CH0_OUT_RST(0)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_EOF(1)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_TOTAL_EOF(1)
	d.descriptor.control &^= dmaOwner
	d.inFlight = false
}

func (d *hardwareDevice) configureDMA() {
	esp.SYSTEM.SetPERIP_CLK_EN1_DMA_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(0)
	esp.DMA.SetOUT_CONF0_CH0_OUT_RST(1)
	esp.DMA.SetOUT_CONF0_CH0_OUT_RST(0)
	esp.DMA.SetOUT_CONF0_CH0_OUT_AUTO_WRBACK(0)
	esp.DMA.SetOUT_CONF0_CH0_OUT_EOF_MODE(0)
	esp.DMA.SetOUT_CONF0_CH0_OUTDSCR_BURST_EN(0)
	esp.DMA.SetOUT_CONF0_CH0_OUT_DATA_BURST_EN(0)
	esp.DMA.SetOUT_CONF1_CH0_OUT_CHECK_OWNER(1)
	esp.DMA.SetOUT_PERI_SEL_CH0_PERI_OUT_SEL(gdmaI2S1Trigger)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_EOF(1)
	esp.DMA.SetOUT_INT_CLR_CH0_OUT_TOTAL_EOF(1)
}

func (d *hardwareDevice) configureI2S() {
	esp.SYSTEM.SetPERIP_CLK_EN0_I2S1_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN0_I2S1_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN0_I2S1_RST(0)

	configurePeripheralOutput(i2sBCLKPin, i2s1BCLKOutSignal)
	configurePeripheralOutput(i2sWSPin, i2s1WSOutSignal)
	configurePeripheralOutput(i2sDataPin, i2s1DataOutSignal)

	hw := esp.I2S1
	hw.SetTX_CONF_TX_START(0)
	hw.SetTX_CONF_TX_RESET(1)
	hw.SetTX_CONF_TX_RESET(0)
	hw.SetTX_CONF_TX_FIFO_RESET(1)
	hw.SetTX_CONF_TX_FIFO_RESET(0)
	hw.SetTX_CONF_TX_SLAVE_MOD(0)
	hw.SetTX_CONF_TX_PDM_EN(0)
	hw.SetTX_CONF_TX_TDM_EN(1)
	hw.SetTX_CONF_TX_PCM_BYPASS(1)
	hw.SetTX_CONF_TX_MONO(0)
	hw.SetTX_CONF_TX_CHAN_EQUAL(0)
	hw.SetTX_CONF_TX_BIG_ENDIAN(0)
	hw.SetTX_CONF_TX_LEFT_ALIGN(1)
	hw.SetTX_CONF_TX_BIT_ORDER(0)
	hw.SetTX_CONF_TX_WS_IDLE_POL(0)

	// 16-bit stereo Philips I2S. A 160 MHz source is divided to a
	// 12.288 MHz module clock (13 + 1/48), then by 8 to 1.536 MHz BCLK.
	hw.SetTX_CONF1_TX_BCK_DIV_NUM(i2sBCLKDivider - 1)
	hw.SetTX_CONF1_TX_BITS_MOD(15)
	hw.SetTX_CONF1_TX_TDM_CHAN_BITS(15)
	hw.SetTX_CONF1_TX_HALF_SAMPLE_BITS(15)
	hw.SetTX_CONF1_TX_TDM_WS_WIDTH(15)
	hw.SetTX_CONF1_TX_MSB_SHIFT(1)
	hw.SetTX_CONF1_TX_BCK_NO_DLY(0)
	hw.SetTX_TDM_CTRL_TX_TDM_TOT_CHAN_NUM(1)
	hw.SetTX_TDM_CTRL_TX_TDM_CHAN0_EN(1)
	hw.SetTX_TDM_CTRL_TX_TDM_CHAN1_EN(1)
	hw.SetTX_TDM_CTRL_TX_TDM_SKIP_MSK_EN(0)

	hw.SetTX_CLKM_CONF_TX_CLK_SEL(2)
	hw.SetTX_CLKM_CONF_TX_CLKM_DIV_NUM(2)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_YN1(0)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_Y(1)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_Z(0)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_X(0)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_YN1(0)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_Z(1)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_Y(0)
	hw.SetTX_CLKM_DIV_CONF_TX_CLKM_DIV_X(47)
	hw.SetTX_CLKM_CONF_TX_CLKM_DIV_NUM(i2sClockDivInt)
	hw.SetTX_CLKM_CONF_CLK_EN(1)
	hw.SetTX_CLKM_CONF_TX_CLK_ACTIVE(1)
}

func (d *hardwareDevice) startI2S() error {
	hw := esp.I2S1
	hw.SetTX_CONF_TX_UPDATE(1)
	// This is a one-time hardware register latch, not an Update-path wait.
	for attempts := 0; attempts < 10_000 && hw.GetTX_CONF_TX_UPDATE() != 0; attempts++ {
	}
	if hw.GetTX_CONF_TX_UPDATE() != 0 {
		return fmt.Errorf("I2S1 register update timed out")
	}
	hw.SetTX_CONF_TX_START(1)
	return nil
}

func configurePeripheralOutput(pin machine.Pin, signal uint32) {
	ioConfig := uint32(1<<esp.IO_MUX_GPIO_MCU_SEL_Pos) |
		esp.IO_MUX_GPIO_FUN_IE |
		uint32(2<<esp.IO_MUX_GPIO_FUN_DRV_Pos)
	io := (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.IO_MUX.GPIO0), uintptr(pin)*4))
	io.Set(ioConfig)
	if pin < 32 {
		esp.GPIO.ENABLE_W1TS.Set(1 << pin)
	} else {
		esp.GPIO.ENABLE1_W1TS.Set(1 << (pin - 32))
	}
	out := (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.GPIO.FUNC0_OUT_SEL_CFG), uintptr(pin)*4))
	out.Set(signal)
}
