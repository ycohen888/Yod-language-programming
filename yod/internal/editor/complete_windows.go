//go:build windows

package editor

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	acMaxVisible = 12
	acItemH      = 22
	acPopupW     = 268
	acScrollW    = 10
	acClassName  = "YodCompleteList"
	maNoActivate = 3
	spiGetWorkArea = 0x0030
	dtRTLReading = 0x00020000
	dtSingleLine = 0x00000020
	dtVCenter    = 0x00000004
	dtRight      = 0x00000002
	dtLeft       = 0x00000000
	dtEndEllipsis = 0x00008000
)

var (
	acClassOnce bool
	gdi32DLL    = windows.NewLazySystemDLL("gdi32.dll")
	user32DLL   = windows.NewLazySystemDLL("user32.dll")
	procCreateSolidBrush = gdi32DLL.NewProc("CreateSolidBrush")
	procDeleteObject     = gdi32DLL.NewProc("DeleteObject")
	procSetBkMode        = gdi32DLL.NewProc("SetBkMode")
	procSetTextColor     = gdi32DLL.NewProc("SetTextColor")
	procSelectObject     = gdi32DLL.NewProc("SelectObject")
	procCreateFont       = gdi32DLL.NewProc("CreateFontW")
	procFillRect         = user32DLL.NewProc("FillRect")
	procDrawText         = user32DLL.NewProc("DrawTextW")
	procSystemParams     = user32DLL.NewProc("SystemParametersInfoW")
)

type autoComplete struct {
	owner   *CodeEdit
	hwnd    win.HWND
	items   []completeItem
	sel     int
	scroll  int // אינדקס הפריט הראשון הגלוי
	visible bool
	query   completeQuery
	font    win.HFONT
}

func (ce *CodeEdit) ensureAutoComplete() {
	if ce.ac != nil {
		return
	}
	ce.ac = &autoComplete{owner: ce}
}

func registerCompleteClass() {
	if acClassOnce {
		return
	}
	acClassOnce = true
	var wc win.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(completeWndProc)
	wc.HInstance = win.GetModuleHandle(nil)
	wc.HCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW))
	wc.HbrBackground = win.HBRUSH(win.GetStockObject(win.BLACK_BRUSH))
	wc.LpszClassName = syscall.StringToUTF16Ptr(acClassName)
	win.RegisterClassEx(&wc)
}

func completeWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	ac := acFromHWND(hwnd)
	switch msg {
	case win.WM_CREATE:
		cs := (*win.CREATESTRUCT)(unsafe.Pointer(lParam))
		if cs != nil && cs.CreateParams != 0 {
			win.SetWindowLongPtr(hwnd, win.GWLP_USERDATA, cs.CreateParams)
		}
		return 0
	case win.WM_MOUSEACTIVATE:
		return maNoActivate
	case win.WM_LBUTTONDOWN, win.WM_LBUTTONDBLCLK:
		if ac == nil {
			return 0
		}
		y := int(int16(win.HIWORD(uint32(lParam))))
		x := int(int16(win.LOWORD(uint32(lParam))))
		if ac.clickScrollbar(x, y) {
			return 0
		}
		idx := ac.scroll + y/acItemH
		if idx >= 0 && idx < len(ac.items) {
			ac.sel = idx
			ac.accept()
		}
		return 0
	case win.WM_MOUSEMOVE:
		if ac == nil {
			return 0
		}
		y := int(int16(win.HIWORD(uint32(lParam))))
		idx := ac.scroll + y/acItemH
		if idx >= 0 && idx < len(ac.items) && idx != ac.sel {
			ac.sel = idx
			win.InvalidateRect(hwnd, nil, true)
		}
		return 0
	case win.WM_MOUSEWHEEL:
		if ac == nil {
			return 0
		}
		delta := int16(win.HIWORD(uint32(wParam)))
		steps := int(delta) / 120
		if steps == 0 {
			if delta > 0 {
				steps = 1
			} else {
				steps = -1
			}
		}
		ac.scrollBy(-steps)
		return 0
	case win.WM_PAINT:
		if ac != nil {
			ac.paint()
		}
		return 0
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_DESTROY:
		delete(acByHWND, hwnd)
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

var acByHWND = map[win.HWND]*autoComplete{}

func acFromHWND(hwnd win.HWND) *autoComplete {
	if ac, ok := acByHWND[hwnd]; ok {
		return ac
	}
	ptr := win.GetWindowLongPtr(hwnd, win.GWLP_USERDATA)
	if ptr == 0 {
		return nil
	}
	return (*autoComplete)(unsafe.Pointer(ptr))
}

func createSolidBrush(c win.COLORREF) win.HBRUSH {
	r, _, _ := procCreateSolidBrush.Call(uintptr(c))
	return win.HBRUSH(r)
}

func deleteObject(obj win.HGDIOBJ) {
	procDeleteObject.Call(uintptr(obj))
}

func fillRect(hdc win.HDC, rc *win.RECT, brush win.HBRUSH) {
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rc)), uintptr(brush))
}

func drawText(hdc win.HDC, text string, rc *win.RECT, format uint32) {
	ptr := syscall.StringToUTF16Ptr(text)
	procDrawText.Call(uintptr(hdc), uintptr(unsafe.Pointer(ptr)), uintptr(^uint32(0)), uintptr(unsafe.Pointer(rc)), uintptr(format))
}

func (ac *autoComplete) ensureFont() {
	if ac.font != 0 {
		return
	}
	r, _, _ := procCreateFont.Call(
		uintptr(16), 0, 0, 0, uintptr(win.FW_NORMAL), 0, 0, 0,
		uintptr(177), // HEBREW_CHARSET
		0, 0, uintptr(5), // CLEARTYPE_QUALITY
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Segoe UI"))),
	)
	ac.font = win.HFONT(r)
}

func (ac *autoComplete) create() error {
	if ac.hwnd != 0 {
		return nil
	}
	registerCompleteClass()
	parent := ac.owner.Handle()
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|win.WS_EX_NOACTIVATE,
		syscall.StringToUTF16Ptr(acClassName),
		nil,
		win.WS_POPUP|win.WS_BORDER|win.WS_CLIPCHILDREN,
		0, 0, acPopupW, acItemH,
		parent,
		0,
		win.GetModuleHandle(nil),
		unsafe.Pointer(ac),
	)
	if hwnd == 0 {
		return fmt.Errorf("יצירת חלון השלמה נכשלה")
	}
	ac.hwnd = hwnd
	acByHWND[hwnd] = ac
	ac.ensureFont()
	return nil
}

func (ac *autoComplete) destroy() {
	if ac.font != 0 {
		deleteObject(win.HGDIOBJ(ac.font))
		ac.font = 0
	}
	if ac.hwnd != 0 {
		delete(acByHWND, ac.hwnd)
		win.DestroyWindow(ac.hwnd)
		ac.hwnd = 0
	}
	ac.visible = false
}

func (ac *autoComplete) hide() {
	if ac == nil {
		return
	}
	if ac.hwnd != 0 {
		win.ShowWindow(ac.hwnd, win.SW_HIDE)
	}
	ac.visible = false
	ac.items = nil
}

func (ac *autoComplete) handleKey(key walk.Key) bool {
	if ac == nil || !ac.visible {
		return false
	}
	switch key {
	case walk.KeyEscape:
		ac.hide()
		return true
	case walk.KeyUp:
		ac.moveSel(-1)
		return true
	case walk.KeyDown:
		ac.moveSel(+1)
		return true
	case walk.KeyPrior:
		ac.moveSel(-acMaxVisible)
		return true
	case walk.KeyNext:
		ac.moveSel(+acMaxVisible)
		return true
	case walk.KeyTab, walk.KeyReturn:
		return ac.accept()
	}
	return false
}

func (ac *autoComplete) show(items []completeItem, q completeQuery) {
	if ac == nil || len(items) == 0 {
		if ac != nil {
			ac.hide()
		}
		return
	}
	if err := ac.create(); err != nil {
		return
	}
	ac.items = items
	ac.query = q
	ac.sel = 0
	ac.scroll = 0

	n := len(items)
	if n > acMaxVisible {
		n = acMaxVisible
	}
	h := n*acItemH + 2
	cx, cy := ac.owner.caretScreenPos()

	var wa win.RECT
	procSystemParams.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&wa)), 0)

	// בעברית/RTL: מימין לסמן — התפריט נפתח שמאלה (הקצה הימני ליד הסמן)
	x := cx - acPopupW
	y := cy + int(codeFontSize) + 6
	if y+h > int(wa.Bottom) {
		y = cy - h - 4
	}
	if y < int(wa.Top) {
		y = int(wa.Top) + 4
	}
	if x < int(wa.Left) {
		x = int(wa.Left) + 4
	}
	if x+acPopupW > int(wa.Right) {
		x = int(wa.Right) - acPopupW - 4
	}

	win.SetWindowPos(ac.hwnd, win.HWND_TOPMOST, int32(x), int32(y), acPopupW, int32(h),
		win.SWP_NOACTIVATE|win.SWP_SHOWWINDOW)
	ac.visible = true
	win.InvalidateRect(ac.hwnd, nil, true)
}

func (ce *CodeEdit) caretScreenPos() (x, y int) {
	// עדיפות למיקום סמן המערכת — מדויק יותר ב־RTL
	var caret win.POINT
	if win.GetCaretPos(&caret) {
		win.ClientToScreen(ce.Handle(), &caret)
		if caret.X != 0 || caret.Y != 0 {
			return int(caret.X), int(caret.Y)
		}
	}
	cp, _ := ce.TextSelection()
	var pt win.POINT
	ce.SendMessage(win.EM_POSFROMCHAR, uintptr(unsafe.Pointer(&pt)), uintptr(cp))
	if pt.X == 0 && pt.Y == 0 && cp > 0 {
		// נפילה: תו קודם
		ce.SendMessage(win.EM_POSFROMCHAR, uintptr(unsafe.Pointer(&pt)), uintptr(cp-1))
		pt.X += 8
	}
	win.ClientToScreen(ce.Handle(), &pt)
	return int(pt.X), int(pt.Y)
}

func (ac *autoComplete) visibleCount() int {
	n := len(ac.items)
	if n > acMaxVisible {
		return acMaxVisible
	}
	return n
}

func (ac *autoComplete) maxScroll() int {
	m := len(ac.items) - ac.visibleCount()
	if m < 0 {
		return 0
	}
	return m
}

func (ac *autoComplete) ensureSelVisible() {
	vis := ac.visibleCount()
	if vis <= 0 {
		return
	}
	if ac.sel < ac.scroll {
		ac.scroll = ac.sel
	}
	if ac.sel >= ac.scroll+vis {
		ac.scroll = ac.sel - vis + 1
	}
	if ac.scroll < 0 {
		ac.scroll = 0
	}
	if ac.scroll > ac.maxScroll() {
		ac.scroll = ac.maxScroll()
	}
}

func (ac *autoComplete) scrollBy(delta int) {
	if ac == nil || !ac.visible || len(ac.items) <= acMaxVisible {
		return
	}
	ac.scroll += delta
	if ac.scroll < 0 {
		ac.scroll = 0
	}
	if ac.scroll > ac.maxScroll() {
		ac.scroll = ac.maxScroll()
	}
	// שמור את הסימון בטווח הגלוי אם אפשר
	if ac.sel < ac.scroll {
		ac.sel = ac.scroll
	}
	if ac.sel >= ac.scroll+ac.visibleCount() {
		ac.sel = ac.scroll + ac.visibleCount() - 1
	}
	win.InvalidateRect(ac.hwnd, nil, true)
}

func (ac *autoComplete) clickScrollbar(x, y int) bool {
	if len(ac.items) <= acMaxVisible {
		return false
	}
	var rc win.RECT
	win.GetClientRect(ac.hwnd, &rc)
	// פס גלילה בצד שמאל (RTL — הפרטים משמאל)
	if x > int(rc.Left)+acScrollW {
		return false
	}
	h := int(rc.Bottom - rc.Top)
	if h <= 0 {
		return false
	}
	ratio := float64(y) / float64(h)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	ac.scroll = int(ratio * float64(ac.maxScroll()) + 0.5)
	if ac.scroll > ac.maxScroll() {
		ac.scroll = ac.maxScroll()
	}
	win.InvalidateRect(ac.hwnd, nil, true)
	return true
}

func (ac *autoComplete) paint() {
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(ac.hwnd, &ps)
	defer win.EndPaint(ac.hwnd, &ps)

	var rc win.RECT
	win.GetClientRect(ac.hwnd, &rc)

	bg := createSolidBrush(win.COLORREF(walk.RGB(37, 37, 38)))
	fillRect(hdc, &rc, bg)
	deleteObject(win.HGDIOBJ(bg))

	ac.ensureFont()
	old, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(ac.font))
	defer procSelectObject.Call(uintptr(hdc), old)
	procSetBkMode.Call(uintptr(hdc), uintptr(win.TRANSPARENT))

	hasScroll := len(ac.items) > acMaxVisible
	listLeft := rc.Left
	if hasScroll {
		listLeft += acScrollW
	}

	vis := ac.visibleCount()
	for row := 0; row < vis; row++ {
		i := ac.scroll + row
		if i < 0 || i >= len(ac.items) {
			break
		}
		it := ac.items[i]
		rowRc := win.RECT{
			Left: listLeft, Top: rc.Top + int32(row*acItemH),
			Right: rc.Right, Bottom: rc.Top + int32((row+1)*acItemH),
		}
		if i == ac.sel {
			selBrush := createSolidBrush(win.COLORREF(walk.RGB(14, 124, 112)))
			fillRect(hdc, &rowRc, selBrush)
			deleteObject(win.HGDIOBJ(selBrush))
			procSetTextColor.Call(uintptr(hdc), uintptr(win.COLORREF(walk.RGB(255, 255, 255))))
		} else {
			procSetTextColor.Call(uintptr(hdc), uintptr(win.COLORREF(walk.RGB(220, 220, 220))))
		}

		detail := it.Detail
		if detail == "" {
			detail = it.Kind.label()
		}

		textRect := rowRc
		textRect.Left += 72
		textRect.Right -= 8
		drawText(hdc, it.Text, &textRect, dtRTLReading|dtSingleLine|dtVCenter|dtRight|dtEndEllipsis)

		if i == ac.sel {
			procSetTextColor.Call(uintptr(hdc), uintptr(win.COLORREF(walk.RGB(200, 230, 220))))
		} else {
			procSetTextColor.Call(uintptr(hdc), uintptr(win.COLORREF(walk.RGB(140, 145, 150))))
		}
		detRect := rowRc
		detRect.Left += 8
		detRect.Right = rowRc.Left + 68
		drawText(hdc, detail, &detRect, dtRTLReading|dtSingleLine|dtVCenter|dtLeft|dtEndEllipsis)
	}

	if hasScroll {
		ac.paintScrollbar(hdc, &rc)
	}
}

func (ac *autoComplete) paintScrollbar(hdc win.HDC, rc *win.RECT) {
	track := win.RECT{Left: rc.Left, Top: rc.Top, Right: rc.Left + acScrollW, Bottom: rc.Bottom}
	trackBrush := createSolidBrush(win.COLORREF(walk.RGB(45, 46, 48)))
	fillRect(hdc, &track, trackBrush)
	deleteObject(win.HGDIOBJ(trackBrush))

	total := len(ac.items)
	vis := ac.visibleCount()
	h := int(rc.Bottom - rc.Top)
	if total <= 0 || h <= 0 {
		return
	}
	thumbH := h * vis / total
	if thumbH < 16 {
		thumbH = 16
	}
	if thumbH > h {
		thumbH = h
	}
	maxS := ac.maxScroll()
	thumbY := 0
	if maxS > 0 {
		thumbY = ac.scroll * (h - thumbH) / maxS
	}
	thumb := win.RECT{
		Left: track.Left + 2, Top: rc.Top + int32(thumbY),
		Right: track.Right - 2, Bottom: rc.Top + int32(thumbY+thumbH),
	}
	thumbBrush := createSolidBrush(win.COLORREF(walk.RGB(14, 124, 112)))
	fillRect(hdc, &thumb, thumbBrush)
	deleteObject(win.HGDIOBJ(thumbBrush))
}

func (ac *autoComplete) moveSel(delta int) {
	if !ac.visible || len(ac.items) == 0 {
		return
	}
	ac.sel += delta
	if ac.sel < 0 {
		ac.sel = len(ac.items) - 1
	}
	if ac.sel >= len(ac.items) {
		ac.sel = 0
	}
	ac.ensureSelVisible()
	win.InvalidateRect(ac.hwnd, nil, true)
}

func (ac *autoComplete) accept() bool {
	if ac == nil || !ac.visible || len(ac.items) == 0 {
		return false
	}
	if ac.sel < 0 || ac.sel >= len(ac.items) {
		return false
	}
	item := ac.items[ac.sel]
	ce := ac.owner
	_, end := ce.TextSelection()
	from := ac.query.ReplaceFrom
	if from < 0 {
		from = end
	}
	ac.hide()
	if ce.acDebounce != nil {
		ce.acDebounce.Stop()
	}
	ce.SetTextSelection(from, end)
	ce.insertTextRaw(item.Text)
	ce.textChangedPublisher.Publish()
	ce.scheduleHighlight()
	return true
}

func (ce *CodeEdit) forceCompletion() {
	ce.ensureAutoComplete()
	if ce.acDebounce != nil {
		ce.acDebounce.Stop()
	}
	ce.acHold = true
	text := ce.indexText()
	caret, _ := ce.TextSelection()
	q := analyzeCompletion(text, caret)
	if q.Mode == modeNone {
		q = completeQuery{Mode: modeIdent, Prefix: "", ReplaceFrom: caret}
	}
	var items []completeItem
	switch {
	case q.Mode == modeIdent && q.Prefix == "":
		items = identSuggestions("")
	case q.Mode == modeInclude && q.Prefix == "":
		items = librarySuggestions("")
	case q.Mode == modeMember && q.Prefix == "":
		items = memberSuggestions(q.TypeHint, q.Receiver, "")
	default:
		items = buildSuggestions(q)
		if len(items) == 0 && q.Mode == modeIdent {
			items = identSuggestions("")
		}
	}
	if len(items) == 0 {
		ce.ac.hide()
		ce.acHold = false
		return
	}
	ce.ac.show(items, q)
	time.AfterFunc(200*time.Millisecond, func() {
		ce.Synchronize(func() { ce.acHold = false })
	})
}

func (ce *CodeEdit) refreshCompletion() {
	ce.ensureAutoComplete()
	if ce.suppress {
		return
	}
	if ce.acHold {
		return
	}
	text := ce.indexText()
	caret, _ := ce.TextSelection()
	q := analyzeCompletion(text, caret)
	if q.Mode == modeNone {
		ce.ac.hide()
		return
	}
	items := buildSuggestions(q)
	if len(items) == 0 {
		ce.ac.hide()
		return
	}
	ce.ac.show(items, q)
}
