package modgadget

import (
	"testing"
	"time"
)

type fakeKeyboard struct {
	events [8]KeyEvent
	count  int
	index  int
	reads  int
}

type fakeVolumeController struct {
	ups, downs, toggles int
}

func (controller *fakeVolumeController) VolumeUp()   { controller.ups++ }
func (controller *fakeVolumeController) VolumeDown() { controller.downs++ }
func (controller *fakeVolumeController) ToggleMute() { controller.toggles++ }

var _ VolumeController = (*fakeVolumeController)(nil)

var _ Keyboard = (*fakeKeyboard)(nil)

func (keyboard *fakeKeyboard) ReadKeyEvent() (KeyEvent, bool) {
	keyboard.reads++
	if keyboard.index >= keyboard.count {
		return KeyEvent{}, false
	}
	event := keyboard.events[keyboard.index]
	keyboard.index++
	return event, true
}

func TestKeyboardOptionalAndDrain(t *testing.T) {
	if (KeyEvent{}).Action != KeyActionUnknown || KeyActionUnknown == KeyDown {
		t.Fatal("zero KeyEvent has a valid KeyDown action")
	}
	New(nil).Update(time.Time{})
	New(nil, WithKeyboard(nil)).Update(time.Time{})

	keyboard := &fakeKeyboard{count: 2}
	keyboard.events[0] = KeyEvent{Code: KeyA, Rune: 'A', Action: KeyDown, Modifiers: ModShift}
	keyboard.events[1] = KeyEvent{Code: KeyA, Action: KeyUp}
	g := New(nil, WithKeyboard(keyboard))
	var got [2]KeyEvent
	count := 0
	g.OnKey(func(event KeyEvent) bool {
		got[count] = event
		count++
		return false
	})
	g.Update(time.Time{})
	if count != 2 || got[0] != keyboard.events[0] || got[1] != keyboard.events[1] {
		t.Fatalf("events = %#v count=%d", got, count)
	}
	if keyboard.reads != 3 {
		t.Fatalf("reads = %d, want two events plus empty read", keyboard.reads)
	}
	if !got[0].Modifiers.Has(ModShift) || got[0].Modifiers.Has(ModControl) {
		t.Fatalf("modifiers = %#x", got[0].Modifiers)
	}
	if !(ModShift | ModControl).Has(ModShift | ModControl) {
		t.Fatal("Has did not recognize multiple bits")
	}
}

func TestSystemVolumeShortcutsConsumeBeforeHandlers(t *testing.T) {
	controller := &fakeVolumeController{}
	keyboard := &fakeKeyboard{}
	keyboard.events = [8]KeyEvent{}
	// fakeKeyboard has room for eight events; exercise the three shortcuts,
	// releases, missing Fn, and an unrelated Fn key in those slots.
	keyboard.count = 8
	keyboard.events[0] = KeyEvent{Code: KeyF12, Action: KeyDown, Modifiers: ModFn}
	keyboard.events[1] = KeyEvent{Code: KeyF11, Action: KeyDown, Modifiers: ModFn | ModShift}
	keyboard.events[2] = KeyEvent{Code: KeyM, Action: KeyDown, Modifiers: ModFn | ModControl}
	keyboard.events[3] = KeyEvent{Code: KeyF12, Action: KeyUp, Modifiers: ModFn}
	keyboard.events[4] = KeyEvent{Code: KeyF11, Action: KeyDown}
	keyboard.events[5] = KeyEvent{Code: KeyM, Action: KeyUp, Modifiers: ModFn}
	keyboard.events[6] = KeyEvent{Code: KeyQ, Action: KeyDown, Modifiers: ModFn}
	keyboard.events[7] = KeyEvent{Code: KeyM, Action: KeyDown}

	g := New(nil, WithKeyboard(keyboard), WithVolumeController(controller))
	delivered := make([]KeyEvent, 0, 5)
	g.OnKey(func(event KeyEvent) bool {
		delivered = append(delivered, event)
		return false
	})
	g.Update(time.Time{})
	if controller.ups != 1 || controller.downs != 1 || controller.toggles != 1 {
		t.Fatalf("volume calls up=%d down=%d toggle=%d", controller.ups, controller.downs, controller.toggles)
	}
	if len(delivered) != 5 {
		t.Fatalf("delivered=%#v", delivered)
	}
	if delivered[3].Code != KeyQ || !delivered[3].Modifiers.Has(ModFn) {
		t.Fatalf("unrelated Fn event=%#v", delivered[3])
	}
}

func TestSystemVolumeShortcutsWithoutControllerAreDelivered(t *testing.T) {
	keyboard := &fakeKeyboard{count: 3}
	keyboard.events[0] = KeyEvent{Code: KeyF12, Action: KeyDown, Modifiers: ModFn}
	keyboard.events[1] = KeyEvent{Code: KeyF11, Action: KeyDown, Modifiers: ModFn}
	keyboard.events[2] = KeyEvent{Code: KeyM, Action: KeyDown, Modifiers: ModFn}
	g := New(nil, WithKeyboard(keyboard), WithVolumeController(nil))
	delivered := 0
	g.OnKey(func(KeyEvent) bool { delivered++; return false })
	g.Update(time.Time{})
	if delivered != 3 {
		t.Fatalf("delivered=%d want=3", delivered)
	}
}

func TestSystemVolumeShortcutAllocations(t *testing.T) {
	keyboard := &fakeKeyboard{}
	controller := &fakeVolumeController{}
	g := New(nil, WithKeyboard(keyboard), WithVolumeController(controller))
	if allocs := testing.AllocsPerRun(100, func() {
		keyboard.index, keyboard.count = 0, 1
		keyboard.events[0] = KeyEvent{Code: KeyF12, Action: KeyDown, Modifiers: ModFn}
		g.Update(time.Time{})
	}); allocs != 0 {
		t.Fatalf("system shortcut allocations=%v", allocs)
	}
}

type infiniteVolumeKeyboard struct{ reads int }

func (keyboard *infiniteVolumeKeyboard) ReadKeyEvent() (KeyEvent, bool) {
	keyboard.reads++
	return KeyEvent{Code: KeyF12, Action: KeyDown, Modifiers: ModFn}, true
}

func TestSystemVolumeShortcutKeepsKeyboardEventLimit(t *testing.T) {
	keyboard := &infiniteVolumeKeyboard{}
	controller := &fakeVolumeController{}
	g := New(nil, WithKeyboard(keyboard), WithVolumeController(controller))
	g.Update(time.Time{})
	if keyboard.reads != maxKeyEventsPerUpdate || controller.ups != maxKeyEventsPerUpdate || controller.downs != 0 || controller.toggles != 0 {
		t.Fatalf("reads=%d up=%d down=%d toggle=%d limit=%d",
			keyboard.reads, controller.ups, controller.downs, controller.toggles, maxKeyEventsPerUpdate)
	}
}

func TestKeyHandlerOrderConsumeAndRemoval(t *testing.T) {
	g := New(nil)
	if id := g.OnKey(nil); id != 0 {
		t.Fatalf("nil handler ID = %d", id)
	}
	if id := (*Gadget)(nil).OnKey(func(KeyEvent) bool { return false }); id != 0 {
		t.Fatalf("nil Gadget ID = %d", id)
	}
	var order [3]int
	called := 0
	id1 := g.OnKey(func(KeyEvent) bool { order[called] = 1; called++; return false })
	id2 := g.OnKey(func(KeyEvent) bool { order[called] = 2; called++; return true })
	id3 := g.OnKey(func(KeyEvent) bool { order[called] = 3; called++; return false })
	if id1 == 0 || id2 == 0 || id3 == 0 || id1 == id2 || id2 == id3 || id1 == id3 {
		t.Fatalf("listener IDs = %d, %d, %d", id1, id2, id3)
	}
	g.dispatchKey(KeyEvent{Action: KeyDown})
	if called != 2 || order != [3]int{1, 2, 0} {
		t.Fatalf("dispatch order = %v called=%d", order, called)
	}
	if !g.RemoveListener(id2) || g.RemoveListener(id2) || g.RemoveListener(0) || g.RemoveListener(ListenerID(60000)) {
		t.Fatal("unexpected RemoveListener result")
	}
	if (*Gadget)(nil).RemoveListener(id1) {
		t.Fatal("nil Gadget removed listener")
	}
	called, order = 0, [3]int{}
	g.dispatchKey(KeyEvent{Action: KeyDown})
	if called != 2 || order != [3]int{1, 3, 0} {
		t.Fatalf("after removal order = %v called=%d", order, called)
	}
}

func TestKeyDispatchMutationRules(t *testing.T) {
	g := New(nil)
	var calls [4]int
	var self ListenerID
	var later ListenerID
	self = g.OnKey(func(KeyEvent) bool {
		calls[0]++
		g.RemoveListener(self)
		g.RemoveListener(later)
		g.OnKey(func(KeyEvent) bool { calls[3]++; return false })
		return false
	})
	g.OnKey(func(KeyEvent) bool { calls[1]++; return false })
	later = g.OnKey(func(KeyEvent) bool { calls[2]++; return false })
	g.dispatchKey(KeyEvent{Action: KeyDown})
	if calls != [4]int{1, 1, 0, 0} {
		t.Fatalf("first event calls = %v", calls)
	}
	if len(g.keyListeners) != 2 {
		t.Fatalf("listeners after dispatch compact = %d", len(g.keyListeners))
	}
	g.dispatchKey(KeyEvent{Action: KeyDown})
	if calls != [4]int{1, 2, 0, 1} {
		t.Fatalf("second event calls = %v", calls)
	}
}

func TestListenerCompactionDoesNotGrow(t *testing.T) {
	g := New(nil)
	kept := g.OnKey(func(KeyEvent) bool { return false })
	for i := 0; i < 1000; i++ {
		id := g.OnKey(func(KeyEvent) bool { return false })
		if !g.RemoveListener(id) {
			t.Fatalf("remove iteration %d", i)
		}
	}
	if len(g.keyListeners) != 1 || g.keyListeners[0].id != kept {
		t.Fatalf("listeners after churn = %#v", g.keyListeners)
	}
}

type infiniteKeyboard struct{ reads int }

func (keyboard *infiniteKeyboard) ReadKeyEvent() (KeyEvent, bool) {
	keyboard.reads++
	return KeyEvent{Code: KeyA, Rune: 'a', Action: KeyDown}, true
}

func TestKeyboardUpdateEventLimitAndViewportProgress(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	keyboard := &infiniteKeyboard{}
	g := New(&testDisplay{width: 20, height: 10}, WithStyles(testStyles()), WithKeyboard(keyboard))
	v := g.Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(10), ScrollLoop())
	calls := 0
	g.OnKey(func(KeyEvent) bool { calls++; return false })
	g.Update(base)
	g.Update(base.Add(time.Second))
	if calls != 2*maxKeyEventsPerUpdate || keyboard.reads != 2*maxKeyEventsPerUpdate {
		t.Fatalf("calls=%d reads=%d limit=%d", calls, keyboard.reads, maxKeyEventsPerUpdate)
	}
	if v.offset != 10 {
		t.Fatalf("Viewport update did not run, offset=%d", v.offset)
	}
}

func TestKeyboardUpdateKeepsScrollBehavior(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	keyboard := &fakeKeyboard{}
	v := New(&testDisplay{width: 20, height: 10}, WithStyles(testStyles()), WithKeyboard(keyboard)).Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(10), ScrollLoop())
	v.owner.Update(base)
	v.owner.Update(base.Add(time.Second))
	if v.offset != 10 {
		t.Fatalf("scroll offset = %d", v.offset)
	}
}

func TestKeyboardUpdateSteadyAllocations(t *testing.T) {
	keyboard := &fakeKeyboard{}
	g := New(nil, WithKeyboard(keyboard))
	if allocs := testing.AllocsPerRun(100, func() { g.Update(time.Time{}) }); allocs != 0 {
		t.Fatalf("empty Update allocations = %v", allocs)
	}
	keyboard.events[0] = KeyEvent{Code: KeyA, Rune: 'a', Action: KeyDown}
	g.OnKey(func(KeyEvent) bool { return false })
	if allocs := testing.AllocsPerRun(100, func() {
		keyboard.index, keyboard.count = 0, 1
		g.Update(time.Time{})
	}); allocs != 0 {
		t.Fatalf("dispatch allocations = %v", allocs)
	}
}
