//go:build windows

package editor

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// GUIDs / flags לדיאלוג תיקיות מודרני (Vista+)
var (
	clsidFileOpenDialog = win.CLSID{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = win.IID{0xD57C7288, 0xD4AD, 0x4768, [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidIShellItem       = win.IID{0x43826D1E, 0xE718, 0x42EE, [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

const (
	fosPickFolders     = 0x20
	fosForceFileSystem = 0x40
	fosPathMustExist   = 0x800
	sigdnFileSysPath   = 0x80058000
)

type iFileOpenDialogVtbl struct {
	QueryInterface        uintptr
	AddRef                uintptr
	Release               uintptr
	Show                  uintptr
	SetFileTypes          uintptr
	SetFileTypeIndex      uintptr
	GetFileTypeIndex      uintptr
	Advise                uintptr
	Unadvise              uintptr
	SetOptions            uintptr
	GetOptions            uintptr
	SetDefaultFolder      uintptr
	SetFolder             uintptr
	GetFolder             uintptr
	GetCurrentSelection   uintptr
	SetFileName           uintptr
	GetFileName           uintptr
	SetTitle              uintptr
	SetOkButtonLabel      uintptr
	SetFileNameLabel      uintptr
	GetResult             uintptr
	AddPlace              uintptr
	SetDefaultExtension   uintptr
	Close                 uintptr
	SetClientGuid         uintptr
	ClearClientData       uintptr
	SetFilter             uintptr
	GetResults            uintptr
	GetSelectedItems      uintptr
}

type iFileOpenDialog struct {
	LpVtbl *iFileOpenDialogVtbl
}

type iShellItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

type iShellItem struct {
	LpVtbl *iShellItemVtbl
}

func (d *iFileOpenDialog) Release() uint32 {
	ret, _, _ := syscall.Syscall(d.LpVtbl.Release, 1, uintptr(unsafe.Pointer(d)), 0, 0)
	return uint32(ret)
}

func (d *iFileOpenDialog) Show(hwnd win.HWND) win.HRESULT {
	ret, _, _ := syscall.Syscall(d.LpVtbl.Show, 2, uintptr(unsafe.Pointer(d)), uintptr(hwnd), 0)
	return win.HRESULT(ret)
}

func (d *iFileOpenDialog) GetOptions(opts *uint32) win.HRESULT {
	ret, _, _ := syscall.Syscall(d.LpVtbl.GetOptions, 2, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(opts)), 0)
	return win.HRESULT(ret)
}

func (d *iFileOpenDialog) SetOptions(opts uint32) win.HRESULT {
	ret, _, _ := syscall.Syscall(d.LpVtbl.SetOptions, 2, uintptr(unsafe.Pointer(d)), uintptr(opts), 0)
	return win.HRESULT(ret)
}

func (d *iFileOpenDialog) SetTitle(title *uint16) win.HRESULT {
	ret, _, _ := syscall.Syscall(d.LpVtbl.SetTitle, 2, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(title)), 0)
	return win.HRESULT(ret)
}

func (d *iFileOpenDialog) SetFolder(item *iShellItem) win.HRESULT {
	ret, _, _ := syscall.Syscall(d.LpVtbl.SetFolder, 2, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(item)), 0)
	return win.HRESULT(ret)
}

func (d *iFileOpenDialog) GetResult(item **iShellItem) win.HRESULT {
	ret, _, _ := syscall.Syscall(d.LpVtbl.GetResult, 2, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(item)), 0)
	return win.HRESULT(ret)
}

func (s *iShellItem) Release() uint32 {
	ret, _, _ := syscall.Syscall(s.LpVtbl.Release, 1, uintptr(unsafe.Pointer(s)), 0, 0)
	return uint32(ret)
}

func (s *iShellItem) GetDisplayName(sigdn uint32, name **uint16) win.HRESULT {
	ret, _, _ := syscall.Syscall(s.LpVtbl.GetDisplayName, 3,
		uintptr(unsafe.Pointer(s)), uintptr(sigdn), uintptr(unsafe.Pointer(name)))
	return win.HRESULT(ret)
}

var (
	modShell32                    = syscall.NewLazyDLL("shell32.dll")
	procSHCreateItemFromParsingName = modShell32.NewProc("SHCreateItemFromParsingName")
)

func shCreateItemFromParsingName(path string) (*iShellItem, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var item *iShellItem
	hr, _, _ := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(p)),
		0,
		uintptr(unsafe.Pointer(&iidIShellItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	if win.HRESULT(hr) != win.S_OK || item == nil {
		return nil, fmt.Errorf("SHCreateItemFromParsingName: 0x%X", hr)
	}
	runtime.KeepAlive(p)
	return item, nil
}

// pickFolder — דיאלוג בחירת תיקייה מודרני (IFileOpenDialog + FOS_PICKFOLDERS).
func pickFolder(owner walk.Form, title, initial string) (string, bool, error) {
	hr := win.OleInitialize()
	if hr != win.S_OK && hr != win.S_FALSE {
		return "", false, fmt.Errorf("OleInitialize: 0x%X", uint32(hr))
	}

	var ownerHwnd win.HWND
	if owner != nil {
		ownerHwnd = owner.Handle()
	}

	var unk unsafe.Pointer
	hr = win.CoCreateInstance(
		(*win.CLSID)(unsafe.Pointer(&clsidFileOpenDialog)),
		nil,
		win.CLSCTX_INPROC_SERVER,
		(*win.IID)(unsafe.Pointer(&iidIFileOpenDialog)),
		&unk,
	)
	if hr != win.S_OK || unk == nil {
		// נפילה ל־SHBrowseForFolder אם COM נכשל
		return pickFolderLegacy(ownerHwnd, title, initial)
	}
	dlg := (*iFileOpenDialog)(unk)
	defer dlg.Release()

	var opts uint32
	if hr = dlg.GetOptions(&opts); hr != win.S_OK {
		return "", false, fmt.Errorf("GetOptions: 0x%X", uint32(hr))
	}
	opts |= fosPickFolders | fosForceFileSystem | fosPathMustExist
	if hr = dlg.SetOptions(opts); hr != win.S_OK {
		return "", false, fmt.Errorf("SetOptions: 0x%X", uint32(hr))
	}

	if title != "" {
		t, err := syscall.UTF16PtrFromString(title)
		if err != nil {
			return "", false, err
		}
		_ = dlg.SetTitle(t)
		runtime.KeepAlive(t)
	}

	if initial != "" {
		if item, err := shCreateItemFromParsingName(initial); err == nil {
			_ = dlg.SetFolder(item)
			item.Release()
		}
	}

	hr = dlg.Show(ownerHwnd)
	const errorCancelled = 0x800704C7
	if uint32(hr) == errorCancelled || hr == win.HRESULT(1) /* S_FALSE */ {
		return "", false, nil
	}
	if hr != win.S_OK {
		return "", false, fmt.Errorf("Show: 0x%X", uint32(hr))
	}

	var item *iShellItem
	if hr = dlg.GetResult(&item); hr != win.S_OK || item == nil {
		return "", false, nil
	}
	defer item.Release()

	var name *uint16
	if hr = item.GetDisplayName(sigdnFileSysPath, &name); hr != win.S_OK || name == nil {
		return "", false, nil
	}
	defer win.CoTaskMemFree(uintptr(unsafe.Pointer(name)))

	path := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(name))[:])
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}

// pickFolderLegacy — גיבוי ישן אם IFileOpenDialog לא זמין.
func pickFolderLegacy(ownerHwnd win.HWND, title, initial string) (string, bool, error) {
	const (
		bifReturnOnlyFSDirs = 0x00000001
		bifNewDialogStyle   = 0x00000040
		bffmInitialized     = 1
		bffmSetSelectionW   = win.WM_USER + 103
	)

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

var browseInitialUTF16 *uint16
var browseFolderCallbackPtr uintptr

func init() {
	browseFolderCallbackPtr = syscall.NewCallback(browseFolderCallback)
}

func browseFolderCallback(hwnd win.HWND, msg uint32, lp, wp uintptr) uintptr {
	const bffmInitialized = 1
	const bffmSetSelectionW = win.WM_USER + 103
	if msg == bffmInitialized && browseInitialUTF16 != nil {
		win.SendMessage(hwnd, bffmSetSelectionW, 1, uintptr(unsafe.Pointer(browseInitialUTF16)))
	}
	return 0
}
