//go:build windows

package stdlib

import (
	"syscall"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// enableSurfaceArrowKeys — IsDialogMessage בולע חצים אלא אם הווידג'ט
// מחזיר DLGC_WANTARROWS ב־WM_GETDLGCODE. CustomWidget של Walk לא עושה זאת.
func enableSurfaceArrowKeys(st *controlState, cw *walk.CustomWidget) {
	if st == nil || cw == nil || st.canvasKeysWired {
		return
	}
	hwnd := cw.Handle()
	if hwnd == 0 {
		return
	}
	orig := win.GetWindowLongPtr(hwnd, win.GWLP_WNDPROC)
	if orig == 0 {
		return
	}
	cb := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		if msg == win.WM_GETDLGCODE {
			return uintptr(win.DLGC_WANTARROWS | win.DLGC_WANTCHARS | win.DLGC_WANTALLKEYS)
		}
		return win.CallWindowProc(orig, hwnd, msg, wParam, lParam)
	})
	win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, cb)
	st.canvasKeyKeep = cb
	st.canvasKeysWired = true
}
