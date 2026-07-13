//go:build windows

package stdlib

import "testing"

func TestWindowsModuleHasCanvasAPIs(t *testing.T) {
	m := NewWindowsModule()
	for _, name := range []string{"שורה", "משטח", "חלון", "כפתור"} {
		if _, ok := m.Attrs[name]; !ok {
			t.Fatalf("חסר במודול חלונות: %q", name)
		}
	}
}
