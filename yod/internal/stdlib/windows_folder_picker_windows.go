//go:build windows

package stdlib

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// בחירת תיקייה מודרנית (IFileOpenDialog + FOS_PICKFOLDERS).
// לא משתמשים ב־walk.ShowBrowseFolder — OleUninitialize שלו שובר COM של החלון,
// ו־SHParseDisplayName על נתיב ריק עלול לגרום לדיאלוג שנכשל בשקט.

const (
	fosPickFolders     = 0x00000020
	fosForceFileSystem = 0x00000040
	fosPathMustExist   = 0x00000800
	sigdnFileSysPath   = 0x80058000
	hresultCancel32    = uint32(0x800704C7)
)

var (
	clsidFileOpenDialog = win.CLSID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = win.IID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
)

type comObject struct {
	lpVtbl *uintptr
}

func (o *comObject) vtable() *[100]uintptr {
	return (*[100]uintptr)(unsafe.Pointer(o.lpVtbl))
}

func (o *comObject) Release() {
	if o == nil || o.lpVtbl == nil {
		return
	}
	syscall.SyscallN(o.vtable()[2], uintptr(unsafe.Pointer(o)))
}

func comHR(vtbl uintptr, this unsafe.Pointer, args ...uintptr) uintptr {
	full := make([]uintptr, 0, 1+len(args))
	full = append(full, uintptr(this))
	full = append(full, args...)
	r1, _, _ := syscall.SyscallN(vtbl, full...)
	return r1
}

func dialogOwnerHWND() win.HWND {
	if form := walk.App().ActiveForm(); form != nil {
		return form.Handle()
	}
	return 0
}

func pickFolderPath(title string) (string, bool, error) {
	owner := dialogOwnerHWND()
	path, ok, err := pickFolderIFileDialog(owner, title)
	if err == nil {
		return path, ok, nil
	}
	// נפילה ל־SHBrowseForFolder אם COM המודרני נכשל
	return pickFolderSHBrowse(owner, title)
}

func pickFolderIFileDialog(owner win.HWND, title string) (string, bool, error) {
	var punk unsafe.Pointer
	hr := win.CoCreateInstance(
		&clsidFileOpenDialog,
		nil,
		0x17, // CLSCTX_ALL
		&iidIFileOpenDialog,
		&punk,
	)
	if hr != win.S_OK || punk == nil {
		return "", false, fmt.Errorf("CoCreateInstance IFileOpenDialog: 0x%X", uint32(hr))
	}
	dlg := (*comObject)(punk)
	defer dlg.Release()
	vt := dlg.vtable()

	// GetOptions (Vtbl 10) / SetOptions (9)
	var opts uint32
	if r := comHR(vt[10], unsafe.Pointer(dlg), uintptr(unsafe.Pointer(&opts))); r != 0 {
		return "", false, fmt.Errorf("GetOptions: 0x%X", uint32(r))
	}
	opts |= fosPickFolders | fosForceFileSystem | fosPathMustExist
	if r := comHR(vt[9], unsafe.Pointer(dlg), uintptr(opts)); r != 0 {
		return "", false, fmt.Errorf("SetOptions: 0x%X", uint32(r))
	}

	if title != "" {
		titlePtr, e := syscall.UTF16PtrFromString(title)
		if e != nil {
			return "", false, e
		}
		// SetTitle = Vtbl 17
		_ = comHR(vt[17], unsafe.Pointer(dlg), uintptr(unsafe.Pointer(titlePtr)))
	}

	// Show = Vtbl 3 (IModalWindow)
	r := comHR(vt[3], unsafe.Pointer(dlg), uintptr(owner))
	if uint32(r) == hresultCancel32 {
		return "", false, nil
	}
	if r != 0 {
		return "", false, fmt.Errorf("Show: 0x%X", uint32(r))
	}

	// GetResult = Vtbl 20
	var item unsafe.Pointer
	if r := comHR(vt[20], unsafe.Pointer(dlg), uintptr(unsafe.Pointer(&item))); r != 0 || item == nil {
		return "", false, fmt.Errorf("GetResult: 0x%X", uint32(r))
	}
	shellItem := (*comObject)(item)
	defer shellItem.Release()
	sit := shellItem.vtable()

	// IShellItem::GetDisplayName = Vtbl 5
	var namePtr *uint16
	if r := comHR(sit[5], unsafe.Pointer(shellItem), uintptr(sigdnFileSysPath), uintptr(unsafe.Pointer(&namePtr))); r != 0 || namePtr == nil {
		return "", false, fmt.Errorf("GetDisplayName: 0x%X", uint32(r))
	}
	defer win.CoTaskMemFree(uintptr(unsafe.Pointer(namePtr)))

	path := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(namePtr))[:])
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}

func pickFolderSHBrowse(owner win.HWND, title string) (string, bool, error) {
	const (
		BIF_RETURNONLYFSDIRS = 0x00000001
		BIF_EDITBOX          = 0x00000010
	)
	var display [win.MAX_PATH]uint16
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return "", false, err
	}
	// דיאלוג קלאסי — אמין גם בלי BIF_NEWDIALOGSTYLE
	bi := win.BROWSEINFO{
		HwndOwner:      owner,
		PszDisplayName: &display[0],
		LpszTitle:      titlePtr,
		UlFlags:        BIF_RETURNONLYFSDIRS | BIF_EDITBOX,
	}
	pidl := win.SHBrowseForFolder(&bi)
	if pidl == 0 {
		return "", false, nil
	}
	defer win.CoTaskMemFree(pidl)

	var pathBuf [win.MAX_PATH]uint16
	if !win.SHGetPathFromIDList(pidl, &pathBuf[0]) {
		return "", false, fmt.Errorf("SHGetPathFromIDList נכשל")
	}
	path := syscall.UTF16ToString(pathBuf[:])
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}
