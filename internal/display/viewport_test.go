package display

import (
	"math"
	"strings"
	"testing"
)

func TestViewportZeroValue(t *testing.T) {
	var viewport Viewport
	if viewport.Bounds() != (Rect{}) {
		t.Fatalf("bounds=%+v", viewport.Bounds())
	}
	result, visible, err := viewport.Clip(Rect{Width: 1, Height: 1})
	if err != nil || visible || result != (ClipResult{}) {
		t.Fatalf("result=%+v visible=%v err=%v", result, visible, err)
	}
}

func TestNewViewportAndBoundsValue(t *testing.T) {
	want := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	viewport, err := NewViewport(want)
	if err != nil || viewport.Bounds() != want {
		t.Fatalf("bounds=%+v err=%v", viewport.Bounds(), err)
	}
	copy := viewport.Bounds()
	copy.X = 99
	if viewport.Bounds() != want {
		t.Fatalf("bounds changed to %+v", viewport.Bounds())
	}
}

func TestViewportClip(t *testing.T) {
	viewport, err := NewViewport(Rect{X: 10, Y: 20, Width: 100, Height: 50})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		local   Rect
		want    ClipResult
		visible bool
	}{
		{"fully visible and translated", Rect{X: 2, Y: 3, Width: 8, Height: 6}, ClipResult{Destination: Rect{X: 12, Y: 23, Width: 8, Height: 6}}, true},
		{"left", Rect{X: -3, Y: 4, Width: 10, Height: 8}, ClipResult{Destination: Rect{X: 10, Y: 24, Width: 7, Height: 8}, SourceX: 3}, true},
		{"top", Rect{X: 4, Y: -3, Width: 8, Height: 10}, ClipResult{Destination: Rect{X: 14, Y: 20, Width: 8, Height: 7}, SourceY: 3}, true},
		{"right", Rect{X: 95, Y: 4, Width: 10, Height: 8}, ClipResult{Destination: Rect{X: 105, Y: 24, Width: 5, Height: 8}}, true},
		{"bottom", Rect{X: 4, Y: 45, Width: 8, Height: 10}, ClipResult{Destination: Rect{X: 14, Y: 65, Width: 8, Height: 5}}, true},
		{"left top", Rect{X: -3, Y: -4, Width: 10, Height: 10}, ClipResult{Destination: Rect{X: 10, Y: 20, Width: 7, Height: 6}, SourceX: 3, SourceY: 4}, true},
		{"right bottom", Rect{X: 95, Y: 45, Width: 10, Height: 10}, ClipResult{Destination: Rect{X: 105, Y: 65, Width: 5, Height: 5}}, true},
		{"all four sides", Rect{X: -10, Y: -10, Width: 120, Height: 70}, ClipResult{Destination: Rect{X: 10, Y: 20, Width: 100, Height: 50}, SourceX: 10, SourceY: 10}, true},
		{"outside left", Rect{X: -20, Width: 10, Height: 5}, ClipResult{}, false},
		{"outside right", Rect{X: 110, Width: 10, Height: 5}, ClipResult{}, false},
		{"outside top", Rect{Y: -10, Width: 5, Height: 5}, ClipResult{}, false},
		{"outside bottom", Rect{Y: 50, Width: 5, Height: 5}, ClipResult{}, false},
		{"touch right", Rect{X: 100, Width: 5, Height: 5}, ClipResult{}, false},
		{"touch bottom", Rect{Y: 50, Width: 5, Height: 5}, ClipResult{}, false},
		{"zero local width", Rect{Width: 0, Height: 5}, ClipResult{}, false},
		{"zero local height", Rect{Width: 5, Height: 0}, ClipResult{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.local
			got, visible, err := viewport.Clip(input)
			if err != nil || visible != tt.visible || got != tt.want {
				t.Fatalf("result=%+v visible=%v want=%+v/%v err=%v", got, visible, tt.want, tt.visible, err)
			}
			if input != tt.local {
				t.Fatalf("input changed from %+v to %+v", tt.local, input)
			}
		})
	}
}

func TestViewportClipEmptyViewport(t *testing.T) {
	for _, bounds := range []Rect{{X: 3, Y: 4, Width: 0, Height: 5}, {X: 3, Y: 4, Width: 5, Height: 0}} {
		viewport, err := NewViewport(bounds)
		if err != nil {
			t.Fatal(err)
		}
		result, visible, err := viewport.Clip(Rect{Width: 2, Height: 2})
		if err != nil || visible || result != (ClipResult{}) {
			t.Fatalf("bounds=%+v result=%+v visible=%v err=%v", bounds, result, visible, err)
		}
	}
}

func TestViewportClipNegativeCoordinatesAndInt16Boundaries(t *testing.T) {
	viewport, err := NewViewport(Rect{X: math.MinInt16, Y: math.MinInt16, Width: 20, Height: 20})
	if err != nil {
		t.Fatal(err)
	}
	got, visible, err := viewport.Clip(Rect{X: -5, Y: -6, Width: 10, Height: 10})
	want := ClipResult{Destination: Rect{X: math.MinInt16, Y: math.MinInt16, Width: 5, Height: 4}, SourceX: 5, SourceY: 6}
	if err != nil || !visible || got != want {
		t.Fatalf("result=%+v visible=%v want=%+v err=%v", got, visible, want, err)
	}

	viewport, err = NewViewport(Rect{X: math.MaxInt16 - 1, Y: math.MaxInt16 - 1, Width: 2, Height: 2})
	if err != nil {
		t.Fatal(err)
	}
	got, visible, err = viewport.Clip(Rect{Width: 1, Height: 1})
	want = ClipResult{Destination: Rect{X: math.MaxInt16 - 1, Y: math.MaxInt16 - 1, Width: 1, Height: 1}}
	if err != nil || !visible || got != want {
		t.Fatalf("result=%+v visible=%v want=%+v err=%v", got, visible, want, err)
	}
}

func TestViewportValidation(t *testing.T) {
	constructorTests := []struct {
		name   string
		bounds Rect
		text   string
	}{
		{"negative viewport width", Rect{Width: -1}, "viewport width"},
		{"negative viewport height", Rect{Height: -1}, "viewport height"},
	}
	for _, tt := range constructorTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewViewport(tt.bounds)
			if err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("err=%v", err)
			}
		})
	}

	viewport, err := NewViewport(Rect{Width: 10, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name  string
		local Rect
		text  string
	}{
		{"negative local width", Rect{Width: -1}, "local rectangle width"},
		{"negative local height", Rect{Height: -1}, "local rectangle height"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := viewport.Clip(tt.local)
			if err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestViewportClipResultConversionOverflow(t *testing.T) {
	viewport, err := NewViewport(Rect{X: math.MaxInt16, Y: math.MaxInt16, Width: 2, Height: 2})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		local Rect
		text  string
	}{
		{"destination X", Rect{X: 1, Width: 1, Height: 1}, "destination X"},
		{"destination Y", Rect{Y: 1, Width: 1, Height: 1}, "destination Y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, visible, err := viewport.Clip(tt.local)
			if err == nil || visible || result != (ClipResult{}) || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("result=%+v visible=%v err=%v", result, visible, err)
			}
		})
	}

	result, visible, err := viewport.Clip(Rect{X: math.MinInt16, Y: math.MinInt16, Width: 1, Height: 1})
	if err != nil || visible || result != (ClipResult{}) {
		t.Fatalf("far outside result=%+v visible=%v err=%v", result, visible, err)
	}
}
