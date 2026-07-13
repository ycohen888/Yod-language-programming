//go:build windows

package stdlib

import (
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/walk"

	"yod/internal/object"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procGetKeyboardState = user32.NewProc("GetKeyboardState")
	procToUnicode        = user32.NewProc("ToUnicode")
)

func handleSurfaceKeyDown(ch *controlState, key walk.Key) {
	mods := walk.ModifiersDown()
	if mods&walk.ModControl != 0 {
		name := ""
		switch key {
		case walk.KeyA:
			name = "בחר_הכל"
		case walk.KeyC:
			name = "העתק"
		case walk.KeyV:
			name = "הדבק"
		case walk.KeyX:
			name = "גזור"
		default:
			return
		}
		invokeKeyCmd(ch, name)
		return
	}

	// אנטר / בריחה / טאב — ללא חזרה אוטומטית
	name := ""
	switch key {
	case walk.KeyReturn:
		name = "אנטר"
	case walk.KeyEscape:
		name = "בריחה"
	case walk.KeyTab:
		name = "טאב"
	case walk.KeyHome:
		name = "התחלה"
	case walk.KeyEnd:
		name = "סוף"
	default:
		return
	}
	invokeKeyCmd(ch, name)
}

func handleSurfaceKeyPress(ch *controlState, key walk.Key) {
	// KeyPress חוזר אוטומטית בלחיצה ארוכה — מתאים למחיקה / חצים
	if walk.ModifiersDown()&walk.ModControl != 0 {
		return
	}
	if walk.ModifiersDown()&walk.ModAlt != 0 {
		return
	}

	switch key {
	case walk.KeyBack:
		invokeKeyCmd(ch, "מחיקה")
		return
	case walk.KeyDelete:
		invokeKeyCmd(ch, "מחק")
		return
	case walk.KeyLeft:
		invokeKeyCmd(ch, "שמאלה")
		return
	case walk.KeyRight:
		invokeKeyCmd(ch, "ימינה")
		return
	case walk.KeyReturn, walk.KeyEscape, walk.KeyTab,
		walk.KeyUp, walk.KeyDown, walk.KeyHome, walk.KeyEnd,
		walk.KeyPrior, walk.KeyNext,
		walk.KeyShift, walk.KeyControl, walk.KeyAlt, walk.KeyCapital:
		return
	}

	if ch.onKeyChar == nil {
		return
	}
	s := unicodeFromVirtualKey(key)
	if s == "" {
		return
	}
	invokeYod(ch.onKeyChar, []object.Object{&object.String{Value: s}})
}

func invokeKeyCmd(ch *controlState, name string) {
	if ch == nil || ch.onKeyCmd == nil || name == "" {
		return
	}
	invokeYod(ch.onKeyCmd, []object.Object{&object.String{Value: name}})
}

func unicodeFromVirtualKey(key walk.Key) string {
	var state [256]byte
	r1, _, _ := procGetKeyboardState.Call(uintptr(unsafe.Pointer(&state[0])))
	if r1 == 0 {
		return ""
	}
	// עם Ctrl — לא לייצר תו (קיצורי דרך מטופלים ב־KeyDown)
	if state[0x11]&0x80 != 0 { // VK_CONTROL
		return ""
	}
	var buf [8]uint16
	n, _, _ := procToUnicode.Call(
		uintptr(key),
		0,
		uintptr(unsafe.Pointer(&state[0])),
		uintptr(unsafe.Pointer(&buf[0])),
		8,
		0,
	)
	nn := int32(n)
	if nn <= 0 {
		return ""
	}
	runes := utf16.Decode(buf[:nn])
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r >= 32 && r != 127 {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return string(out)
}
