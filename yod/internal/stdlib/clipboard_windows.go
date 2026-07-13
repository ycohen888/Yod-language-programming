//go:build windows

package stdlib

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"yod/internal/object"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32CB   = windows.NewLazySystemDLL("user32.dll")
	kernel32CB = windows.NewLazySystemDLL("kernel32.dll")

	procOpenClipboard    = user32CB.NewProc("OpenClipboard")
	procCloseClipboard   = user32CB.NewProc("CloseClipboard")
	procEmptyClipboard   = user32CB.NewProc("EmptyClipboard")
	procGetClipboardData = user32CB.NewProc("GetClipboardData")
	procSetClipboardData = user32CB.NewProc("SetClipboardData")
	procIsClipboardFmt   = user32CB.NewProc("IsClipboardFormatAvailable")

	procGlobalAlloc  = kernel32CB.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32CB.NewProc("GlobalLock")
	procGlobalUnlock = kernel32CB.NewProc("GlobalUnlock")
	procGlobalFree   = kernel32CB.NewProc("GlobalFree")
)

func clipboardRead(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("לוח.קרא מצפה ל־0 ארגומנטים")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	r, _, err := procOpenClipboard.Call(0)
	if r == 0 {
		return errObj("לוח.קרא: לא הצלחתי לפתוח את הלוח: " + err.Error())
	}
	defer procCloseClipboard.Call()

	avail, _, _ := procIsClipboardFmt.Call(cfUnicodeText)
	if avail == 0 {
		return &object.String{Value: ""}
	}

	h, _, err := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return errObj("לוח.קרא: לא הצלחתי לקרוא מהלוח: " + err.Error())
	}
	ptr, _, err := procGlobalLock.Call(h)
	if ptr == 0 {
		return errObj("לוח.קרא: נעילת זיכרון נכשלה: " + err.Error())
	}
	defer procGlobalUnlock.Call(h)

	text := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr)))
	return &object.String{Value: text}
}

func clipboardWrite(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("לוח.כתוב מצפה למחרוזת אחת")
	}
	text := args[0].Inspect()
	if s, ok := args[0].(*object.String); ok {
		text = s.Value
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return errObj("לוח.כתוב: המרה ליוניקוד נכשלה: " + err.Error())
	}
	byteSize := uintptr(len(utf16) * 2)

	hMem, _, err := procGlobalAlloc.Call(gmemMoveable, byteSize)
	if hMem == 0 {
		return errObj("לוח.כתוב: הקצאת זיכרון נכשלה: " + err.Error())
	}
	ptr, _, err := procGlobalLock.Call(hMem)
	if ptr == 0 {
		procGlobalFree.Call(hMem)
		return errObj("לוח.כתוב: נעילת זיכרון נכשלה: " + err.Error())
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16))
	copy(dst, utf16)
	procGlobalUnlock.Call(hMem)

	r, _, err := procOpenClipboard.Call(0)
	if r == 0 {
		procGlobalFree.Call(hMem)
		return errObj("לוח.כתוב: לא הצלחתי לפתוח את הלוח: " + err.Error())
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	set, _, err := procSetClipboardData.Call(cfUnicodeText, hMem)
	if set == 0 {
		procGlobalFree.Call(hMem)
		return errObj("לוח.כתוב: כתיבה ללוח נכשלה: " + err.Error())
	}
	return object.Nil
}
