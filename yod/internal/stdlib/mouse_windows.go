//go:build windows

package stdlib

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"yod/internal/object"
)

const (
	vkLButton = 0x01
	vkRButton = 0x02
	vkMButton = 0x04
)

var (
	user32Mouse         = windows.NewLazySystemDLL("user32.dll")
	procGetCursorPos    = user32Mouse.NewProc("GetCursorPos")
	procGetAsyncKeyState = user32Mouse.NewProc("GetAsyncKeyState")
)

type mousePoint struct {
	X, Y int32
}

func mouseCursorPos() (int, int, error) {
	var pt mousePoint
	r, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r == 0 {
		return 0, 0, err
	}
	return int(pt.X), int(pt.Y), nil
}

func mouseKeyDown(vk int) bool {
	r, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return r&0x8000 != 0
}

func mousePosition(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("עכבר.מיקום מצפה ל־0 ארגומנטים")
	}
	x, y, err := mouseCursorPos()
	if err != nil {
		return errObj("לא הצלחתי לקרוא מיקום עכבר")
	}
	return mousePosHash(x, y)
}

func mouseLeftDown(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("עכבר.שמאל_לחוץ מצפה ל־0 ארגומנטים")
	}
	return &object.Boolean{Value: mouseKeyDown(vkLButton)}
}

func mouseRightDown(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("עכבר.ימין_לחוץ מצפה ל־0 ארגומנטים")
	}
	return &object.Boolean{Value: mouseKeyDown(vkRButton)}
}

func mouseMiddleDown(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("עכבר.אמצע_לחוץ מצפה ל־0 ארגומנטים")
	}
	return &object.Boolean{Value: mouseKeyDown(vkMButton)}
}

func mouseState(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("עכבר.מצב מצפה ל־0 ארגומנטים")
	}
	x, y, err := mouseCursorPos()
	if err != nil {
		return errObj("לא הצלחתי לקרוא מצב עכבר")
	}
	return mouseStateHash(x, y, mouseKeyDown(vkLButton), mouseKeyDown(vkRButton), mouseKeyDown(vkMButton))
}
