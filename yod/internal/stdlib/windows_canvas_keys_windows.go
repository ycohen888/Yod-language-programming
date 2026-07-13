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
	name := ""
	switch key {
	case walk.KeyBack:
		name = "מחיקה"
	case walk.KeyDelete:
		name = "מחק"
	case walk.KeyReturn:
		name = "אנטר"
	case walk.KeyEscape:
		name = "בריחה"
	case walk.KeyTab:
		name = "טאב"
	default:
		return
	}
	if ch.onKeyCmd == nil {
		return
	}
	invokeYod(ch.onKeyCmd, []object.Object{&object.String{Value: name}})
}

func handleSurfaceKeyPress(ch *controlState, key walk.Key) {
	switch key {
	case walk.KeyBack, walk.KeyDelete, walk.KeyReturn, walk.KeyEscape, walk.KeyTab,
		walk.KeyLeft, walk.KeyRight, walk.KeyUp, walk.KeyDown,
		walk.KeyHome, walk.KeyEnd, walk.KeyPrior, walk.KeyNext,
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

func unicodeFromVirtualKey(key walk.Key) string {
	var state [256]byte
	r1, _, _ := procGetKeyboardState.Call(uintptr(unsafe.Pointer(&state[0])))
	if r1 == 0 {
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
