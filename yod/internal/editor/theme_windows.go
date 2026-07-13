//go:build windows

package editor

import (
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

var (
	dwmapi               = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttr = dwmapi.NewProc("DwmSetWindowAttribute")
	uxtheme              = syscall.NewLazyDLL("uxtheme.dll")
	procAllowDarkWindow  = uxtheme.NewProc("AllowDarkModeForWindow")
	procSetPrefAppMode   = uxtheme.NewProc("SetPreferredAppMode")
	procFlushTheme       = uxtheme.NewProc("FlushMenuThemes")
)

const dwmwaUseImmersiveDarkMode = 20

// stripBorder מסיר מסגרת שקועה/גבול מחלון עריכה
func stripBorder(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	ex := uint32(win.GetWindowLong(hwnd, win.GWL_EXSTYLE))
	ex &^= win.WS_EX_CLIENTEDGE | win.WS_EX_STATICEDGE | win.WS_EX_WINDOWEDGE | win.WS_EX_DLGMODALFRAME
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(ex))

	style := uint32(win.GetWindowLong(hwnd, win.GWL_STYLE))
	style &^= win.WS_BORDER | win.WS_DLGFRAME | win.WS_THICKFRAME
	win.SetWindowLong(hwnd, win.GWL_STYLE, int32(style))

	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
}

// applyDarkScrollbars ערכת נושא כהה לסקרול (Windows 10/11)
func applyDarkScrollbars(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	// DarkMode_Explorer — סקרול כהה תואם Explorer
	dark, _ := syscall.UTF16PtrFromString("DarkMode_Explorer")
	win.SetWindowTheme(hwnd, dark, nil)

	var useDark int32 = 1
	procDwmSetWindowAttr.Call(
		uintptr(hwnd),
		dwmwaUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&useDark)),
		unsafe.Sizeof(useDark),
	)

	// AllowDarkModeForWindow (ordinal / שם — אם קיים)
	if procAllowDarkWindow.Find() == nil {
		procAllowDarkWindow.Call(uintptr(hwnd), 1)
	}
}

func preferAppDarkMode() {
	// SetPreferredAppMode(ForceDark=2) — אם זמין ב־uxtheme
	if procSetPrefAppMode.Find() == nil {
		procSetPrefAppMode.Call(2)
	}
	if procFlushTheme.Find() == nil {
		procFlushTheme.Call()
	}
}

// styleEditorPane מסיר גבול + סקרול dark לחלון עריכה
func styleEditorPane(w walk.Window) {
	if w == nil {
		return
	}
	hwnd := w.Handle()
	stripBorder(hwnd)
	applyDarkScrollbars(hwnd)
	clearFocusChrome(w)
}

func clearFocusChrome(w walk.Window) {
	if widget, ok := w.(walk.Widget); ok {
		fx := widget.GraphicsEffects()
		if walk.FocusEffect != nil {
			_ = fx.Remove(walk.FocusEffect)
		}
		if walk.InteractionEffect != nil {
			_ = fx.Remove(walk.InteractionEffect)
		}
	}
}
