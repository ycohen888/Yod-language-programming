//go:build windows

package stdlib

import (
	"strings"
	"syscall"

	"github.com/lxn/win"

	"yod/internal/object"
)

var (
	gdi32Shape             = syscall.NewLazyDLL("gdi32.dll")
	user32Shape            = syscall.NewLazyDLL("user32.dll")
	procCreateEllipticRgn  = gdi32Shape.NewProc("CreateEllipticRgn")
	procCreateRoundRectRgn = gdi32Shape.NewProc("CreateRoundRectRgn")
	procSetWindowRgn       = user32Shape.NewProc("SetWindowRgn")
)

// winSetWindowShape — חותך את החלון לצורה (עיגול / מעוגל) כך שהדסקטופ נראה מסביב
// בלי LWA_COLORKEY (ששובר לחיצות). הצורה נשמרת ומוחלת מחדש אחרי שינוי גודל.
func winSetWindowShape(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חלון.קבע_צורת_חלון מצפה למחרוזת: \"עיגול\" / \"מעוגל\" / \"מלבן\"")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return errObj("חלון.קבע_צורת_חלון מצפה למחרוזת")
	}
	shape := strings.TrimSpace(s.Value)
	switch shape {
	case "עיגול", "מעוגל", "מלבן", "":
		st.windowShape = shape
	default:
		return errObj("צורת חלון לא מוכרת — השתמשו ב־עיגול / מעוגל / מלבן")
	}
	if st.mw != nil {
		applyWindowShape(st)
	}
	return object.Nil
}

func applyWindowShape(st *windowState) {
	if st == nil || st.mw == nil {
		return
	}
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return
	}
	shape := strings.TrimSpace(st.windowShape)
	if shape == "" || shape == "מלבן" {
		_, _, _ = procSetWindowRgn.Call(uintptr(hwnd), 0, 1)
		return
	}

	var r win.RECT
	if !win.GetWindowRect(hwnd, &r) {
		return
	}
	w := r.Right - r.Left
	h := r.Bottom - r.Top
	if w < 2 || h < 2 {
		return
	}

	var rgn uintptr
	switch shape {
	case "עיגול":
		// עיגול מדויק לפי הצד הקטן, ממורכז. +1 כי CreateEllipticRgn לא כולל right/bottom.
		side := w
		if h < side {
			side = h
		}
		ox := (w - side) / 2
		oy := (h - side) / 2
		rgn, _, _ = procCreateEllipticRgn.Call(
			uintptr(ox),
			uintptr(oy),
			uintptr(ox+side+1),
			uintptr(oy+side+1),
		)
	case "מעוגל":
		// רדיוס יחסי לגודל — כרטיס מעוגל ברור (הווידג׳ט נחתך; דסקטופ מסביב = "שקוף")
		rad := w
		if h < w {
			rad = h
		}
		rad = rad / 6
		if rad < 28 {
			rad = 28
		}
		if rad > 72 {
			rad = 72
		}
		rgn, _, _ = procCreateRoundRectRgn.Call(0, 0, uintptr(w+1), uintptr(h+1), uintptr(rad), uintptr(rad))
	default:
		_, _, _ = procSetWindowRgn.Call(uintptr(hwnd), 0, 1)
		return
	}
	if rgn == 0 {
		return
	}
	// SetWindowRgn לוקח בעלות על ה־HRGN — אין DeleteObject.
	_, _, _ = procSetWindowRgn.Call(uintptr(hwnd), rgn, 1)
	// אחרי חיתוך צורה — מבטלים שוב מסגרת DWM לבנה (Win11 מצייר אותה מחוץ ל־RGN)
	disableDwmChromeArtifacts(hwnd)
}
