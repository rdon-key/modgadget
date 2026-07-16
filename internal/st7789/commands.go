package st7789

// ST7789 commands used by the minimal initialization and streaming paths.
const (
	commandSoftwareReset = 0x01
	commandSleepOut      = 0x11
	commandNormalOn      = 0x13
	commandInvertOff     = 0x20
	commandInvertOn      = 0x21
	commandDisplayOn     = 0x29
	commandColumnAddress = 0x2a
	commandRowAddress    = 0x2b
	commandMemoryWrite   = 0x2c
	commandMemoryAccess  = 0x36
	commandPixelFormat   = 0x3a
	commandPorchControl  = 0xb2
	commandFrameRate2    = 0xc6
)

const (
	madctlMY = 0x80
	madctlMX = 0x40
	madctlMV = 0x20

	pixelFormatRGB565 = 0x55
)
