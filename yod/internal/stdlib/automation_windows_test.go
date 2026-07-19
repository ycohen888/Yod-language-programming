//go:build windows

package stdlib

import "testing"

func TestAutomationKeyVK(t *testing.T) {
	cases := map[string]uint16{
		"שמאלה": 0x25,
		"ימינה": 0x27,
		"מחק":   0x2E,
		"בקרה":  0x11,
		"F5":    0x74,
		"a":     'A',
	}
	for name, want := range cases {
		got, ok := automationKeyVK(name)
		if !ok {
			t.Fatalf("automationKeyVK(%q) not found", name)
		}
		if got != want {
			t.Fatalf("automationKeyVK(%q)=%#x want %#x", name, got, want)
		}
	}
}
