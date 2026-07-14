package stdlib

import "testing"

func TestKeyboardNameToVK(t *testing.T) {
	tests := []struct {
		name string
		vk   int
		ok   bool
	}{
		{"שמאלה", 0x25, true},
		{"ימינה", 0x27, true},
		{"למעלה", 0x26, true},
		{"למטה", 0x28, true},
		{"רווח", 0x20, true},
		{"אנטר", 0x0D, true},
		{"A", 0x41, true},
		{"a", 0x41, true},
		{"W", 0x57, true},
		{"5", 0x35, true},
		{"Shift", 0x10, true},
		{"לא_קיים", 0, false},
	}
	for _, tt := range tests {
		vk, ok := keyboardNameToVK(tt.name)
		if ok != tt.ok || (ok && vk != tt.vk) {
			t.Fatalf("%q: got vk=%d ok=%v want vk=%d ok=%v", tt.name, vk, ok, tt.vk, tt.ok)
		}
	}
}
