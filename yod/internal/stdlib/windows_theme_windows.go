//go:build windows

package stdlib

import (
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"

	"yod/internal/object"
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

func preferAppDarkMode() {
	if procSetPrefAppMode.Find() == nil {
		procSetPrefAppMode.Call(2) // ForceDark
	}
	if procFlushTheme.Find() == nil {
		procFlushTheme.Call()
	}
}

// applyDarkChrome — כותרת חלון, תפריטים וסקרולים כהים (Win10/11).
func applyDarkChrome(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	darkExplorer, _ := syscall.UTF16PtrFromString("DarkMode_Explorer")
	win.SetWindowTheme(hwnd, darkExplorer, nil)

	var useDark int32 = 1
	procDwmSetWindowAttr.Call(
		uintptr(hwnd),
		dwmwaUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&useDark)),
		unsafe.Sizeof(useDark),
	)
	if procAllowDarkWindow.Find() == nil {
		procAllowDarkWindow.Call(uintptr(hwnd), 1)
	}
}

func applyDarkThemeToWidget(w walk.Window) {
	if w == nil {
		return
	}
	applyDarkChrome(w.Handle())
}

func clearWidgetFocusEffects(w walk.Window) {
	widget, ok := w.(walk.Widget)
	if !ok {
		return
	}
	fx := widget.GraphicsEffects()
	if walk.FocusEffect != nil {
		_ = fx.Remove(walk.FocusEffect)
	}
	if walk.InteractionEffect != nil {
		_ = fx.Remove(walk.InteractionEffect)
	}
}

func darkCtlBG() walk.Color      { return walk.RGB(30, 37, 46) }
func darkCtlAlt() walk.Color     { return walk.RGB(38, 46, 58) }
func darkCtlText() walk.Color    { return walk.RGB(230, 237, 243) }
func darkCtlMuted() walk.Color   { return walk.RGB(148, 163, 184) }
func darkCtlBorder() walk.Color  { return walk.RGB(48, 54, 61) }
func darkWinBG() walk.Color      { return walk.RGB(13, 17, 23) }
func darkPanelBG() walk.Color    { return walk.RGB(22, 27, 34) }
func darkBtnBG() walk.Color      { return walk.RGB(37, 99, 235) }
func darkBtnText() walk.Color    { return walk.RGB(255, 255, 255) }
func darkFieldBG() walk.Color    { return walk.RGB(30, 37, 46) }
func darkHeaderBG() walk.Color   { return walk.RGB(22, 27, 34) }
func darkSelBG() walk.Color { return walk.RGB(37, 99, 235) }

func parseDarkBool(name string, a ...object.Object) (bool, object.Object) {
	if len(a) != 1 {
		return false, errObj(name + " מצפה לאמת/שקר")
	}
	b, ok := a[0].(*object.Boolean)
	if !ok {
		return false, errObj(name + " מצפה לאמת/שקר")
	}
	return b.Value, nil
}
