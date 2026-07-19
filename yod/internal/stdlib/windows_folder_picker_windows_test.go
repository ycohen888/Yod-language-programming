//go:build windows

package stdlib

import (
	"testing"
	"unsafe"

	"github.com/lxn/win"
)

func TestCoCreateFileOpenDialog(t *testing.T) {
	_ = win.OleInitialize()
	var punk unsafe.Pointer
	hr := win.CoCreateInstance(
		&clsidFileOpenDialog,
		nil,
		0x17, // CLSCTX_ALL
		&iidIFileOpenDialog,
		&punk,
	)
	if hr != win.S_OK || punk == nil {
		t.Fatalf("CoCreateInstance: 0x%X", uint32(hr))
	}
	dlg := (*comObject)(punk)
	defer dlg.Release()
	vt := dlg.vtable()
	var opts uint32
	if r := comHR(vt[10], unsafe.Pointer(dlg), uintptr(unsafe.Pointer(&opts))); r != 0 {
		t.Fatalf("GetOptions: 0x%X", uint32(r))
	}
	opts |= fosPickFolders | fosForceFileSystem | fosPathMustExist
	if r := comHR(vt[9], unsafe.Pointer(dlg), uintptr(opts)); r != 0 {
		t.Fatalf("SetOptions: 0x%X", uint32(r))
	}
}
