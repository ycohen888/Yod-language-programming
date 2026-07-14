package stdlib

import (
	"strings"

	"yod/internal/object"
)

// NewKeyboardModule — סקירת מצב מקלדת (כמו עכבר).
func NewKeyboardModule() *object.Module {
	m := &object.Module{Name: "מקלדת", Attrs: map[string]object.Object{}}
	m.Attrs["לחוץ"] = &object.Builtin{Fn: keyboardPressed}
	m.Attrs["מצב"] = &object.Builtin{Fn: keyboardState}
	return m
}

// keyboardNameToVK — מיפוי שמות יוד ל־Virtual-Key.
func keyboardNameToVK(name string) (int, bool) {
	n := strings.TrimSpace(name)
	n = strings.ReplaceAll(n, " ", "")
	if n == "" {
		return 0, false
	}
	if len([]rune(n)) == 1 {
		r := []rune(n)[0]
		if r >= 'a' && r <= 'z' {
			return int(r - 'a' + 0x41), true
		}
		if r >= 'A' && r <= 'Z' {
			return int(r), true
		}
		if r >= '0' && r <= '9' {
			return int(r), true
		}
	}
	switch strings.ToLower(n) {
	case "שמאלה", "חץ_שמאל", "left":
		return 0x25, true
	case "ימינה", "חץ_ימין", "right":
		return 0x27, true
	case "למעלה", "חץ_למעלה", "up":
		return 0x26, true
	case "למטה", "חץ_למטה", "down":
		return 0x28, true
	case "רווח", "space":
		return 0x20, true
	case "אנטר", "enter", "return":
		return 0x0D, true
	case "בריחה", "escape", "esc":
		return 0x1B, true
	case "טאב", "tab":
		return 0x09, true
	case "shift", "שיפט":
		return 0x10, true
	case "ctrl", "control", "קונטרול":
		return 0x11, true
	case "alt":
		return 0x12, true
	case "מחיקה", "backspace", "בקספייס":
		return 0x08, true
	case "מחק", "delete", "del":
		return 0x2E, true
	case "התחלה", "home":
		return 0x24, true
	case "סוף", "end":
		return 0x23, true
	case "pageup", "עמוד_למעלה":
		return 0x21, true
	case "pagedown", "עמוד_למטה":
		return 0x22, true
	}
	return 0, false
}

var keyboardWatchList = []string{
	"שמאלה", "ימינה", "למעלה", "למטה",
	"רווח", "אנטר", "בריחה", "טאב",
	"Shift", "Ctrl", "Alt",
	"W", "A", "S", "D",
	"P", "R",
}
