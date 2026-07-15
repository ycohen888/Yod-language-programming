//go:build windows

package stdlib

import (
	"syscall"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// enableSurfaceArrowKeys — IsDialogMessage בולע חצים אלא אם הווידג'ט
// מחזיר DLGC_WANTARROWS ב־WM_GETDLGCODE. CustomWidget של Walk לא עושה זאת.
// גם מעביר חזרות אוטומטיות של חצים/מחיקה (לחיצה ארוכה) ל־בעת_מקש.
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
		// bit 30 = המקש כבר היה לחוץ → חזרה אוטומטית של Windows
		if msg == win.WM_KEYDOWN && (lParam&(1<<30)) != 0 {
			if handleSurfaceKeyRepeat(st, walk.Key(wParam)) {
				return 0
			}
		}
		return win.CallWindowProc(orig, hwnd, msg, wParam, lParam)
	})
	win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, cb)
	st.canvasKeyKeep = cb
	st.canvasKeysWired = true
}

// handleSurfaceKeyRepeat — ניווט/מחיקה בלחיצה ארוכה (Walk עלול לא להעביר חזרות).
func handleSurfaceKeyRepeat(ch *controlState, key walk.Key) bool {
	if ch == nil {
		return false
	}
	if walk.ModifiersDown()&walk.ModControl != 0 {
		return false
	}
	name := ""
	switch key {
	case walk.KeyLeft:
		name = "שמאלה"
	case walk.KeyRight:
		name = "ימינה"
	case walk.KeyUp:
		name = "למעלה"
	case walk.KeyDown:
		name = "למטה"
	case walk.KeyHome:
		name = "התחלה"
	case walk.KeyEnd:
		name = "סוף"
	case walk.KeyPrior:
		name = "עמוד למעלה"
	case walk.KeyNext:
		name = "עמוד למטה"
	case walk.KeyBack:
		name = "מחיקה"
	case walk.KeyDelete:
		name = "מחק"
	default:
		return false
	}
	invokeKeyCmd(ch, name)
	return true
}
