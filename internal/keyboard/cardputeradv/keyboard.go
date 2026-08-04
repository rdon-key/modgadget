// Package cardputeradv adapts the Cardputer ADV TCA8418 keyboard to ModGadget.
package cardputeradv

import (
	"fmt"

	"github.com/rdon-key/modgadget"
)

const (
	address          = uint16(0x34)
	registerCFG      = byte(0x01)
	registerINTStat  = byte(0x02)
	registerKeyCount = byte(0x03)
	registerKeyEvent = byte(0x04)
	registerKPGPIO1  = byte(0x1d)
	registerKPGPIO2  = byte(0x1e)
	registerKPGPIO3  = byte(0x1f)

	interruptClearMask = byte(0x1f)
	interruptOverflow  = byte(1 << 3)
	configKeyEvents    = byte(1<<0 | 1<<3)
	hardwareFIFOSize   = 10
	eventQueueSize     = 32
)

// Bus is the I2C transaction subset required by Keyboard.
type Bus interface {
	Tx(address uint16, write, read []byte) error
}

// Keyboard reads direct press and release events from a Cardputer ADV.
type Keyboard struct {
	bus          Bus
	queue        [eventQueueSize]modgadget.KeyEvent
	head         uint8
	count        uint8
	modifiers    modgadget.Modifiers
	pressed      [69]bool
	pressedCodes [69]modgadget.KeyCode
	lastErr      error
	dropped      uint32
}

var _ modgadget.Keyboard = (*Keyboard)(nil)

// New returns a Cardputer ADV keyboard that uses bus.
func New(bus Bus) *Keyboard { return &Keyboard{bus: bus} }

// Configure initializes the TCA8418 matrix and clears pending events.
func (keyboard *Keyboard) Configure() error {
	if keyboard == nil || keyboard.bus == nil {
		return fmt.Errorf("cardputeradv: keyboard bus is nil")
	}
	keyboard.resetSoftwareState(true)
	if err := keyboard.writeRegister(registerCFG, 0); err != nil {
		return err
	}
	if err := keyboard.writeRegister(registerKPGPIO1, 0x7f); err != nil {
		return err
	}
	if err := keyboard.writeRegister(registerKPGPIO2, 0xff); err != nil {
		return err
	}
	if err := keyboard.writeRegister(registerKPGPIO3, 0); err != nil {
		return err
	}
	count, err := keyboard.eventCount()
	if err != nil {
		return err
	}
	if count > hardwareFIFOSize {
		count = hardwareFIFOSize
	}
	for i := 0; i < count; i++ {
		if _, err := keyboard.readRegister(registerKeyEvent); err != nil {
			return err
		}
	}
	if err := keyboard.writeRegister(registerINTStat, interruptClearMask); err != nil {
		return err
	}
	return keyboard.writeRegister(registerCFG, configKeyEvents)
}

// ReadKeyEvent returns and removes the next queued event. When the local queue
// is empty it polls the TCA8418 FIFO once.
func (keyboard *Keyboard) ReadKeyEvent() (modgadget.KeyEvent, bool) {
	if keyboard == nil || keyboard.bus == nil {
		return modgadget.KeyEvent{}, false
	}
	if keyboard.count == 0 {
		if err := keyboard.poll(); err != nil {
			keyboard.lastErr = err
			return modgadget.KeyEvent{}, false
		}
	}
	if keyboard.count == 0 {
		return modgadget.KeyEvent{}, false
	}
	event := keyboard.queue[keyboard.head]
	keyboard.head = (keyboard.head + 1) % eventQueueSize
	keyboard.count--
	return event, true
}

// Err returns the last polling error, if any.
func (keyboard *Keyboard) Err() error {
	if keyboard == nil {
		return nil
	}
	return keyboard.lastErr
}

// DroppedEvents returns the number of newest events discarded because the
// fixed queue was full.
func (keyboard *Keyboard) DroppedEvents() uint32 {
	if keyboard == nil {
		return 0
	}
	return keyboard.dropped
}

func (keyboard *Keyboard) poll() error {
	status, err := keyboard.readRegister(registerINTStat)
	if err != nil {
		return err
	}
	if status&interruptOverflow != 0 {
		keyboard.dropped++
		keyboard.resetInputState()
	}
	count, err := keyboard.eventCount()
	if err != nil {
		return err
	}
	if count > hardwareFIFOSize {
		count = hardwareFIFOSize
	}
	for i := 0; i < count; i++ {
		raw, err := keyboard.readRegister(registerKeyEvent)
		if err != nil {
			return err
		}
		keyboard.processRaw(raw)
	}
	if status != 0 || count > 0 {
		if err := keyboard.writeRegister(registerINTStat, status&interruptClearMask); err != nil {
			return err
		}
	}
	return nil
}

func (keyboard *Keyboard) processRaw(raw byte) {
	keyNumber := raw & 0x7f
	mapping, ok := keyMappings[keyNumber]
	if !ok || mapping.code == modgadget.KeyUnknown {
		return
	}
	isPressed := raw&0x80 != 0
	if keyboard.pressed[keyNumber] == isPressed {
		return
	}
	if isPressed {
		keyboard.pressed[keyNumber] = true
		if mapping.modifier != 0 {
			keyboard.modifiers |= mapping.modifier
		}
		code, value := keyboard.resolve(mapping)
		keyboard.pressedCodes[keyNumber] = code
		if code == modgadget.KeyUnknown {
			return
		}
		keyboard.enqueue(modgadget.KeyEvent{
			Code: code, Rune: value, Action: modgadget.KeyDown, Modifiers: keyboard.modifiers,
		})
		return
	}

	keyboard.pressed[keyNumber] = false
	code := keyboard.pressedCodes[keyNumber]
	keyboard.pressedCodes[keyNumber] = modgadget.KeyUnknown
	if mapping.modifier != 0 {
		keyboard.modifiers &^= mapping.modifier
	}
	if code != modgadget.KeyUnknown {
		keyboard.enqueue(modgadget.KeyEvent{Code: code, Action: modgadget.KeyUp, Modifiers: keyboard.modifiers})
	}
}

func (keyboard *Keyboard) resolve(mapping keyMapping) (modgadget.KeyCode, rune) {
	if mapping.modifier != 0 {
		return mapping.code, 0
	}
	if keyboard.modifiers.Has(modgadget.ModFn) {
		return mapping.fnCode, 0
	}
	value := mapping.normal
	if keyboard.modifiers.Has(modgadget.ModShift) {
		value = mapping.shift
	}
	shortcutModifiers := modgadget.ModControl | modgadget.ModAlt | modgadget.ModMeta
	if keyboard.modifiers&shortcutModifiers != 0 {
		value = 0
	}
	return mapping.code, value
}

func (keyboard *Keyboard) resetInputState() {
	keyboard.head, keyboard.count = 0, 0
	keyboard.modifiers = 0
	keyboard.pressed = [69]bool{}
	keyboard.pressedCodes = [69]modgadget.KeyCode{}
}

func (keyboard *Keyboard) resetSoftwareState(resetDropped bool) {
	keyboard.resetInputState()
	keyboard.lastErr = nil
	if resetDropped {
		keyboard.dropped = 0
	}
}

func (keyboard *Keyboard) enqueue(event modgadget.KeyEvent) {
	if keyboard.count == eventQueueSize {
		keyboard.dropped++
		return
	}
	index := (int(keyboard.head) + int(keyboard.count)) % eventQueueSize
	keyboard.queue[index] = event
	keyboard.count++
}

func (keyboard *Keyboard) eventCount() (int, error) {
	value, err := keyboard.readRegister(registerKeyCount)
	if err != nil {
		return 0, err
	}
	return int(value & 0x0f), nil
}

func (keyboard *Keyboard) readRegister(register byte) (byte, error) {
	var value [1]byte
	request := [1]byte{register}
	if err := keyboard.bus.Tx(address, request[:], value[:]); err != nil {
		return 0, fmt.Errorf("cardputeradv: read register %#02x: %w", register, err)
	}
	return value[0], nil
}

func (keyboard *Keyboard) writeRegister(register, value byte) error {
	data := [2]byte{register, value}
	if err := keyboard.bus.Tx(address, data[:], nil); err != nil {
		return fmt.Errorf("cardputeradv: write register %#02x: %w", register, err)
	}
	return nil
}

type keyMapping struct {
	code     modgadget.KeyCode
	normal   rune
	shift    rune
	fnCode   modgadget.KeyCode
	modifier modgadget.Modifiers
}

var keyMappings = map[byte]keyMapping{
	1:  {code: modgadget.KeyGrave, normal: '`', shift: '~', fnCode: modgadget.KeyEscape},
	5:  {code: modgadget.Key1, normal: '1', shift: '!', fnCode: modgadget.KeyF1},
	11: {code: modgadget.Key2, normal: '2', shift: '@', fnCode: modgadget.KeyF2},
	15: {code: modgadget.Key3, normal: '3', shift: '#', fnCode: modgadget.KeyF3},
	21: {code: modgadget.Key4, normal: '4', shift: '$', fnCode: modgadget.KeyF4},
	25: {code: modgadget.Key5, normal: '5', shift: '%', fnCode: modgadget.KeyF5},
	31: {code: modgadget.Key6, normal: '6', shift: '^', fnCode: modgadget.KeyF6},
	35: {code: modgadget.Key7, normal: '7', shift: '&', fnCode: modgadget.KeyF7},
	41: {code: modgadget.Key8, normal: '8', shift: '*', fnCode: modgadget.KeyF8},
	45: {code: modgadget.Key9, normal: '9', shift: '(', fnCode: modgadget.KeyF9},
	51: {code: modgadget.Key0, normal: '0', shift: ')', fnCode: modgadget.KeyF10},
	55: {code: modgadget.KeyMinus, normal: '-', shift: '_', fnCode: modgadget.KeyF11},
	61: {code: modgadget.KeyEqual, normal: '=', shift: '+', fnCode: modgadget.KeyF12},
	65: {code: modgadget.KeyBackspace, fnCode: modgadget.KeyDelete},
	2:  {code: modgadget.KeyTab},
	6:  {code: modgadget.KeyQ, normal: 'q', shift: 'Q'},
	12: {code: modgadget.KeyW, normal: 'w', shift: 'W'},
	16: {code: modgadget.KeyE, normal: 'e', shift: 'E'},
	22: {code: modgadget.KeyR, normal: 'r', shift: 'R'},
	26: {code: modgadget.KeyT, normal: 't', shift: 'T'},
	32: {code: modgadget.KeyY, normal: 'y', shift: 'Y'},
	36: {code: modgadget.KeyU, normal: 'u', shift: 'U'},
	42: {code: modgadget.KeyI, normal: 'i', shift: 'I'},
	46: {code: modgadget.KeyO, normal: 'o', shift: 'O'},
	52: {code: modgadget.KeyP, normal: 'p', shift: 'P'},
	56: {code: modgadget.KeyLeftBracket, normal: '[', shift: '{'},
	62: {code: modgadget.KeyRightBracket, normal: ']', shift: '}'},
	66: {code: modgadget.KeyBackslash, normal: '\\', shift: '|'},
	3:  {code: modgadget.KeyFn, modifier: modgadget.ModFn},
	7:  {code: modgadget.KeyLeftShift, modifier: modgadget.ModShift},
	13: {code: modgadget.KeyA, normal: 'a', shift: 'A'},
	17: {code: modgadget.KeyS, normal: 's', shift: 'S'},
	23: {code: modgadget.KeyD, normal: 'd', shift: 'D'},
	27: {code: modgadget.KeyF, normal: 'f', shift: 'F'},
	33: {code: modgadget.KeyG, normal: 'g', shift: 'G'},
	37: {code: modgadget.KeyH, normal: 'h', shift: 'H'},
	43: {code: modgadget.KeyJ, normal: 'j', shift: 'J'},
	47: {code: modgadget.KeyK, normal: 'k', shift: 'K'},
	53: {code: modgadget.KeyL, normal: 'l', shift: 'L'},
	57: {code: modgadget.KeySemicolon, normal: ';', shift: ':', fnCode: modgadget.KeyArrowUp},
	63: {code: modgadget.KeyApostrophe, normal: '\'', shift: '"'},
	67: {code: modgadget.KeyEnter},
	4:  {code: modgadget.KeyLeftControl, modifier: modgadget.ModControl},
	8:  {code: modgadget.KeyLeftMeta, modifier: modgadget.ModMeta},
	14: {code: modgadget.KeyLeftAlt, modifier: modgadget.ModAlt},
	18: {code: modgadget.KeyZ, normal: 'z', shift: 'Z'},
	24: {code: modgadget.KeyX, normal: 'x', shift: 'X'},
	28: {code: modgadget.KeyC, normal: 'c', shift: 'C'},
	34: {code: modgadget.KeyV, normal: 'v', shift: 'V'},
	38: {code: modgadget.KeyB, normal: 'b', shift: 'B'},
	44: {code: modgadget.KeyN, normal: 'n', shift: 'N'},
	48: {code: modgadget.KeyM, normal: 'm', shift: 'M'},
	54: {code: modgadget.KeyComma, normal: ',', shift: '<', fnCode: modgadget.KeyArrowLeft},
	58: {code: modgadget.KeyPeriod, normal: '.', shift: '>', fnCode: modgadget.KeyArrowDown},
	64: {code: modgadget.KeySlash, normal: '/', shift: '?', fnCode: modgadget.KeyArrowRight},
	68: {code: modgadget.KeySpace, normal: ' ', shift: ' '},
}
