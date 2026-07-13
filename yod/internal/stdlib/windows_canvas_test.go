//go:build windows

package stdlib

import "testing"

func TestWindowsModuleHasCanvasAPIs(t *testing.T) {
	m := NewWindowsModule()
	for _, name := range []string{"שורה", "עמודה", "מסגרת", "משטח", "חלון", "כפתור", "בחר_שמירה", "בחר_פתיחה"} {
		if _, ok := m.Attrs[name]; !ok {
			t.Fatalf("חסר במודול חלונות: %q", name)
		}
	}
}
