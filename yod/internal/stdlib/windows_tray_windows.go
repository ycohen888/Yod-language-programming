//go:build windows

package stdlib

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/win"

	"yod/internal/object"
)

func hideMainWindow(st *windowState) {
	if st == nil || st.mw == nil {
		return
	}
	st.mw.SetVisible(false)
}

func winHide(st *windowState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלון.הסתר מצפה ל־0 ארגומנטים")
	}
	hideMainWindow(st)
	return object.Nil
}

func winShowAgain(st *windowState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלון.הצג_שוב מצפה ל־0 ארגומנטים")
	}
	if st.mw == nil {
		return object.Nil
	}
	st.mw.SetVisible(true)
	win.ShowWindow(st.mw.Handle(), win.SW_RESTORE)
	_ = st.mw.BringToTop()
	return object.Nil
}

func winOnClosing(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 || !isCallable(args[0]) {
		return errObj("חלון.בסגירה מצפה לפונקציה")
	}
	st.onClosing = args[0]
	return object.Nil
}

func winForceClose(st *windowState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלון.סגור מצפה ל־0 ארגומנטים")
	}
	st.forceClose = true
	st.closed = true
	disposeWindowTray(st)
	disposeWindowBrowsers(st)
	if st.mw != nil {
		hwnd := st.mw.Handle()
		st.forceClose = true
		if hwnd != 0 {
			win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
			go func() {
				time.Sleep(150 * time.Millisecond)
				win.PostQuitMessage(0)
			}()
		} else {
			st.mw.Synchronize(func() {
				st.forceClose = true
				_ = st.mw.Close()
			})
		}
	}
	return object.Nil
}

func winOnTrayMenu(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 || !isCallable(args[0]) {
		return errObj("חלון.במגש מצפה לפונקציה")
	}
	st.onTrayMenu = args[0]
	return object.Nil
}

func winTray(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חלון.מגש מצפה למילון אפשרויות")
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("חלון.מגש מצפה למילון אפשרויות")
	}
	tip := ""
	if v, ok := h.Pairs["רמז"]; ok {
		if s, ok := asString(v); ok {
			tip = s
		}
	}
	iconPath := st.iconPath
	if v, ok := h.Pairs["איקון"]; ok {
		if s, ok := asString(v); ok && s != "" {
			iconPath = s
		}
	}
	items := []trayMenuItem{}
	if v, ok := h.Pairs["פריטים"]; ok {
		arr, ok := v.(*object.Array)
		if !ok {
			return errObj("חלון.מגש: פריטים חייבים להיות רשימה")
		}
		parsed, err := parseTrayMenuItems(arr)
		if err != nil {
			return errObj("חלון.מגש: " + err.Error())
		}
		items = parsed
	}
	st.trayTip = tip
	st.trayIconPath = iconPath
	st.trayItems = items
	st.trayConfigured = true
	if st.mw != nil {
		if err := applyWindowTray(st); err != nil {
			return errObj("חלון.מגש: " + err.Error())
		}
	}
	return object.Nil
}

func parseTrayMenuItems(arr *object.Array) ([]trayMenuItem, error) {
	out := make([]trayMenuItem, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		if s, ok := asString(el); ok {
			if s == "-" || s == "מפריד" {
				out = append(out, trayMenuItem{separator: true})
				continue
			}
			return nil, fmt.Errorf("פריט מחרוזת חייב להיות \"-\" או מילון")
		}
		h, ok := el.(*object.Hash)
		if !ok {
			return nil, fmt.Errorf("כל פריט חייב מחרוזת \"-\" או מילון")
		}
		if typ, ok := h.Pairs["סוג"]; ok {
			if s, ok := asString(typ); ok && (s == "מפריד" || s == "-") {
				out = append(out, trayMenuItem{separator: true})
				continue
			}
		}
		text := ""
		if v, ok := h.Pairs["טקסט"]; ok {
			if s, ok := asString(v); ok {
				text = s
			}
		}
		if text == "" {
			if v, ok := h.Pairs["שם"]; ok {
				if s, ok := asString(v); ok {
					text = s
				}
			}
		}
		if text == "" || text == "-" || text == "מפריד" {
			out = append(out, trayMenuItem{separator: true})
			continue
		}
		value := text
		if v, ok := h.Pairs["ערך"]; ok {
			if s, ok := asString(v); ok && s != "" {
				value = s
			}
		}
		out = append(out, trayMenuItem{text: text, value: value})
	}
	return out, nil
}

func disposeWindowTray(st *windowState) {
	if st == nil {
		return
	}
	if st.tray != nil {
		_ = st.tray.Dispose()
		st.tray = nil
	}
	if st.trayIcon != nil && st.trayIcon != st.icon {
		st.trayIcon.Dispose()
		st.trayIcon = nil
	}
	if st.trayHost != nil {
		st.trayHost.Dispose()
		st.trayHost = nil
	}
}

func loadTrayIcon(path string) (*walk.Icon, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("אין נתיב איקון")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".ico" {
		return nil, fmt.Errorf("מגש דורש קובץ .ico (קיבל: %s)", ext)
	}
	return walk.NewIconFromFile(path)
}

func applyWindowTray(st *windowState) error {
	if st.mw == nil {
		return nil
	}
	disposeWindowTray(st)

	// חלון נסתר נפרד — NewNotifyIcon דורס GWLP_USERDATA של ה־Form
	host, err := walk.NewMainWindow()
	if err != nil {
		return err
	}
	host.SetTitle("")
	host.SetVisible(false)
	st.trayHost = host

	ni, err := walk.NewNotifyIcon(host)
	if err != nil {
		disposeWindowTray(st)
		return err
	}

	iconPath := st.trayIconPath
	if iconPath == "" {
		iconPath = st.iconPath
	}
	var ic *walk.Icon
	if iconPath != "" {
		ic, err = loadTrayIcon(iconPath)
		if err != nil && st.icon != nil {
			ic = st.icon
			err = nil
		}
		if err != nil {
			// ממשיכים בלי איקון מותאם — עדיף מגש מאשר כלום
			ic = nil
		}
	}
	if ic != nil {
		if ic != st.icon {
			st.trayIcon = ic
		}
		_ = ni.SetIcon(ic)
	} else if st.icon != nil {
		_ = ni.SetIcon(st.icon)
	}

	if st.trayTip != "" {
		_ = ni.SetToolTip(st.trayTip)
	}
	for _, it := range st.trayItems {
		if it.separator {
			_ = ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
			continue
		}
		action := walk.NewAction()
		_ = action.SetText(it.text)
		val := it.value
		action.Triggered().Attach(func() {
			if st.onTrayMenu != nil {
				invokeYod(st.onTrayMenu, []object.Object{&object.String{Value: val}})
			} else if val == "הצג" || val == "show" {
				_ = winShowAgain(st)
			} else if val == "יציאה" || val == "exit" {
				_ = winForceClose(st)
			}
		})
		_ = ni.ContextMenu().Actions().Add(action)
	}
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		now := time.Now()
		if !st.trayLastLeftClick.IsZero() && now.Sub(st.trayLastLeftClick) < 450*time.Millisecond {
			st.trayLastLeftClick = time.Time{}
			_ = winShowAgain(st)
			return
		}
		st.trayLastLeftClick = now
	})
	if err := ni.SetVisible(true); err != nil {
		_ = ni.Dispose()
		disposeWindowTray(st)
		return err
	}
	st.tray = ni
	return nil
}

func winTrayNotify(st *windowState, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלון.מגש_התראה מצפה לכותרת וטקסט")
	}
	title, ok1 := asString(args[0])
	info, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חלון.מגש_התראה מצפה למחרוזות")
	}
	if st.tray == nil {
		return errObj("חלון.מגש_התראה: אין מגש פעיל — קראו ל־מגש קודם")
	}
	if err := st.tray.ShowInfo(I18nText(title), I18nText(info)); err != nil {
		return errObj("חלון.מגש_התראה: " + err.Error())
	}
	return object.Nil
}

func winRemoveTray(st *windowState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלון.הסר_מגש מצפה ל־0 ארגומנטים")
	}
	disposeWindowTray(st)
	st.trayConfigured = false
	return object.Nil
}
