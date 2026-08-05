package cardputeradv

import (
	"testing"

	"github.com/rdon-key/modgadget"
)

type registerBus struct {
	registers [256]byte
	events    [hardwareFIFOSize]byte
	eventPos  int
	writes    [][2]byte
}

func (bus *registerBus) Tx(_ uint16, write, read []byte) error {
	if len(read) != 0 {
		register := write[0]
		switch register {
		case registerKeyCount:
			remaining := int(bus.registers[registerKeyCount]) - bus.eventPos
			if remaining > 0 {
				read[0] = byte(remaining)
			}
		case registerKeyEvent:
			if bus.eventPos < int(bus.registers[registerKeyCount]) {
				read[0] = bus.events[bus.eventPos]
				bus.eventPos++
			}
		default:
			read[0] = bus.registers[register]
		}
		return nil
	}
	register, value := write[0], write[1]
	bus.writes = append(bus.writes, [2]byte{register, value})
	if register == registerINTStat {
		bus.registers[register] &^= value
	} else {
		bus.registers[register] = value
	}
	return nil
}

func drain(keyboard *Keyboard) []modgadget.KeyEvent {
	events := make([]modgadget.KeyEvent, 0, keyboard.count)
	for keyboard.count > 0 {
		event := keyboard.queue[keyboard.head]
		keyboard.head = (keyboard.head + 1) % eventQueueSize
		keyboard.count--
		events = append(events, event)
	}
	return events
}

func press(keyboard *Keyboard, key byte) modgadget.KeyEvent {
	keyboard.processRaw(0x80 | key)
	events := drain(keyboard)
	return events[len(events)-1]
}

func TestThreeLayerMapping(t *testing.T) {
	tests := []struct {
		name     string
		modifier byte
		key      byte
		code     modgadget.KeyCode
		value    rune
	}{
		{"normal 1", 0, 5, modgadget.Key1, '1'},
		{"Aa 1", 7, 5, modgadget.Key1, '!'},
		{"Fn 1", 3, 5, modgadget.KeyF1, 0},
		{"normal semicolon", 0, 57, modgadget.KeySemicolon, ';'},
		{"Aa semicolon", 7, 57, modgadget.KeySemicolon, ':'},
		{"Fn semicolon", 3, 57, modgadget.KeyArrowUp, 0},
		{"Fn backspace", 3, 65, modgadget.KeyDelete, 0},
		{"normal grave", 0, 1, modgadget.KeyGrave, '`'},
		{"Aa grave", 7, 1, modgadget.KeyGrave, '~'},
		{"Fn grave", 3, 1, modgadget.KeyEscape, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := New(&registerBus{})
			if test.modifier != 0 {
				press(keyboard, test.modifier)
			}
			event := press(keyboard, test.key)
			if event.Code != test.code || event.Rune != test.value || event.Action != modgadget.KeyDown {
				t.Fatalf("event = %#v", event)
			}
		})
	}
}

func TestLayeredDownUpCodeConsistency(t *testing.T) {
	tests := []struct {
		name     string
		key      byte
		wantCode modgadget.KeyCode
	}{
		{"F1", 5, modgadget.KeyF1},
		{"arrow", 57, modgadget.KeyArrowUp},
		{"delete", 65, modgadget.KeyDelete},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := New(&registerBus{})
			press(keyboard, 3)
			down := press(keyboard, test.key)
			keyboard.processRaw(3) // release Fn first
			drain(keyboard)
			keyboard.processRaw(test.key)
			up := drain(keyboard)
			if down.Code != test.wantCode || len(up) != 1 || up[0].Code != down.Code || up[0].Action != modgadget.KeyUp || up[0].Rune != 0 {
				t.Fatalf("down=%#v up=%#v", down, up)
			}
		})
	}

	keyboard := New(&registerBus{})
	press(keyboard, 7)
	down := press(keyboard, 13)
	keyboard.processRaw(7)
	drain(keyboard)
	keyboard.processRaw(13)
	up := drain(keyboard)
	if down.Code != modgadget.KeyA || down.Rune != 'A' || len(up) != 1 || up[0].Code != modgadget.KeyA {
		t.Fatalf("shift code consistency down=%#v up=%#v", down, up)
	}
}

func TestFnUnassignedKeysUseBaseCodeAndZeroRune(t *testing.T) {
	tests := []struct {
		name string
		key  byte
		code modgadget.KeyCode
	}{
		{"M", 48, modgadget.KeyM},
		{"V", 34, modgadget.KeyV},
		{"Q", 6, modgadget.KeyQ},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := New(&registerBus{})
			press(keyboard, 3)
			down := press(keyboard, test.key)
			if down.Code != test.code || down.Rune != 0 || down.Action != modgadget.KeyDown || !down.Modifiers.Has(modgadget.ModFn) {
				t.Fatalf("Fn+%s down=%#v", test.name, down)
			}
			keyboard.processRaw(3) // release Fn before the base key
			drain(keyboard)
			keyboard.processRaw(test.key)
			up := drain(keyboard)
			if len(up) != 1 || up[0].Code != test.code || up[0].Rune != 0 || up[0].Action != modgadget.KeyUp {
				t.Fatalf("Fn+%s up=%#v", test.name, up)
			}
		})
	}
}

func TestFnSystemAndDedicatedMappings(t *testing.T) {
	tests := []struct {
		name string
		key  byte
		code modgadget.KeyCode
	}{
		{"1", 5, modgadget.KeyF1},
		{"minus", 55, modgadget.KeyF11},
		{"equal", 61, modgadget.KeyF12},
		{"semicolon", 57, modgadget.KeyArrowUp},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := New(&registerBus{})
			press(keyboard, 3)
			event := press(keyboard, test.key)
			if event.Code != test.code || event.Rune != 0 || !event.Modifiers.Has(modgadget.ModFn) {
				t.Fatalf("Fn+%s=%#v", test.name, event)
			}
		})
	}

	keyboard := New(&registerBus{})
	event := press(keyboard, 48)
	if event.Code != modgadget.KeyM || event.Rune != 'm' || event.Modifiers != 0 {
		t.Fatalf("normal M=%#v", event)
	}
}

func TestModifierEventStateAndMultipleModifiers(t *testing.T) {
	keyboard := New(&registerBus{})
	tests := []struct {
		key  byte
		code modgadget.KeyCode
		bit  modgadget.Modifiers
	}{
		{7, modgadget.KeyLeftShift, modgadget.ModShift},
		{4, modgadget.KeyLeftControl, modgadget.ModControl},
		{14, modgadget.KeyLeftAlt, modgadget.ModAlt},
		{8, modgadget.KeyLeftMeta, modgadget.ModMeta},
		{3, modgadget.KeyFn, modgadget.ModFn},
	}
	for _, test := range tests {
		down := press(keyboard, test.key)
		if down.Code != test.code || !down.Modifiers.Has(test.bit) {
			t.Fatalf("modifier down = %#v", down)
		}
	}
	if !keyboard.modifiers.Has(modgadget.ModShift | modgadget.ModControl | modgadget.ModAlt | modgadget.ModMeta | modgadget.ModFn) {
		t.Fatalf("combined modifiers = %#x", keyboard.modifiers)
	}
	for _, test := range tests {
		keyboard.processRaw(test.key)
		up := drain(keyboard)
		if len(up) != 1 || up[0].Code != test.code || up[0].Modifiers.Has(test.bit) {
			t.Fatalf("modifier up = %#v", up)
		}
	}
}

func TestShortcutModifiersSuppressRune(t *testing.T) {
	for _, modifier := range []byte{4, 14, 8} {
		keyboard := New(&registerBus{})
		press(keyboard, modifier)
		event := press(keyboard, 28)
		if event.Code != modgadget.KeyC || event.Rune != 0 {
			t.Fatalf("modifier %d C = %#v", modifier, event)
		}
	}
}

func TestPressHoldReleaseAndRepress(t *testing.T) {
	keyboard := New(&registerBus{})
	keyboard.processRaw(0x80 | 13)
	keyboard.processRaw(0x80 | 13)
	keyboard.processRaw(13)
	keyboard.processRaw(13)
	keyboard.processRaw(0x80 | 13)
	events := drain(keyboard)
	if len(events) != 3 || events[0].Action != modgadget.KeyDown || events[1].Action != modgadget.KeyUp || events[2].Action != modgadget.KeyDown {
		t.Fatalf("events = %#v", events)
	}
}

func TestOverflowResetsInputStateAndQueue(t *testing.T) {
	bus := &registerBus{}
	keyboard := New(bus)
	press(keyboard, 3)
	keyboard.processRaw(0x80 | 5)
	if keyboard.count == 0 || keyboard.modifiers == 0 || !keyboard.pressed[5] || keyboard.pressedCodes[5] != modgadget.KeyF1 {
		t.Fatal("test did not establish pre-overflow state")
	}
	bus.registers[registerINTStat] = interruptOverflow
	if err := keyboard.poll(); err != nil {
		t.Fatal(err)
	}
	if keyboard.count != 0 || keyboard.modifiers != 0 || keyboard.pressed[5] || keyboard.pressedCodes[5] != modgadget.KeyUnknown || keyboard.DroppedEvents() != 1 {
		t.Fatalf("state after overflow: count=%d mods=%#x pressed=%v code=%v dropped=%d", keyboard.count, keyboard.modifiers, keyboard.pressed[5], keyboard.pressedCodes[5], keyboard.DroppedEvents())
	}
	event := press(keyboard, 5)
	if event.Code != modgadget.Key1 || event.Rune != '1' {
		t.Fatalf("post-overflow press = %#v", event)
	}
}

func TestConfigureResetsSoftwareAndHardwareState(t *testing.T) {
	bus := &registerBus{}
	bus.registers[registerKeyCount] = 2
	bus.events[0], bus.events[1] = 0x80|13, 13
	keyboard := New(bus)
	keyboard.enqueue(modgadget.KeyEvent{Code: modgadget.KeyA, Action: modgadget.KeyDown})
	keyboard.modifiers = modgadget.ModShift
	keyboard.pressed[13] = true
	keyboard.pressedCodes[13] = modgadget.KeyA
	keyboard.dropped = 7
	keyboard.lastErr = testingError("old")
	if err := keyboard.Configure(); err != nil {
		t.Fatal(err)
	}
	if keyboard.count != 0 || keyboard.head != 0 || keyboard.modifiers != 0 || keyboard.pressed[13] || keyboard.pressedCodes[13] != 0 || keyboard.dropped != 0 || keyboard.lastErr != nil {
		t.Fatalf("state survived Configure: %+v", keyboard)
	}
	if bus.registers[registerCFG] != configKeyEvents || configKeyEvents != 1<<0|1<<3 {
		t.Fatalf("CFG = %#02x", bus.registers[registerCFG])
	}
}

type testingError string

func (err testingError) Error() string { return string(err) }

func TestStableOrderRingWrapAndLocalOverflow(t *testing.T) {
	keyboard := New(&registerBus{})
	for i := 0; i < eventQueueSize-1; i++ {
		keyboard.enqueue(modgadget.KeyEvent{Rune: rune(i)})
	}
	for i := 0; i < eventQueueSize-2; i++ {
		keyboard.head = (keyboard.head + 1) % eventQueueSize
		keyboard.count--
	}
	keyboard.enqueue(modgadget.KeyEvent{Rune: 'x'})
	keyboard.enqueue(modgadget.KeyEvent{Rune: 'y'})
	events := drain(keyboard)
	if len(events) != 3 || events[1].Rune != 'x' || events[2].Rune != 'y' {
		t.Fatalf("wrapped events = %#v", events)
	}
	for i := 0; i < eventQueueSize; i++ {
		keyboard.enqueue(modgadget.KeyEvent{Rune: rune(i)})
	}
	keyboard.enqueue(modgadget.KeyEvent{Rune: 'z'})
	if keyboard.count != eventQueueSize || keyboard.DroppedEvents() != 1 {
		t.Fatalf("count=%d dropped=%d", keyboard.count, keyboard.DroppedEvents())
	}
}

func TestAllPhysicalMappingsAndAllocations(t *testing.T) {
	if len(keyMappings) != 56 {
		t.Fatalf("mapped keys = %d, want 56", len(keyMappings))
	}
	for keyNumber, mapping := range keyMappings {
		if keyNumber == 0 || mapping.code == modgadget.KeyUnknown {
			t.Fatalf("invalid mapping %d: %#v", keyNumber, mapping)
		}
	}
	keyboard := New(&registerBus{})
	if allocs := testing.AllocsPerRun(100, func() {
		keyboard.processRaw(0x80 | 13)
		keyboard.processRaw(13)
		keyboard.head, keyboard.count = 0, 0
	}); allocs != 0 {
		t.Fatalf("event processing allocations = %v", allocs)
	}
}
