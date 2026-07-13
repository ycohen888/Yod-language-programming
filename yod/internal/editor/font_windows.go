//go:build windows

package editor

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// גופנים לקוד — עדיפות לכאלה שתומכים היטב בעברית + לטינית
var codeFontCandidates = []string{
	"Cascadia Code",
	"Cascadia Mono",
	"Courier New",
	"Consolas",
	"Lucida Console",
}

var (
	resolvedCodeFont string
	codeFontSize     = 14 // נקודות — נוח יותר לקריאה בעברית
)

func pickCodeFont() string {
	if resolvedCodeFont != "" {
		return resolvedCodeFont
	}
	for _, name := range codeFontCandidates {
		if fontFamilyExists(name) {
			resolvedCodeFont = name
			return resolvedCodeFont
		}
	}
	resolvedCodeFont = "Courier New"
	return resolvedCodeFont
}

func fontFamilyExists(name string) bool {
	f, err := walk.NewFont(name, 12, 0)
	if err != nil {
		return false
	}
	f.Dispose()
	return true
}

// קבועי RichEdit לשפות / IME
const (
	emGetLangOptions = win.WM_USER + 121
	emSetLangOptions = win.WM_USER + 120
	imfAutoKeyboard  = 0x0001
	imfAutoFont      = 0x0002
	imfDualFont      = 0x0080
	emSetUndoLimit   = win.WM_USER + 82
)

func configureHebrewTyping(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	opts := uint32(win.SendMessage(hwnd, emGetLangOptions, 0, 0))
	// חשוב: בלי AUTOFONT — RichEdit מחליף גופן באמצע עברית ושובר את החוויה
	opts &^= imfAutoFont | imfDualFont
	// מקלדת מתאימה כשעוברים בין עברית/אנגלית בטקסט
	opts |= imfAutoKeyboard
	win.SendMessage(hwnd, emSetLangOptions, 0, uintptr(opts))
	win.SendMessage(hwnd, emSetUndoLimit, 200, 0)
}

// ensureProcessDPI — חדות טקסט במסכים מודרניים
func ensureProcessDPI() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	if p := user32.NewProc("SetProcessDPIAware"); p.Find() == nil {
		_, _, _ = p.Call()
	}
}
