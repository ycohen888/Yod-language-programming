//go:build windows

package editor

import (
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

const (
	bifReturnOnlyFSDirs = 0x00000001
	bifNewDialogStyle   = 0x00000040
	bffmInitialized     = 1
	bffmSetSelectionW   = win.WM_USER + 103
)

// pickFolder פותח דיאלוג בחירת תיקייה (מודרני).
// לא משתמש ב־walk.ShowBrowseFolder — שם יש באג OleUninitialize + PidlRoot ששובר את הבחירה.
func pickFolder(owner walk.Form, title, initial string) (string, bool, error) {
	hr := win.OleInitialize()
	weInitialized := hr == win.S_OK
	if hr != win.S_OK && hr != win.S_FALSE {
		return "", false, syscall.Errno(hr)
	}
	if weInitialized {
		defer win.OleUninitialize()
	}

	var ownerHwnd win.HWND
	if owner != nil {
		ownerHwnd = owner.Handle()
	}

	var initialUTF16 *uint16
	if initial != "" {
		initialUTF16 = syscall.StringToUTF16Ptr(initial)
	}

	cb := syscall.NewCallback(func(hwnd win.HWND, msg uint32, lp, wp uintptr) uintptr {
		if msg == bffmInitialized && initialUTF16 != nil {
			win.SendMessage(hwnd, bffmSetSelectionW, 1, uintptr(unsafe.Pointer(initialUTF16)))
		}
		return 0
	})

	bi := win.BROWSEINFO{
		HwndOwner: ownerHwnd,
		LpszTitle: syscall.StringToUTF16Ptr(title),
		UlFlags:   bifReturnOnlyFSDirs | bifNewDialogStyle,
		Lpfn:      cb,
		// לא מגדירים PidlRoot — זה מגביל את העץ ושובר בחירה
	}

	pidl := win.SHBrowseForFolder(&bi)
	if pidl == 0 {
		return "", false, nil
	}
	defer win.CoTaskMemFree(pidl)

	var path [win.MAX_PATH]uint16
	if !win.SHGetPathFromIDList(pidl, &path[0]) {
		return "", false, nil
	}
	s := syscall.UTF16ToString(path[:])
	if s == "" {
		return "", false, nil
	}
	return s, true, nil
}
