//go:build windows

package stdlib

import (
	"testing"

	"yod/internal/object"
)

func TestChartsModuleAttrs(t *testing.T) {
	m := NewChartsModule()
	for _, name := range []string{"גרף", "עמודות", "קו", "עוגה"} {
		if _, ok := m.Attrs[name]; !ok {
			t.Fatalf("חסר בגרפים: %q", name)
		}
	}
}

func TestNiceCeiling(t *testing.T) {
	cases := map[float64]float64{
		3:    5,
		10:   10,
		12:   20,
		99:   100,
		250:  500,
		1000: 1000,
	}
	for in, want := range cases {
		got := niceCeiling(in)
		if got != want {
			t.Fatalf("niceCeiling(%v)=%v want %v", in, got, want)
		}
	}
}

func TestChartWidgetCreate(t *testing.T) {
	w := newChartWidget("עמודות")
	gw, ok := w.(*object.GuiWidget)
	if !ok {
		t.Fatalf("expected GuiWidget, got %T", w)
	}
	if gw.Kind != "גרף" {
		t.Fatalf("kind=%q", gw.Kind)
	}
	for _, name := range []string{"קבע_כותרת", "קבע_תוויות", "קבע_סדרות", "קבע_נתונים", "קבע_סוג", "רענן"} {
		if _, ok := gw.Attrs[name]; !ok {
			t.Fatalf("חסר מתודה %q", name)
		}
	}
}
