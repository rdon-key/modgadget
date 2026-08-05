package modgadget

// Keyboard provides queued key events.
type Keyboard interface {
	ReadKeyEvent() (KeyEvent, bool)
}

// VolumeController handles the standard system volume operations.
type VolumeController interface {
	VolumeUp()
	VolumeDown()
	ToggleMute()
}

// KeyEvent describes one physical key transition.
type KeyEvent struct {
	Code      KeyCode
	Rune      rune
	Action    KeyAction
	Modifiers Modifiers
}

// KeyCode identifies the logical key after the keyboard applies its active
// modifier layer. Values through KeyLeftMeta use USB HID Keyboard/Keypad Usage
// IDs.
type KeyCode uint16

const (
	// KeyUnknown identifies a key that has no known mapping.
	KeyUnknown KeyCode = 0x00
	// KeyA identifies the A key.
	KeyA KeyCode = 0x04
	// KeyB identifies the B key.
	KeyB KeyCode = 0x05
	// KeyC identifies the C key.
	KeyC KeyCode = 0x06
	// KeyD identifies the D key.
	KeyD KeyCode = 0x07
	// KeyE identifies the E key.
	KeyE KeyCode = 0x08
	// KeyF identifies the F key.
	KeyF KeyCode = 0x09
	// KeyG identifies the G key.
	KeyG KeyCode = 0x0a
	// KeyH identifies the H key.
	KeyH KeyCode = 0x0b
	// KeyI identifies the I key.
	KeyI KeyCode = 0x0c
	// KeyJ identifies the J key.
	KeyJ KeyCode = 0x0d
	// KeyK identifies the K key.
	KeyK KeyCode = 0x0e
	// KeyL identifies the L key.
	KeyL KeyCode = 0x0f
	// KeyM identifies the M key.
	KeyM KeyCode = 0x10
	// KeyN identifies the N key.
	KeyN KeyCode = 0x11
	// KeyO identifies the O key.
	KeyO KeyCode = 0x12
	// KeyP identifies the P key.
	KeyP KeyCode = 0x13
	// KeyQ identifies the Q key.
	KeyQ KeyCode = 0x14
	// KeyR identifies the R key.
	KeyR KeyCode = 0x15
	// KeyS identifies the S key.
	KeyS KeyCode = 0x16
	// KeyT identifies the T key.
	KeyT KeyCode = 0x17
	// KeyU identifies the U key.
	KeyU KeyCode = 0x18
	// KeyV identifies the V key.
	KeyV KeyCode = 0x19
	// KeyW identifies the W key.
	KeyW KeyCode = 0x1a
	// KeyX identifies the X key.
	KeyX KeyCode = 0x1b
	// KeyY identifies the Y key.
	KeyY KeyCode = 0x1c
	// KeyZ identifies the Z key.
	KeyZ KeyCode = 0x1d
	// Key1 identifies the 1 key.
	Key1 KeyCode = 0x1e
	// Key2 identifies the 2 key.
	Key2 KeyCode = 0x1f
	// Key3 identifies the 3 key.
	Key3 KeyCode = 0x20
	// Key4 identifies the 4 key.
	Key4 KeyCode = 0x21
	// Key5 identifies the 5 key.
	Key5 KeyCode = 0x22
	// Key6 identifies the 6 key.
	Key6 KeyCode = 0x23
	// Key7 identifies the 7 key.
	Key7 KeyCode = 0x24
	// Key8 identifies the 8 key.
	Key8 KeyCode = 0x25
	// Key9 identifies the 9 key.
	Key9 KeyCode = 0x26
	// Key0 identifies the 0 key.
	Key0 KeyCode = 0x27
	// KeyEnter identifies the Enter key.
	KeyEnter KeyCode = 0x28
	// KeyEscape identifies the Escape key.
	KeyEscape KeyCode = 0x29
	// KeyBackspace identifies the Backspace/Delete key.
	KeyBackspace KeyCode = 0x2a
	// KeyTab identifies the Tab key.
	KeyTab KeyCode = 0x2b
	// KeySpace identifies the Space key.
	KeySpace KeyCode = 0x2c
	// KeyMinus identifies the minus key.
	KeyMinus KeyCode = 0x2d
	// KeyEqual identifies the equal key.
	KeyEqual KeyCode = 0x2e
	// KeyLeftBracket identifies the left bracket key.
	KeyLeftBracket KeyCode = 0x2f
	// KeyRightBracket identifies the right bracket key.
	KeyRightBracket KeyCode = 0x30
	// KeyBackslash identifies the backslash key.
	KeyBackslash KeyCode = 0x31
	// KeySemicolon identifies the semicolon key.
	KeySemicolon KeyCode = 0x33
	// KeyApostrophe identifies the apostrophe key.
	KeyApostrophe KeyCode = 0x34
	// KeyGrave identifies the grave-accent key.
	KeyGrave KeyCode = 0x35
	// KeyComma identifies the comma key.
	KeyComma KeyCode = 0x36
	// KeyPeriod identifies the period key.
	KeyPeriod KeyCode = 0x37
	// KeySlash identifies the slash key.
	KeySlash KeyCode = 0x38
	// KeyF1 identifies the F1 key.
	KeyF1 KeyCode = 0x3a
	// KeyF2 identifies the F2 key.
	KeyF2 KeyCode = 0x3b
	// KeyF3 identifies the F3 key.
	KeyF3 KeyCode = 0x3c
	// KeyF4 identifies the F4 key.
	KeyF4 KeyCode = 0x3d
	// KeyF5 identifies the F5 key.
	KeyF5 KeyCode = 0x3e
	// KeyF6 identifies the F6 key.
	KeyF6 KeyCode = 0x3f
	// KeyF7 identifies the F7 key.
	KeyF7 KeyCode = 0x40
	// KeyF8 identifies the F8 key.
	KeyF8 KeyCode = 0x41
	// KeyF9 identifies the F9 key.
	KeyF9 KeyCode = 0x42
	// KeyF10 identifies the F10 key.
	KeyF10 KeyCode = 0x43
	// KeyF11 identifies the F11 key.
	KeyF11 KeyCode = 0x44
	// KeyF12 identifies the F12 key.
	KeyF12 KeyCode = 0x45
	// KeyDelete identifies the forward-delete key.
	KeyDelete KeyCode = 0x4c
	// KeyArrowRight identifies the right-arrow key.
	KeyArrowRight KeyCode = 0x4f
	// KeyArrowLeft identifies the left-arrow key.
	KeyArrowLeft KeyCode = 0x50
	// KeyArrowDown identifies the down-arrow key.
	KeyArrowDown KeyCode = 0x51
	// KeyArrowUp identifies the up-arrow key.
	KeyArrowUp KeyCode = 0x52
	// KeyLeftControl identifies the Cardputer ADV Control key.
	KeyLeftControl KeyCode = 0xe0
	// KeyLeftShift identifies the Cardputer ADV Aa/Shift key.
	KeyLeftShift KeyCode = 0xe1
	// KeyLeftAlt identifies the Cardputer ADV Alt key.
	KeyLeftAlt KeyCode = 0xe2
	// KeyLeftMeta identifies the Cardputer ADV Opt key.
	KeyLeftMeta KeyCode = 0xe3
	// KeyFn identifies the Cardputer ADV Fn key; it is outside USB HID usages.
	KeyFn KeyCode = 0x100
)

// KeyAction identifies an unknown, pressed, or released key transition.
type KeyAction uint8

const (
	// KeyActionUnknown reports that no key transition is specified.
	KeyActionUnknown KeyAction = iota
	// KeyDown reports a transition to pressed.
	KeyDown
	// KeyUp reports a transition to released.
	KeyUp
)

// Modifiers is a bit set of modifiers active at an event.
type Modifiers uint16

const (
	// ModShift reports an active Shift key.
	ModShift Modifiers = 1 << iota
	// ModControl reports an active Control key.
	ModControl
	// ModAlt reports an active Alt key.
	ModAlt
	// ModMeta reports an active Meta/Option key.
	ModMeta
	// ModFn reports an active Fn key.
	ModFn
)

// Has reports whether all bits in value are set.
func (m Modifiers) Has(value Modifiers) bool { return m&value == value }

// KeyHandler handles a direct key event and reports whether it consumed it.
type KeyHandler func(KeyEvent) bool

// ListenerID identifies a registered event handler. Zero is invalid.
type ListenerID uint16

type keyListener struct {
	id      ListenerID
	handler KeyHandler
}

// WithKeyboard sets the keyboard event source used by Gadget.Update.
func WithKeyboard(keyboard Keyboard) Option {
	return func(g *Gadget) { g.keyboard = keyboard }
}

// WithVolumeController enables standard volume shortcuts for controller.
func WithVolumeController(controller VolumeController) Option {
	return func(g *Gadget) { g.volumeController = controller }
}

const (
	systemVolumeDownKey uint8 = 1 << iota
	systemVolumeUpKey
	systemMuteKey
)

func systemVolumeKeyBit(code KeyCode) uint8 {
	switch code {
	case KeyF11:
		return systemVolumeDownKey
	case KeyF12:
		return systemVolumeUpKey
	case KeyM:
		return systemMuteKey
	default:
		return 0
	}
}

func (g *Gadget) handleSystemKey(event KeyEvent) bool {
	if g == nil {
		return false
	}
	bit := systemVolumeKeyBit(event.Code)
	if bit == 0 {
		return false
	}

	switch event.Action {
	case KeyDown:
		if !event.Modifiers.Has(ModFn) {
			g.capturedSystemKeys &^= bit
			return false
		}
		if g.volumeController == nil {
			return false
		}
		if g.capturedSystemKeys&bit != 0 {
			return true
		}
		g.capturedSystemKeys |= bit
		switch event.Code {
		case KeyF12:
			g.volumeController.VolumeUp()
		case KeyF11:
			g.volumeController.VolumeDown()
		case KeyM:
			g.volumeController.ToggleMute()
		}
		return true

	case KeyUp:
		if g.capturedSystemKeys&bit == 0 {
			return false
		}
		g.capturedSystemKeys &^= bit
		return true
	}

	return false
}

// OnKey registers handler and returns its ListenerID.
func (g *Gadget) OnKey(handler KeyHandler) ListenerID {
	if g == nil || handler == nil {
		return 0
	}
	for attempts := 0; attempts < 1<<16-1; attempts++ {
		g.nextListener++
		if g.nextListener == 0 {
			g.nextListener++
		}
		id := g.nextListener
		used := false
		for i := range g.keyListeners {
			if g.keyListeners[i].id == id && g.keyListeners[i].handler != nil {
				used = true
				break
			}
		}
		if !used {
			g.keyListeners = append(g.keyListeners, keyListener{id: id, handler: handler})
			return id
		}
	}
	return 0
}

// RemoveListener removes a listener by ID.
func (g *Gadget) RemoveListener(id ListenerID) bool {
	if g == nil || id == 0 {
		return false
	}
	for i := range g.keyListeners {
		if g.keyListeners[i].id == id && g.keyListeners[i].handler != nil {
			g.keyListeners[i].handler = nil
			if g.keyDispatchDepth == 0 {
				g.compactKeyListeners()
			} else {
				g.keyListenersDirty = true
			}
			return true
		}
	}
	return false
}

func (g *Gadget) dispatchKey(event KeyEvent) {
	g.keyDispatchDepth++
	count := len(g.keyListeners)
	for i := 0; i < count; i++ {
		handler := g.keyListeners[i].handler
		if handler != nil && handler(event) {
			break
		}
	}
	g.keyDispatchDepth--
	if g.keyDispatchDepth == 0 && g.keyListenersDirty {
		g.compactKeyListeners()
	}
}

func (g *Gadget) compactKeyListeners() {
	next := 0
	for i := range g.keyListeners {
		if g.keyListeners[i].handler != nil {
			g.keyListeners[next] = g.keyListeners[i]
			next++
		}
	}
	clear(g.keyListeners[next:])
	g.keyListeners = g.keyListeners[:next]
	g.keyListenersDirty = false
}
