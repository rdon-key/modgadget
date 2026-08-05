# ModGadget Keyboard API

> The current keyboard API provides direct key events only. It does not perform
> text composition or IME conversion.

## Scope

The API polls a physical keyboard, dispatches `KeyEvent` values, and supports
event consumption and listener removal. It does not provide Text mode, an IME,
composition, focus, widgets, automatic key repeat, or a shortcut manager.

`KeyEvent.Rune` is the single printable character produced directly by the
keyboard's current keymap and modifiers. It is useful for an ASCII typing game,
but it is not an IME commit string.

## Keyboard source

```go
type Keyboard interface {
	ReadKeyEvent() (KeyEvent, bool)
}
```

Each successful call removes the next queued event. An empty queue returns a
zero event and `false`; this is not an error. Register the source at creation:

```go
gadget := modgadget.New(
	panel,
	modgadget.WithStyles(styles),
	modgadget.WithKeyboard(keyboard),
)
```

Keyboard is independent of `Display`. Omitting it, or passing
`WithKeyboard(nil)`, leaves display behavior unchanged.

## Key events

```go
type KeyEvent struct {
	Code      KeyCode
	Rune      rune
	Action    KeyAction
	Modifiers Modifiers
}
```

- `Code` identifies the logical key after the keyboard applies its active
  layer. Standard keys use USB HID Keyboard Usage IDs. `KeyFn` uses a
  ModGadget-specific value because HID has no equivalent.
- `Rune` is nonzero only for printable `KeyDown`. The Cardputer Aa layer
  selects capitals and punctuation. Control, Alt, or Meta suppresses Rune so
  shortcuts are not treated as text.
- `Action` is `KeyActionUnknown`, `KeyDown`, or `KeyUp`; there is no generated
  repeat action. The zero value is `KeyActionUnknown`, not a press.
- `Modifiers` captures state at the event. `Has` can test combined bits.

Enter, Backspace, Tab, Escape, arrows, modifiers, and all release events have a
zero Rune.

### KeyCode values

The public key codes are:

- `KeyUnknown`;
- `KeyA` through `KeyZ`;
- `Key0` through `Key9`;
- `KeyEnter`, `KeyEscape`, `KeyBackspace`, `KeyTab`, and `KeySpace`;
- `KeyMinus`, `KeyEqual`, `KeyLeftBracket`, `KeyRightBracket`,
  `KeyBackslash`, `KeySemicolon`, `KeyApostrophe`, `KeyGrave`, `KeyComma`,
  `KeyPeriod`, and `KeySlash`;
- `KeyF1` through `KeyF12`, `KeyDelete`, `KeyArrowUp`, `KeyArrowDown`,
  `KeyArrowLeft`, and `KeyArrowRight`;
- `KeyLeftControl`, `KeyLeftShift`, `KeyLeftAlt`, `KeyLeftMeta`, and `KeyFn`.

Except for `KeyFn`, their numeric values are the matching USB HID Keyboard/
Keypad Usage IDs. The Cardputer adapter does not claim every public code is a
separate physical key; its exact physical coverage is listed below.

### Actions and modifiers

`KeyActionUnknown`, `KeyDown`, and `KeyUp` are the `KeyAction` values. Modifier
bits can be combined from `ModShift`, `ModControl`, `ModAlt`, `ModMeta`, and
`ModFn`:

```go
if event.Modifiers.Has(modgadget.ModShift | modgadget.ModControl) {
	// Both modifiers were active at this event.
}
```

## Handlers and consumption

```go
id := gadget.OnKey(func(event modgadget.KeyEvent) bool {
	if event.Action != modgadget.KeyDown {
		return false
	}
	if event.Code == modgadget.KeyEnter {
		resetGame()
		return true
	}
	if event.Rune != 0 {
		typeRune(event.Rune)
		return true
	}
	return false
})

removed := gadget.RemoveListener(id)
```

Handlers run in registration order. `false` continues dispatch; `true`
consumes the event. A nil handler returns `ListenerID(0)`. Removing a live ID
returns true; removing zero, an unknown ID, or an already removed ID returns
false. The same handler may be registered more than once under different IDs.

During dispatch, a newly added handler starts with the next event, a handler
removed before its turn is skipped, and a handler may remove itself while its
current call completes. Removed entries are compacted immediately outside
dispatch, or after the outermost active dispatch completes.

## Update loop

`Gadget.Update` processes at most 64 queued key events first, then updates
time-dependent Viewport scrolling. Remaining events stay in the source for the
next Update. This bound prevents a broken source that always returns true from
blocking Viewport updates. Handlers may change Viewport text or scroll state.

```go
for {
	now := time.Now()
	gadget.Update(now)
	if err := gadget.Render(); err != nil {
		panic(err)
	}
	time.Sleep(16 * time.Millisecond)
}
```

## Cardputer ADV adapter

The repository adapter is `internal/keyboard/cardputeradv`. Repository examples
can use it, but external modules cannot import it while it remains internal.

```go
if err := machine.I2C0.Configure(machine.I2CConfig{
	Frequency: 400_000,
	SDA:       machine.GPIO8,
	SCL:       machine.GPIO9,
}); err != nil {
	panic(err)
}
keyboard := cardputeradv.New(machine.I2C0)
if err := keyboard.Configure(); err != nil {
	panic(err)
}
```

The adapter talks to the TCA8418 at I2C address `0x34`. The controller provides
press/release edges through a ten-event FIFO. The adapter additionally tracks
state, so a held key does not generate repeated `KeyDown` events.

Hardware events are drained in FIFO order into a fixed 32-event ring. Event
delivery allocates no heap memory. When the ring is full, the newest event is
dropped and `DroppedEvents` increases, preserving existing order. On hardware
FIFO overflow, the local queue, pressed keys, saved logical Codes, and all
modifiers are cleared because a release may have been lost. No synthetic KeyUp
is generated. The counter increases by one because the controller cannot report
the exact number lost. Polling I2C errors are available from `Err`.

`Configure` also clears the local queue, pressed state, saved Codes, modifiers,
dropped counter, and previous polling error before draining the hardware FIFO.

The TCA8418 CFG value enables key-event interrupts (bit 0) and overflow
interrupts (bit 3). Bit 5 is not an interrupt mask: it selects whether new
overflow data pushes old FIFO data out. It remains disabled, matching the
official Cardputer driver behavior; either overflow mode requires software
state recovery after event loss.

### Official three-layer map

The adapter follows M5Stack's
[`M5Cardputer` Keyboard map](https://github.com/m5stack/M5Cardputer/blob/master/src/utility/Keyboard/Keyboard.h).
Aa changes printable Rune while Fn selects another logical Code. When a key has
no dedicated Fn mapping, it still produces its base Code with `ModFn` and a zero
Rune. TCA8418 register meanings follow the
[Texas Instruments TCA8418 datasheet](https://www.ti.com/lit/ds/symlink/tca8418.pdf).

| Physical row | Normal | Aa/Shift | Fn |
| --- | --- | --- | --- |
| Top | `` ` 1 2 3 4 5 6 7 8 9 0 - = Backspace `` | `~ ! @ # $ % ^ & * ( ) _ + Backspace` | `Escape F1 F2 F3 F4 F5 F6 F7 F8 F9 F10 F11 F12 Delete` |
| QWERTY | `Tab q w e r t y u i o p [ ] \` | `Tab Q W E R T Y U I O P { } \|` | Base Code with Rune zero |
| Home | `Fn Aa a s d f g h j k l ; ' Enter` | `Fn Aa A S D F G H J K L : " Enter` | Base Code with Rune zero; semicolon is ArrowUp |
| Bottom | `Control Opt Alt z x c v b n m , . / Space` | `Control Opt Alt Z X C V B N M < > ? Space` | Base Code with Rune zero; comma/period/slash are arrows |

Examples:

- Aa+1: `Code=Key1`, `Rune='!'`, `Modifiers=ModShift`.
- Fn+1: `Code=KeyF1`, `Rune=0`, `Modifiers=ModFn`.
- Fn+semicolon/comma/period/slash produces ArrowUp/Left/Down/Right.
- Fn+Backspace produces `KeyDelete`.
- Fn+M produces `Code=KeyM`, `Rune=0`, `Modifiers=ModFn`.

### Standard volume keys

When a `VolumeController` is supplied to `Gadget`, these shortcuts are handled
before application listeners:

```text
Fn + =    Volume Up
Fn + -    Volume Down
Fn + M    Mute / Unmute
```

Volume Up and Down stop at HIGH and MUTE rather than wrapping. Unmute restores
the level active before mute. A handled KeyDown performs the operation, and
both that KeyDown and its corresponding KeyUp are consumed instead of being
delivered to `OnKey`. The captured KeyUp remains consumed if Fn is released
first. Without a controller, both events are delivered normally. Other Fn
combinations remain application events.

```text
Volume Up:   MUTE -> LOW -> MEDIUM -> HIGH
Volume Down: HIGH -> MEDIUM -> LOW -> MUTE
```

The logical Code selected at KeyDown is saved per physical key. KeyUp reuses
that Code even if Fn or Aa was released first. Every KeyUp has Rune zero.

Modifier events update state before emission: modifier KeyDown contains its
bit, while modifier KeyUp is emitted after its bit is cleared. Non-modifier
events contain the modifier state active at that event. Cardputer Opt maps to
`ModMeta`; Control, Alt, or Meta makes a printable event's Rune zero.

## Typing-game example

See [`examples/keyboard-typing/main.go`](../examples/keyboard-typing/main.go).
It displays `TinyGo Makes Small Devices Fun!`, compares printable KeyDown runes,
counts misses, accepts Backspace to move back, and uses Enter to restart. Capital
letters and the exclamation mark exercise Aa/Shift. The target and typed text
use the same explicit two-line break so neither is horizontally clipped.

## Current limitations and future separation

- The Cardputer adapter and bundled display/font drivers are internal.
- There is no automatic repeat, configurable keymap, focus, bubbling, capture,
  shortcut manager, Text mode, composition, or IME.
- The adapter relies on TCA8418 edge handling instead of adding a debounce timer.

```text
Keyboard hardware
    ↓
KeyEvent
    ├── OnKey
    └── future Input Method
            ↓
        future TextEvent
```

The future concepts above are not current APIs. An IME committing multiple
characters should use a separate event, not extend `KeyEvent.Rune`.
