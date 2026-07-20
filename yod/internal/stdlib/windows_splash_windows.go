//go:build windows

package stdlib

import (
	"strings"
	"syscall"
	"time"

	"github.com/lxn/win"
)

var procSetLayeredWindowAttributes = syscall.NewLazyDLL("user32.dll").NewProc("SetLayeredWindowAttributes")

const lwaAlpha = 0x2
const lwaColorKey = 0x1

// setWindowAlpha — שקיפות חלון (0=שקוף לגמרי, 255=אטום) דרך WS_EX_LAYERED.
// מאפשר ל־WebView2 לצייר בזמן שהמשתמש עדיין לא רואה את החלון.
func setWindowAlpha(hwnd win.HWND, alpha byte) {
	if hwnd == 0 {
		return
	}
	ex := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	if ex&win.WS_EX_LAYERED == 0 {
		win.SetWindowLong(hwnd, win.GWL_EXSTYLE, ex|win.WS_EX_LAYERED)
	}
	_, _, _ = procSetLayeredWindowAttributes.Call(
		uintptr(hwnd),
		0,
		uintptr(alpha),
		uintptr(lwaAlpha),
	)
}

// setWindowColorKey — מפתח צבע (זהירות: עם WebView2 גורם לעתים למסגרת לבנה ולחיצות שבורות).
func setWindowColorKey(hwnd win.HWND, color win.COLORREF) {
	if hwnd == 0 {
		return
	}
	ex := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	if ex&win.WS_EX_LAYERED == 0 {
		win.SetWindowLong(hwnd, win.GWL_EXSTYLE, ex|win.WS_EX_LAYERED)
	}
	_, _, _ = procSetLayeredWindowAttributes.Call(
		uintptr(hwnd),
		uintptr(color),
		0,
		uintptr(lwaColorKey),
	)
}

// clearWindowLayered — מסיר WS_EX_LAYERED (מבטל Color Key / alpha ישנים).
func clearWindowLayered(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	ex := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	if ex&win.WS_EX_LAYERED == 0 {
		return
	}
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, ex&^win.WS_EX_LAYERED)
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
}


func htmlLooksLikeDesignSplash(html string) bool {
	if html == "" {
		return false
	}
	return strings.Contains(html, "yod-splash") ||
		strings.Contains(html, "__יוד_הצג_טעינה") ||
		strings.Contains(html, "שפת יוד")
}

func messageIsSplashReady(msg string) bool {
	return strings.Contains(msg, "טעינה_מוכנה")
}

func applySplashTransparent(st *windowState) {
	if st == nil || st.mw == nil {
		return
	}
	hwnd := win.HWND(st.mw.Handle())
	// קודם שקיפות, ואז Show — בלי פריים כהה על המסך
	setWindowAlpha(hwnd, 0)
	st.mw.SetVisible(true)
	applyWindowPlacement(st)
}

func revealSplashWindow(st *windowState) {
	if st == nil || !st.deferShow || st.revealed || st.mw == nil || st.closed {
		return
	}
	st.revealed = true
	st.mw.Synchronize(func() {
		if st.closed || st.mw == nil {
			return
		}
		st.mw.SetVisible(true)
		setWindowAlpha(win.HWND(st.mw.Handle()), 255)
		applyWindowPlacement(st)
		st.mw.Show()
		time.AfterFunc(480*time.Millisecond, func() {
			if st.mw == nil || st.closed {
				return
			}
			st.mw.Synchronize(func() {
				releaseDeferredDesignReady(st)
			})
		})
	})
}
