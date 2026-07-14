//go:build windows

package stdlib

import (
	"golang.org/x/sys/windows"

	"yod/internal/object"
)

var (
	user32Kbd               = windows.NewLazySystemDLL("user32.dll")
	procGetAsyncKeyStateKbd = user32Kbd.NewProc("GetAsyncKeyState")
)

func keyboardKeyDown(vk int) bool {
	r, _, _ := procGetAsyncKeyStateKbd.Call(uintptr(vk))
	return r&0x8000 != 0
}

func keyboardPressed(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מקלדת.לחוץ מצפה לשם מקש אחד")
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("מקלדת.לחוץ מצפה למחרוזת שם מקש")
	}
	vk, ok := keyboardNameToVK(name)
	if !ok {
		return errObj("מקש לא מוכר: " + name)
	}
	return &object.Boolean{Value: keyboardKeyDown(vk)}
}

func keyboardState(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מקלדת.מצב מצפה ל־0 ארגומנטים")
	}
	h := object.NewHash()
	pressed := []object.Object{}
	for _, name := range keyboardWatchList {
		vk, ok := keyboardNameToVK(name)
		if !ok {
			continue
		}
		down := keyboardKeyDown(vk)
		h.Set(name, &object.Boolean{Value: down})
		if down {
			pressed = append(pressed, &object.String{Value: name})
		}
	}
	h.Set("לחוצים", &object.Array{Elements: pressed})
	return h
}
