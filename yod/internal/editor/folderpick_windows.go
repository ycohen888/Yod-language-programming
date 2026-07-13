//go:build windows

package editor

import (
	"runtime"
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

// נתיב התחלתי לדיאלוג — חייב להיות גלובלי כי NewCallback אוסר closures.
var browseInitialUTF16 *uint16

func browseFolderCallback(hwnd win.HWND, msg uint32, lp, wp uintptr) uintptr {
	if msg == bffmInitialized && browseInitialUTF16 != nil {
		win.SendMessage(hwnd, bffmSetSelectionW, 1, uintptr(unsafe.Pointer(browseInitialUTF16)))
	}
	return 0
}

var browseFolderCallbackPtr uintptr

func init() {
	browseFolderCallbackPtr = syscall.NewCallback(browseFolderCallback)
}

// pickFolder פותח דיאלוג בחירת תיקייה.
// לא קוראים ל־OleUninitialize — זה הורס COM ואז טעינת אייקוני העץ קורסת.
func pickFolder(owner walk.Form, title, initial string) (string, bool, error) {
	hr := win.OleInitialize()
	if hr != win.S_OK && hr != win.S_FALSE {
		return "", false, syscall.Errno(hr)
	}

	var ownerHwnd win.HWND
	if owner != nil {
		ownerHwnd = owner.Handle()
	}

	titleBuf := syscall.StringToUTF16(title)
	var initialBuf []uint16
	browseInitialUTF16 = nil
	if initial != "" {
		initialBuf = syscall.StringToUTF16(initial)
		browseInitialUTF16 = &initialBuf[0]
	}
	defer func() { browseInitialUTF16 = nil }()

	bi := win.BROWSEINFO{
		HwndOwner: ownerHwnd,
		LpszTitle: &titleBuf[0],
		UlFlags:   bifReturnOnlyFSDirs | bifNewDialogStyle,
		Lpfn:      browseFolderCallbackPtr,
	}

	pidl := win.SHBrowseForFolder(&bi)
	runtime.KeepAlive(titleBuf)
	runtime.KeepAlive(initialBuf)
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
