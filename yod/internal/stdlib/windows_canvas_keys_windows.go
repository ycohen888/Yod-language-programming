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
	procMapVirtualKey    = user32.NewProc("MapVirtualKeyW")
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
	case walk.KeyLeft:
		name = "שמאלה"
	case walk.KeyRight:
		name = "ימינה"
	case walk.KeyUp:
		name = "למעלה"
	case walk.KeyDown:
		name = "למטה"
	case walk.KeyPrior:
		name = "עמוד למעלה"
	case walk.KeyNext:
		name = "עמוד למטה"
	case walk.KeySpace:
		name = "רווח"
	case walk.KeyOEMPeriod:
		name = "."
	case walk.KeyF1:
		name = "F1"
	case walk.KeyF5:
		name = "F5"
	case walk.KeyF9:
		name = "F9"
	case walk.Key1:
		name = "1"
	case walk.Key2:
		name = "2"
	case walk.Key3:
		name = "3"
	case walk.KeyP:
		name = "P"
	case walk.KeyR:
		name = "R"
	case walk.KeyT:
		name = "T"
	case walk.KeyB:
		name = "B"
	case walk.KeyC:
		name = "C"
	case walk.KeyI:
		name = "I"
	case walk.KeyH:
		name = "H"
	case walk.KeyN:
		name = "N"
	case walk.KeyW:
		name = "W"
	case walk.KeyA:
		name = "A"
	case walk.KeyS:
		name = "S"
	case walk.KeyD:
		name = "D"
	default:
		return
	}
	invokeKeyCmd(ch, name)
}

func handleSurfaceKeyPress(ch *controlState, key walk.Key) {
	// KeyPress של Walk מגיע מ־WM_KEYDOWN עם VK — לא עם תו ממופה.
	// כשיש subclass ל־WM_CHAR — התווים מגיעים משם; אחרת גיבוי ToUnicode.
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
	case walk.KeyLeft, walk.KeyRight, walk.KeyUp, walk.KeyDown,
		walk.KeyReturn, walk.KeyEscape, walk.KeyTab,
		walk.KeyHome, walk.KeyEnd, walk.KeyPrior, walk.KeyNext,
		walk.KeyF1, walk.KeyF5, walk.KeyF9,
		walk.KeyShift, walk.KeyControl, walk.KeyAlt, walk.KeyCapital:
		return
	}

	if ch != nil && ch.canvasKeysWired {
		// WM_CHAR ב־enableSurfaceArrowKeys מטפל בתווים (כולל עברית)
		return
	}
	if ch == nil || ch.onKeyChar == nil {
		return
	}
	s := unicodeFromVirtualKey(key)
	if s == "" {
		return
	}
	invokeYod(ch.onKeyChar, []object.Object{&object.String{Value: s}})
}

// handleSurfaceChar — תו מ־WM_CHAR (Unicode אחרי TranslateMessage).
func handleSurfaceChar(ch *controlState, r rune) {
	if ch == nil || ch.onKeyChar == nil {
		return
	}
	if walk.ModifiersDown()&walk.ModControl != 0 {
		return
	}
	if walk.ModifiersDown()&walk.ModAlt != 0 {
		return
	}
	// תווי בקרה (מחיקה, טאב, אנטר…) — לא להכניס לשדה
	if r < 32 || r == 127 {
		return
	}
	invokeYod(ch.onKeyChar, []object.Object{&object.String{Value: string(r)}})
}

func invokeKeyCmd(ch *controlState, name string) {
	if ch == nil || ch.onKeyCmd == nil || name == "" {
		return
	}
	invokeYod(ch.onKeyCmd, []object.Object{&object.String{Value: name}})
}

// unicodeFromVirtualKey — גיבוי נדיר; ההקלדה הרגילה עוברת ב־WM_CHAR.
func unicodeFromVirtualKey(key walk.Key) string {
	var state [256]byte
	r1, _, _ := procGetKeyboardState.Call(uintptr(unsafe.Pointer(&state[0])))
	if r1 == 0 {
		return ""
	}
	if state[0x11]&0x80 != 0 { // VK_CONTROL
		return ""
	}
	scan, _, _ := procMapVirtualKey.Call(uintptr(key), 0) // MAPVK_VK_TO_VSC
	var buf [8]uint16
	n, _, _ := procToUnicode.Call(
		uintptr(key),
		scan,
		uintptr(unsafe.Pointer(&state[0])),
		uintptr(unsafe.Pointer(&buf[0])),
		8,
		0,
	)
	nn := int32(n)
	if nn < 0 {
		// מקש מת — קריאה נוספת מנקה מצב
		n, _, _ = procToUnicode.Call(
			uintptr(key),
			scan,
			uintptr(unsafe.Pointer(&state[0])),
			uintptr(unsafe.Pointer(&buf[0])),
			8,
			0,
		)
		nn = int32(n)
	}
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
