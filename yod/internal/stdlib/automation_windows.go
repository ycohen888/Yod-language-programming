//go:build windows

package stdlib

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"yod/internal/object"
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseEventMove     = 0x0001
	mouseEventLeftDown = 0x0002
	mouseEventLeftUp   = 0x0004
	mouseEventRightDown = 0x0008
	mouseEventRightUp  = 0x0010
	mouseEventMiddleDown = 0x0020
	mouseEventMiddleUp = 0x0040
	mouseEventWheel      = 0x0800
	mouseEventAbsolute   = 0x8000

	keyEventKeyUp   = 0x0002
	keyEventUnicode = 0x0004

	vkReturn = 0x0D
	vkTab    = 0x09
	vkEscape = 0x1B
	vkBack   = 0x08
	vkSpace  = 0x20
	vkControl = 0x11
	vkMenu    = 0x12 // Alt
	vkShift   = 0x10
	vkLWin    = 0x5B
)

var (
	user32Auto       = windows.NewLazySystemDLL("user32.dll")
	gdi32Auto        = windows.NewLazySystemDLL("gdi32.dll")
	kernel32Auto     = windows.NewLazySystemDLL("kernel32.dll")
	procSetCursorPos = user32Auto.NewProc("SetCursorPos")
	procSendInput    = user32Auto.NewProc("SendInput")
	procGetSystemMetrics = user32Auto.NewProc("GetSystemMetrics")
	procGetDC        = user32Auto.NewProc("GetDC")
	procReleaseDC    = user32Auto.NewProc("ReleaseDC")
	procCreateCompatibleDC = gdi32Auto.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32Auto.NewProc("CreateCompatibleBitmap")
	procSelectObject = gdi32Auto.NewProc("SelectObject")
	procBitBlt       = gdi32Auto.NewProc("BitBlt")
	procDeleteObject = gdi32Auto.NewProc("DeleteObject")
	procDeleteDC     = gdi32Auto.NewProc("DeleteDC")
	procGetDIBits    = gdi32Auto.NewProc("GetDIBits")
	procFindWindowW  = user32Auto.NewProc("FindWindowW")
	procEnumWindows  = user32Auto.NewProc("EnumWindows")
	procGetWindowTextW = user32Auto.NewProc("GetWindowTextW")
	procIsWindowVisible = user32Auto.NewProc("IsWindowVisible")
	procSetForegroundWindow = user32Auto.NewProc("SetForegroundWindow")
	procGetWindowRect = user32Auto.NewProc("GetWindowRect")
	procShowWindow = user32Auto.NewProc("ShowWindow")
	procGetForegroundWindow = user32Auto.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32Auto.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput = user32Auto.NewProc("AttachThreadInput")
	procGetCurrentThreadId = kernel32Auto.NewProc("GetCurrentThreadId")
)

type sendInputMouse struct {
	Type  uint32
	_     uint32
	Dx    int32
	Dy    int32
	Data  uint32
	Flags uint32
	Time  uint32
	Extra uintptr
}

// sizeof(INPUT) על amd64 = 40: Type+pad + union (MOUSEINPUT 32 בתוכו)
type sendInputKey struct {
	Type  uint32
	_     uint32
	Vk    uint16
	Scan  uint16
	Flags uint32
	Time  uint32
	Extra uintptr
	_pad  [12]byte
}

func automationMoveMouse(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.הזז_עכבר", 2, args); err != nil {
		return err
	}
	xN, ok1 := args[0].(*object.Number)
	yN, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("הזז_עכבר מצפה ל־(x, y) מספרים")
	}
	r, _, err := procSetCursorPos.Call(uintptr(int32(xN.Value)), uintptr(int32(yN.Value)))
	if r == 0 {
		return errObj("הזזת עכבר נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func automationClickMouse(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("לחץ_עכבר מצפה ל־(\"שמאל\"|\"ימין\"|\"אמצע\" [, כפול])")
	}
	btn, ok := asString(args[0])
	if !ok {
		return errObj("כפתור עכבר חייב להיות מחרוזת")
	}
	times := 1
	if len(args) == 2 {
		if b, ok := args[1].(*object.Boolean); ok && b.Value {
			times = 2
		} else if n, ok := args[1].(*object.Number); ok {
			times = int(n.Value)
		}
	}
	var down, up uint32
	switch btn {
	case "שמאל", "left":
		down, up = mouseEventLeftDown, mouseEventLeftUp
	case "ימין", "right":
		down, up = mouseEventRightDown, mouseEventRightUp
	case "אמצע", "middle":
		down, up = mouseEventMiddleDown, mouseEventMiddleUp
	default:
		return errObj("כפתור לא מוכר: " + btn + " (שמאל/ימין/אמצע)")
	}
	for i := 0; i < times; i++ {
		if err := sendMouseFlag(down); err != nil {
			return errObj(err.Error())
		}
		if err := sendMouseFlag(up); err != nil {
			return errObj(err.Error())
		}
		if i+1 < times {
			time.Sleep(40 * time.Millisecond)
		}
	}
	return &object.Null{}
}

func sendMouseFlag(flags uint32) error {
	in := sendInputMouse{Type: inputMouse, Flags: flags}
	n, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if n == 0 {
		return fmt.Errorf("לחיצת עכבר נכשלה: %v", err)
	}
	return nil
}

func automationType(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.הקלד", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("הקלד מצפה למחרוזת")
	}
	for _, r := range s {
		if err := sendUnicode(r, false); err != nil {
			return errObj(err.Error())
		}
		if err := sendUnicode(r, true); err != nil {
			return errObj(err.Error())
		}
	}
	return &object.Null{}
}

func sendUnicode(r rune, up bool) error {
	flags := uint32(keyEventUnicode)
	if up {
		flags |= keyEventKeyUp
	}
	in := sendInputKey{Type: inputKeyboard, Scan: uint16(r), Flags: flags}
	n, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if n == 0 {
		return fmt.Errorf("הקלדה נכשלה: %v", err)
	}
	return nil
}

func automationKeyPress(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.לחץ_מקש", 1, args); err != nil {
		return err
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("לחץ_מקש מצפה לשם מקש (מחרוזת)")
	}
	vk, ok := automationKeyVK(name)
	if !ok {
		return errObj("מקש לא מוכר: " + name + " (אנטר/טאב/Escape/Backspace/רווח)")
	}
	if err := sendVK(vk, false); err != nil {
		return errObj(err.Error())
	}
	if err := sendVK(vk, true); err != nil {
		return errObj(err.Error())
	}
	return &object.Null{}
}

func automationKeyVK(name string) (uint16, bool) {
	if vk, ok := keyboardNameToVK(name); ok {
		return uint16(vk), true
	}
	n := strings.TrimSpace(strings.ToLower(name))
	switch n {
	case "בקרה":
		return vkControl, true
	case "הזזה":
		return vkShift, true
	case "חלונות", "meta":
		return vkLWin, true
	case "יציאה":
		return vkEscape, true
	}
	if strings.HasPrefix(n, "f") && len(n) >= 2 {
		var num int
		if _, err := fmt.Sscanf(n, "f%d", &num); err == nil && num >= 1 && num <= 24 {
			return uint16(0x70 + num - 1), true
		}
	}
	return 0, false
}

func automationHotkey(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.קיצור", 1, args); err != nil {
		return err
	}
	arr, ok := args[0].(*object.Array)
	if !ok || len(arr.Elements) == 0 {
		return errObj("קיצור מצפה לרשימת מקשים (למשל [\"בקרה\",\"s\"])")
	}
	vks := make([]uint16, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		s, ok := asString(el)
		if !ok {
			return errObj("כל מקש בקיצור חייב להיות מחרוזת")
		}
		vk, ok := automationKeyVK(s)
		if !ok {
			return errObj("מקש לא מוכר בקיצור: " + s)
		}
		vks = append(vks, vk)
	}
	for _, vk := range vks {
		if err := sendVK(vk, false); err != nil {
			return errObj(err.Error())
		}
	}
	for i := len(vks) - 1; i >= 0; i-- {
		if err := sendVK(vks[i], true); err != nil {
			return errObj(err.Error())
		}
	}
	return &object.Null{}
}

func automationKeyDown(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.החזק", 1, args); err != nil {
		return err
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("החזק מצפה לשם מקש")
	}
	vk, ok := automationKeyVK(name)
	if !ok {
		return errObj("מקש לא מוכר: " + name)
	}
	if err := sendVK(vk, false); err != nil {
		return errObj(err.Error())
	}
	return &object.Null{}
}

func automationKeyUp(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.שחרר", 1, args); err != nil {
		return err
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("שחרר מצפה לשם מקש")
	}
	vk, ok := automationKeyVK(name)
	if !ok {
		return errObj("מקש לא מוכר: " + name)
	}
	if err := sendVK(vk, true); err != nil {
		return errObj(err.Error())
	}
	return &object.Null{}
}

type winRect struct {
	Left, Top, Right, Bottom int32
}

func automationFindWindow(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.מצא_חלון", 1, args); err != nil {
		return err
	}
	pattern, ok := asString(args[0])
	if !ok {
		return errObj("מצא_חלון מצפה למחרוזת כותרת (תומך *)")
	}
	hwnd, title, err := findWindowByTitle(pattern)
	if err != nil {
		return errObj(err.Error())
	}
	if hwnd == 0 {
		return &object.Null{}
	}
	h := object.NewHash()
	h.Set("מזהה", &object.Number{Value: float64(hwnd)})
	h.Set("כותרת", &object.String{Value: title})
	return h
}

func automationActivateWindow(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.הפעל_חלון", 1, args); err != nil {
		return err
	}
	hwnd, errObjv := resolveHwnd(args[0])
	if errObjv != nil {
		return errObjv
	}
	const swRestore = 9
	procShowWindow.Call(hwnd, swRestore)
	// AttachThreadInput — מעלה סיכוי להעברת מיקוד ב־Windows 10+
	fg, _, _ := procGetForegroundWindow.Call()
	if fg != 0 && fg != hwnd {
		var fgTid, tgtTid uint32
		procGetWindowThreadProcessId.Call(fg, uintptr(unsafe.Pointer(&fgTid)))
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&tgtTid)))
		curTid, _, _ := procGetCurrentThreadId.Call()
		if fgTid != 0 && tgtTid != 0 {
			procAttachThreadInput.Call(uintptr(fgTid), curTid, 1)
			procAttachThreadInput.Call(uintptr(tgtTid), curTid, 1)
		}
	}
	r, _, err := procSetForegroundWindow.Call(hwnd)
	if fg != 0 && fg != hwnd {
		var fgTid, tgtTid uint32
		procGetWindowThreadProcessId.Call(fg, uintptr(unsafe.Pointer(&fgTid)))
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&tgtTid)))
		curTid, _, _ := procGetCurrentThreadId.Call()
		if fgTid != 0 && tgtTid != 0 {
			procAttachThreadInput.Call(uintptr(fgTid), curTid, 0)
			procAttachThreadInput.Call(uintptr(tgtTid), curTid, 0)
		}
	}
	if r == 0 {
		return errObj("הפעלת חלון נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func automationDragMouse(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.גרור_עכבר", 4, args); err != nil {
		return err
	}
	x1, ok1 := args[0].(*object.Number)
	y1, ok2 := args[1].(*object.Number)
	x2, ok3 := args[2].(*object.Number)
	y2, ok4 := args[3].(*object.Number)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return errObj("גרור_עכבר מצפה ל־(x1, y1, x2, y2)")
	}
	if _, _, err := procSetCursorPos.Call(uintptr(int32(x1.Value)), uintptr(int32(y1.Value))); err != nil {
		return errObj("גרירה: הזזה לנקודת התחלה נכשלה")
	}
	if err := sendMouseFlag(mouseEventLeftDown); err != nil {
		return errObj(err.Error())
	}
	time.Sleep(30 * time.Millisecond)
	if _, _, err := procSetCursorPos.Call(uintptr(int32(x2.Value)), uintptr(int32(y2.Value))); err != nil {
		_ = sendMouseFlag(mouseEventLeftUp)
		return errObj("גרירה: הזזה ליעד נכשלה")
	}
	time.Sleep(30 * time.Millisecond)
	if err := sendMouseFlag(mouseEventLeftUp); err != nil {
		return errObj(err.Error())
	}
	return &object.Null{}
}

func automationScrollWheel(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.גלגל_עכבר", 1, args); err != nil {
		return err
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("גלגל_עכבר מצפה למספר (חיובי=למעלה, שלילי=למטה)")
	}
	steps := int(n.Value)
	if steps == 0 {
		return &object.Null{}
	}
	delta := int32(steps * 120)
	in := sendInputMouse{Type: inputMouse, Data: uint32(delta), Flags: mouseEventWheel}
	cnt, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if cnt == 0 {
		return errObj("גלגל עכבר נכשל: " + err.Error())
	}
	return &object.Null{}
}

func automationTypeDelayed(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("הקלד_מעכב מצפה ל־(מחרוזת [, מילישניות])")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("הקלד_מעכב מצפה למחרוזת")
	}
	delay := 40
	if len(args) == 2 {
		if d, ok := args[1].(*object.Number); ok {
			delay = int(d.Value)
			if delay < 0 {
				delay = 0
			}
		}
	}
	for _, r := range s {
		if err := sendUnicode(r, false); err != nil {
			return errObj(err.Error())
		}
		if err := sendUnicode(r, true); err != nil {
			return errObj(err.Error())
		}
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
	return &object.Null{}
}

func automationListWindows(args ...object.Object) object.Object {
	pattern := "*"
	if len(args) == 1 {
		p, ok := asString(args[0])
		if !ok {
			return errObj("רשימת_חלונות מצפה למחרוזת נוסחת או ל־0 ארגומנטים")
		}
		pattern = p
	} else if len(args) > 1 {
		return errObj("רשימת_חלונות מצפה ל־0 או 1 ארגומנטים")
	}
	wins := enumerateWindows(pattern, 0)
	out := &object.Array{Elements: make([]object.Object, 0, len(wins))}
	for _, w := range wins {
		h := object.NewHash()
		h.Set("מזהה", &object.Number{Value: float64(w.hwnd)})
		h.Set("כותרת", &object.String{Value: w.title})
		out.Elements = append(out.Elements, h)
	}
	return out
}

func automationWaitForWindow(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("המתן_עד_חלון מצפה ל־(נוסחת_כותרת [, מילישניות])")
	}
	pattern, ok := asString(args[0])
	if !ok {
		return errObj("המתן_עד_חלון: נוסחת כותרת חייבת להיות מחרוזת")
	}
	timeout := 5000
	if len(args) == 2 {
		if ms, ok := args[1].(*object.Number); ok {
			timeout = int(ms.Value)
			if timeout < 0 {
				timeout = 0
			}
		}
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
	for {
		hwnd, title, err := findWindowByTitle(pattern)
		if err != nil {
			return errObj(err.Error())
		}
		if hwnd != 0 {
			h := object.NewHash()
			h.Set("מזהה", &object.Number{Value: float64(hwnd)})
			h.Set("כותרת", &object.String{Value: title})
			return h
		}
		if time.Now().After(deadline) {
			return &object.Null{}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type winEntry struct {
	hwnd  uintptr
	title string
}

func enumerateWindows(pattern string, limit int) []winEntry {
	var out []winEntry
	type enumState struct {
		pattern string
		limit   int
		out     *[]winEntry
	}
	st := &enumState{pattern: pattern, limit: limit, out: &out}
	cb := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		state := (*enumState)(unsafe.Pointer(lparam))
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		var buf [512]uint16
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
		if n == 0 {
			return 1
		}
		title := windows.UTF16ToString(buf[:])
		if !matchTitlePattern(state.pattern, title) {
			return 1
		}
		*state.out = append(*state.out, winEntry{hwnd: hwnd, title: title})
		if state.limit > 0 && len(*state.out) >= state.limit {
			return 0
		}
		return 1
	})
	procEnumWindows.Call(cb, uintptr(unsafe.Pointer(st)))
	return out
}

func automationWindowRect(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.מלבן_חלון", 1, args); err != nil {
		return err
	}
	hwnd, errObjv := resolveHwnd(args[0])
	if errObjv != nil {
		return errObjv
	}
	var rc winRect
	r, _, err := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	if r == 0 {
		return errObj("קריאת מלבן חלון נכשלה: " + err.Error())
	}
	h := object.NewHash()
	h.Set("x", &object.Number{Value: float64(rc.Left)})
	h.Set("y", &object.Number{Value: float64(rc.Top)})
	h.Set("רוחב", &object.Number{Value: float64(rc.Right-rc.Left)})
	h.Set("גובה", &object.Number{Value: float64(rc.Bottom-rc.Top)})
	return h
}

func resolveHwnd(arg object.Object) (uintptr, object.Object) {
	switch v := arg.(type) {
	case *object.Number:
		hwnd := uintptr(v.Value)
		if hwnd == 0 {
			return 0, errObj("חלון לא נמצא")
		}
		return hwnd, nil
	case *object.String:
		h, _, err := findWindowByTitle(v.Value)
		if err != nil {
			return 0, errObj(err.Error())
		}
		if h == 0 {
			return 0, errObj("חלון לא נמצא")
		}
		return h, nil
	case *object.Hash:
		if id, ok := v.Pairs["מזהה"]; ok {
			if n, ok := id.(*object.Number); ok {
				hwnd := uintptr(n.Value)
				if hwnd == 0 {
					return 0, errObj("חלון לא נמצא")
				}
				return hwnd, nil
			}
		}
	}
	return 0, errObj("מצפים למזהה, כותרת או מילון מ־מצא_חלון")
}

func findWindowByTitle(pattern string) (uintptr, string, error) {
	if !strings.Contains(pattern, "*") {
		ptr, err := windows.UTF16PtrFromString(pattern)
		if err != nil {
			return 0, "", err
		}
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(ptr)))
		if hwnd != 0 {
			return hwnd, pattern, nil
		}
	}
	type enumState struct {
		pattern string
		hwnd    uintptr
		title   string
	}
	st := &enumState{pattern: pattern}
	cb := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		state := (*enumState)(unsafe.Pointer(lparam))
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		var buf [512]uint16
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
		if n == 0 {
			return 1
		}
		title := windows.UTF16ToString(buf[:])
		if matchTitlePattern(state.pattern, title) {
			state.hwnd = hwnd
			state.title = title
			return 0
		}
		return 1
	})
	procEnumWindows.Call(cb, uintptr(unsafe.Pointer(st)))
	return st.hwnd, st.title, nil
}

func sendVK(vk uint16, up bool) error {
	flags := uint32(0)
	if up {
		flags = keyEventKeyUp
	}
	in := sendInputKey{Type: inputKeyboard, Vk: vk, Flags: flags}
	n, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if n == 0 {
		return fmt.Errorf("לחיצת מקש נכשלה: %v", err)
	}
	return nil
}

func automationSleep(args ...object.Object) object.Object {
	if err := expectArgs("אוטומציה.המתן", 1, args); err != nil {
		return err
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("המתן מצפה למספר מילישניות")
	}
	ms := int(n.Value)
	if ms < 0 {
		ms = 0
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return &object.Null{}
}

func automationScreenshot(args ...object.Object) object.Object {
	var img image.Image
	var err error
	switch len(args) {
	case 0:
		img, err = captureScreen(0, 0, 0, 0)
	case 4:
		x, ok1 := args[0].(*object.Number)
		y, ok2 := args[1].(*object.Number)
		w, ok3 := args[2].(*object.Number)
		h, ok4 := args[3].(*object.Number)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return errObj("צלם_מסך מצפה ל־() או (x, y, רוחב, גובה)")
		}
		img, err = captureScreen(int(x.Value), int(y.Value), int(w.Value), int(h.Value))
	default:
		return errObj("צלם_מסך מצפה ל־() או (x, y, רוחב, גובה)")
	}
	if err != nil {
		return errObj("צילום מסך נכשל: " + err.Error())
	}
	return wrapImage(img)
}

func automationScreenshotToFile(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 5 {
		return errObj("צלם_מסך_לקובץ מצפה ל־(נתיב [, x, y, רוחב, גובה])")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("צלם_מסך_לקובץ מצפה לנתיב מחרוזת")
	}
	path = resolveAppPath(path)
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		path += ".jpg"
		ext = ".jpg"
	}
	var img image.Image
	var err error
	switch len(args) {
	case 1:
		img, err = captureScreen(0, 0, 0, 0)
	case 5:
		x, ok1 := args[1].(*object.Number)
		y, ok2 := args[2].(*object.Number)
		w, ok3 := args[3].(*object.Number)
		h, ok4 := args[4].(*object.Number)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return errObj("צלם_מסך_לקובץ: x,y,רוחב,גובה חייבים להיות מספרים")
		}
		img, err = captureScreen(int(x.Value), int(y.Value), int(w.Value), int(h.Value))
	default:
		return errObj("צלם_מסך_לקובץ מצפה ל־(נתיב) או (נתיב, x, y, רוחב, גובה)")
	}
	if err != nil {
		return errObj("צילום מסך נכשל: " + err.Error())
	}
	if err := saveScreenshotFile(img, path, ext); err != nil {
		return errObj("שמירת צילום נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func saveScreenshotFile(img image.Image, path, ext string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	rgba := imageToRGBA(img)
	switch ext {
	case ".png":
		return png.Encode(f, rgba)
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, rgba, &jpeg.Options{Quality: 85})
	default:
		return jpeg.Encode(f, rgba, &jpeg.Options{Quality: 85})
	}
}

const (
	smXVirtualScreen = 76
	smYVirtualScreen = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79
	srcCopy = 0x00CC0020
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func captureScreen(x, y, w, h int) (image.Image, error) {
	if w <= 0 || h <= 0 {
		vx, _, _ := procGetSystemMetrics.Call(smXVirtualScreen)
		vy, _, _ := procGetSystemMetrics.Call(smYVirtualScreen)
		vw, _, _ := procGetSystemMetrics.Call(smCXVirtualScreen)
		vh, _, _ := procGetSystemMetrics.Call(smCYVirtualScreen)
		x, y = int(int32(vx)), int(int32(vy))
		w, h = int(int32(vw)), int(int32(vh))
		if w <= 0 || h <= 0 {
			// fallback primary
			cx, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
			cy, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
			x, y = 0, 0
			w, h = int(cx), int(cy)
		}
	}
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, fmt.Errorf("GetDC נכשל")
	}
	defer procReleaseDC.Call(0, hdc)

	mdc, _, _ := procCreateCompatibleDC.Call(hdc)
	if mdc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC נכשל")
	}
	defer procDeleteDC.Call(mdc)

	bmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	if bmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap נכשל")
	}
	defer procDeleteObject.Call(bmp)

	procSelectObject.Call(mdc, bmp)
	r, _, err := procBitBlt.Call(mdc, 0, 0, uintptr(w), uintptr(h), hdc, uintptr(x), uintptr(y), srcCopy)
	if r == 0 {
		return nil, fmt.Errorf("BitBlt נכשל: %v", err)
	}

	bi := bitmapInfoHeader{
		Size:     40,
		Width:    int32(w),
		Height:   -int32(h), // top-down
		Planes:   1,
		BitCount: 32,
	}
	buf := make([]byte, w*h*4)
	ret, _, err := procGetDIBits.Call(hdc, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), 0)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits נכשל: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			i := (py*w + px) * 4
			// BGRA from GetDIBits
			img.Pix[i+0] = buf[i+2]
			img.Pix[i+1] = buf[i+1]
			img.Pix[i+2] = buf[i+0]
			img.Pix[i+3] = 255
		}
	}
	return img, nil
}
