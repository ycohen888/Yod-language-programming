//go:build windows

package stdlib

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// uiLayoutRTL — כיוון ממשק לפי שפת תרגום פעילה (עברית וכו' = RTL).
func uiLayoutRTL() bool {
	return i18nDirForLang(i18nLang) == "rtl"
}

// applyWindowTitleDir — כותרת חלון Windows:
// RTL: לוגו+שם מימין; משמאל סגירה ← הגדל ← מזער.
// LTR: כמו ברירת מחדל של Windows (לוגו+שם משמאל; מזער/הגדל/סגור מימין).
// תוכן הלקוח לא יורש שיקוף (WebView/פקדים מנהלים dir בעצמם).
func applyWindowTitleDir(mw *walk.MainWindow) {
	if mw == nil {
		return
	}
	rtl := uiLayoutRTL()
	_ = mw.SetRightToLeftLayout(rtl)
	_ = mw.SetRightToLeftReading(rtl)
	if !rtl {
		return
	}
	clearContainerLayoutRTL(mw)
	// רענון מסגרת כדי ש־DWM יצייר מחדש את כפתורי הכותרת
	hwnd := mw.Handle()
	if hwnd != 0 {
		win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
			win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
	}
}

func clearContainerLayoutRTL(c walk.Container) {
	if c == nil {
		return
	}
	// מבטלים שיקוף על אזור הלקוח והילדים — הכותרת נשארת RTL על ה־HWND הראשי
	if cb := c.AsContainerBase(); cb != nil {
		clearHwndLayoutRTL(cb.Handle())
	}
	list := c.Children()
	if list == nil {
		return
	}
	for i := 0; i < list.Len(); i++ {
		clearWidgetTreeLayoutRTL(list.At(i))
	}
}

func clearWidgetTreeLayoutRTL(w walk.Widget) {
	if w == nil {
		return
	}
	clearHwndLayoutRTL(w.Handle())
	if cont, ok := w.(walk.Container); ok {
		list := cont.Children()
		if list == nil {
			return
		}
		for i := 0; i < list.Len(); i++ {
			clearWidgetTreeLayoutRTL(list.At(i))
		}
	}
}

func clearHwndLayoutRTL(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	ex := uint32(win.GetWindowLong(hwnd, win.GWL_EXSTYLE))
	ex |= win.WS_EX_NOINHERITLAYOUT
	ex &^= win.WS_EX_LAYOUTRTL
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(ex))
}
