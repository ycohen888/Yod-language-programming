//go:build windows

package editor

import (
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"

	"yod/internal/highlight"
)

var (
	msfteditOnce  bool
	richeditClass = win.MSFTEDIT_CLASS
)

func loadRichEdit() {
	if msfteditOnce {
		return
	}
	msfteditOnce = true
	if _, err := windows.LoadLibrary("msftedit.dll"); err != nil {
		_, _ = windows.LoadLibrary("riched20.dll")
		richeditClass = win.RICHEDIT_CLASS
	}
}

// CodeEdit — עורך קוד RTL עם צביעת תחביר PHP Dark+
type CodeEdit struct {
	walk.WidgetBase
	textChangedPublisher walk.EventPublisher
	highlighting         bool
	debounce             *time.Timer
	hlGen                uint64 // דור צביעה — מבטל עבודה ישנה
	suppress             bool
	onZoom               func(size int) // עדכון gutter / סטטוס אחרי שינוי גודל
	onSelChange          func()         // עדכון שורה/עמודה בסטטוס
	ac                   *autoComplete
	acDebounce           *time.Timer
	ignoreSpaceChar      bool // אחרי Ctrl+Space — לא להכניס רווח
	acHold               bool // השלמה ידנית — לא לסגור מיד בגלל רענון
	// היסטוריית טקסט משלנו — RichEdit+highlight שוברים את Undo המקורי
	undoStack    []textSnap
	redoStack    []textSnap
	lastSnap     textSnap
	haveLast     bool
	histLock     bool
	histBurst      bool
	histDebounce   *time.Timer
	lastTextLen    int
	fullHLOnce     bool // טעינה / הדבקה גדולה / Undo — צביעה מלאה
}

func NewCodeEdit(parent walk.Container) (*CodeEdit, error) {
	loadRichEdit()
	ce := new(CodeEdit)

	style := uint32(win.WS_TABSTOP | win.WS_VISIBLE | win.WS_VSCROLL | win.WS_HSCROLL |
		win.ES_MULTILINE | win.ES_WANTRETURN | win.ES_AUTOVSCROLL | win.ES_AUTOHSCROLL)

	if err := walk.InitWidget(ce, parent, richeditClass, style, 0); err != nil {
		return nil, err
	}

	ce.SendMessage(win.EM_SETBKGNDCOLOR, 0, uintptr(highlight.ColorBackground))
	ce.applyDefaultFormat()
	ce.setCodePara()
	ce.SendMessage(win.EM_SETTARGETDEVICE, 0, 1)
	// חובה ב־RichEdit — אחרת אין EN_CHANGE בזמן הקלדה (צבעים + מספרי שורות)
	mask := ce.SendMessage(win.EM_GETEVENTMASK, 0, 0)
	ce.SendMessage(win.EM_SETEVENTMASK, 0, mask|uintptr(win.ENM_CHANGE|win.ENM_SELCHANGE))
	// שוליים פנימיים — נוח יותר לקריאה בעברית
	const (
		ecLeftMargin  = 0x0001
		ecRightMargin = 0x0002
	)
	ce.SendMessage(win.EM_SETMARGINS, ecLeftMargin|ecRightMargin, uintptr(14|(14<<16)))
	configureHebrewTyping(ce.Handle())

	styleEditorPane(ce)

	ce.MustRegisterProperty("Text", walk.NewProperty(
		func() interface{} { return ce.Text() },
		func(v interface{}) error {
			s, _ := v.(string)
			return ce.SetText(s)
		},
		ce.textChangedPublisher.Event()))

	return ce, nil
}

func (ce *CodeEdit) applyDefaultFormat() {
	ce.applyDefaultFormatFlags(win.SCF_ALL)
}

func (ce *CodeEdit) applyDefaultFormatSelection() {
	ce.applyDefaultFormatFlags(win.SCF_SELECTION)
}

func (ce *CodeEdit) applyDefaultFormatFlags(flags uint32) {
	var cf win.CHARFORMAT2
	cf.CbSize = uint32(unsafe.Sizeof(cf))
	cf.DwMask = win.CFM_COLOR | win.CFM_FACE | win.CFM_SIZE | win.CFM_CHARSET
	cf.CrTextColor = win.COLORREF(highlight.ColorForeground)
	cf.YHeight = int32(codeFontSize * 20)
	cf.BCharSet = win.DEFAULT_CHARSET
	copy(cf.SzFaceName[:], syscall.StringToUTF16(pickCodeFont()))
	ce.SendMessage(win.EM_SETCHARFORMAT, uintptr(flags), uintptr(unsafe.Pointer(&cf)))
}

func (ce *CodeEdit) setCodePara() {
	// פסקה RTL אמיתית — נוח לכתיבה בעברית (בלוקים עם «סוף»)
	var pf win.PARAFORMAT2
	pf.CbSize = uint32(unsafe.Sizeof(pf))
	pf.DwMask = win.PFM_RTLPARA | win.PFM_ALIGNMENT | win.PFM_TABSTOPS | win.PFM_LINESPACING | win.PFM_SPACEAFTER
	pf.WEffects = win.PFE_RTLPARA
	pf.WAlignment = win.PFA_RIGHT
	// מרווח שורה ~1.2× — נוח יותר לקריאה בעברית
	pf.BLineSpacingRule = 5 // DyLineSpacing/20 = מספר שורות
	pf.DyLineSpacing = 24
	pf.DySpaceAfter = 40
	pf.CTabCount = 8
	for i := int32(0); i < 8; i++ {
		pf.RgxTabs[i] = (i + 1) * 480 // ~2 רווחים לסגנון יוד
	}
	ce.SendMessage(win.EM_SETPARAFORMAT, 0, uintptr(unsafe.Pointer(&pf)))
}

func (ce *CodeEdit) Text() string {
	return stripBidiMarks(ce.rawText())
}

// rawText — הטקסט כמו ב־RichEdit (כולל סימני בידי) לסמן/השלמה
func (ce *CodeEdit) rawText() string {
	n := int(ce.SendMessage(win.WM_GETTEXTLENGTH, 0, 0))
	if n <= 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	ce.SendMessage(win.WM_GETTEXT, uintptr(n+1), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

// indexText — טקסט תואם לאינדקסי סמן של RichEdit (\r בלבד, לא \r\n)
func (ce *CodeEdit) indexText() string {
	t := ce.rawText()
	// WM_GETTEXT מחזיר CRLF, אבל EM_EXGETSEL סופר ירידת שורה כ־\r אחד
	return strings.ReplaceAll(t, "\r\n", "\r")
}

func (ce *CodeEdit) SetText(text string) error {
	ce.suppress = true
	clean := stripBidiMarks(text)
	ptr := syscall.StringToUTF16Ptr(clean)
	ok := ce.SendMessage(win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(ptr)))
	ce.suppress = false
	if ok != win.TRUE {
		return syscall.EINVAL
	}
	ce.applyDefaultFormat()
	ce.setCodePara()
	ce.resetHistoryFromCurrent()
	ce.fullHLOnce = true
	ce.lastTextLen = len(clean)
	ce.rehighlight()
	ce.textChangedPublisher.Publish()
	return nil
}

func (ce *CodeEdit) TextChanged() *walk.Event {
	return ce.textChangedPublisher.Event()
}

func (ce *CodeEdit) TextSelection() (start, end int) {
	var cr win.CHARRANGE
	ce.SendMessage(win.EM_EXGETSEL, 0, uintptr(unsafe.Pointer(&cr)))
	return int(cr.CpMin), int(cr.CpMax)
}

func (ce *CodeEdit) SetTextSelection(start, end int) {
	cr := win.CHARRANGE{CpMin: int32(start), CpMax: int32(end)}
	ce.SendMessage(win.EM_EXSETSEL, 0, uintptr(unsafe.Pointer(&cr)))
}

func (ce *CodeEdit) LineCount() int {
	n := int(ce.SendMessage(win.EM_GETLINECOUNT, 0, 0))
	if n < 1 {
		return 1
	}
	return n
}

func (ce *CodeEdit) FirstVisibleLine() int {
	return int(ce.SendMessage(win.EM_GETFIRSTVISIBLELINE, 0, 0))
}

func (ce *CodeEdit) LineFromChar(cp int) int {
	return int(ce.SendMessage(win.EM_LINEFROMCHAR, uintptr(cp), 0))
}

func (ce *CodeEdit) LineIndex(line int) int {
	return int(ce.SendMessage(win.EM_LINEINDEX, uintptr(line), 0))
}

func (ce *CodeEdit) ScrollCaret() {
	ce.SendMessage(win.EM_SCROLLCARET, 0, 0)
}

const (
	highlightIdle  = 700 * time.Millisecond
	completionIdle = 500 * time.Millisecond
	historyIdle    = 600 * time.Millisecond
)

func (ce *CodeEdit) scheduleHighlight() {
	if ce.suppress {
		return
	}
	gen := atomic.AddUint64(&ce.hlGen, 1)
	if ce.debounce != nil {
		ce.debounce.Stop()
	}
	ce.debounce = time.AfterFunc(highlightIdle, func() {
		ce.Synchronize(func() {
			if ce.suppress || atomic.LoadUint64(&ce.hlGen) != gen {
				return
			}
			text := ce.Text()
			start, end := ce.TextSelection()
			full := ce.fullHLOnce
			ce.fullHLOnce = false
			delta := len(text) - ce.lastTextLen
			if delta < 0 {
				delta = -delta
			}
			if delta > 120 {
				full = true
			}
			ce.lastTextLen = len(text)
			lineStart, lineEnd := 0, 0
			if !full {
				lineStart, lineEnd = ce.lineRangeAround(start, 1)
			}
			go func(myGen uint64, src string, selStart, selEnd int, doFull bool, ls, le int) {
				spans := highlight.SpansRichEdit(src)
				ce.Synchronize(func() {
					if ce.suppress || atomic.LoadUint64(&ce.hlGen) != myGen {
						return
					}
					if doFull {
						ce.applyHighlightFull(spans, selStart, selEnd)
					} else {
						ce.applyHighlightRange(spans, ls, le, selStart, selEnd)
					}
				})
			}(gen, text, start, end, full, lineStart, lineEnd)
		})
	})
}

// onTextEdited — מינימום בעת הקלדה; עבודה כבדה רק אחרי הפסקה
func (ce *CodeEdit) onTextEdited() {
	if ce.suppress || ce.highlighting {
		return
	}
	ce.recordHistory()
	ce.textChangedPublisher.Publish()
	ce.scheduleHighlight()
	// השלמה אוטומטית רק אם החלון כבר פתוח — לא לפתוח תוך כדי הקלדה
	if ce.ac != nil && ce.ac.visible {
		ce.scheduleCompletion()
	} else if ce.ac != nil {
		ce.ac.hide()
	}
}

func (ce *CodeEdit) scheduleCompletion() {
	ce.ensureAutoComplete()
	if ce.acDebounce != nil {
		ce.acDebounce.Stop()
	}
	ce.acDebounce = time.AfterFunc(completionIdle, func() {
		ce.Synchronize(func() {
			if ce.suppress || ce.highlighting {
				return
			}
			ce.refreshCompletion()
		})
	})
}

func (ce *CodeEdit) lineRangeAround(cp, pad int) (start, end int) {
	line := ce.LineFromChar(cp)
	first := line - pad
	if first < 0 {
		first = 0
	}
	last := line + pad
	maxLine := ce.LineCount() - 1
	if maxLine < 0 {
		maxLine = 0
	}
	if last > maxLine {
		last = maxLine
	}
	start = ce.LineIndex(first)
	if last >= maxLine {
		end = int(ce.SendMessage(win.WM_GETTEXTLENGTH, 0, 0))
	} else {
		end = ce.LineIndex(last + 1)
	}
	if end < start {
		end = start
	}
	return start, end
}

func (ce *CodeEdit) rehighlight() {
	if ce.highlighting {
		return
	}
	ce.fullHLOnce = true
	text := ce.Text()
	start, end := ce.TextSelection()
	ce.lastTextLen = len(text)
	ce.applyHighlightFull(highlight.SpansRichEdit(text), start, end)
}

func (ce *CodeEdit) applyHighlightFull(spans []highlight.Span, selStart, selEnd int) {
	if ce.highlighting {
		return
	}
	ce.highlighting = true
	wasSuppress := ce.suppress
	ce.suppress = true
	defer func() {
		ce.suppress = wasSuppress
		ce.highlighting = false
	}()

	win.SendMessage(ce.Handle(), win.WM_SETREDRAW, 0, 0)
	defer func() {
		win.SendMessage(ce.Handle(), win.WM_SETREDRAW, 1, 0)
		win.InvalidateRect(ce.Handle(), nil, true)
	}()

	ce.withUndoSuspended(func() {
		ce.SetTextSelection(0, -1)
		ce.applyDefaultFormat()
		ce.paintSpans(spans, 0, int(^uint(0)>>1))
		ce.SetTextSelection(selStart, selEnd)
		ce.setCodePara()
	})
}

func (ce *CodeEdit) applyHighlightRange(spans []highlight.Span, rangeStart, rangeEnd, selStart, selEnd int) {
	if ce.highlighting || rangeEnd <= rangeStart {
		return
	}
	ce.highlighting = true
	wasSuppress := ce.suppress
	ce.suppress = true
	defer func() {
		ce.suppress = wasSuppress
		ce.highlighting = false
	}()

	win.SendMessage(ce.Handle(), win.WM_SETREDRAW, 0, 0)
	defer func() {
		win.SendMessage(ce.Handle(), win.WM_SETREDRAW, 1, 0)
		win.InvalidateRect(ce.Handle(), nil, true)
	}()

	ce.withUndoSuspended(func() {
		ce.SetTextSelection(rangeStart, rangeEnd)
		ce.applyDefaultFormatSelection()
		ce.paintSpans(spans, rangeStart, rangeEnd)
		ce.SetTextSelection(selStart, selEnd)
	})
}

func (ce *CodeEdit) paintSpans(spans []highlight.Span, clipStart, clipEnd int) {
	for _, sp := range spans {
		if sp.End <= clipStart || sp.Start >= clipEnd {
			continue
		}
		a, b := sp.Start, sp.End
		if a < clipStart {
			a = clipStart
		}
		if b > clipEnd {
			b = clipEnd
		}
		if b <= a {
			continue
		}
		ce.SetTextSelection(a, b)
		var cf win.CHARFORMAT2
		cf.CbSize = uint32(unsafe.Sizeof(cf))
		cf.DwMask = win.CFM_COLOR
		cf.DwEffects = 0
		cf.CrTextColor = win.COLORREF(sp.Kind.ColorBGR())
		ce.SendMessage(win.EM_SETCHARFORMAT, win.SCF_SELECTION, uintptr(unsafe.Pointer(&cf)))
	}
}

func (ce *CodeEdit) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_GETDLGCODE:
		// כשחלון ההשלמה פתוח — כל המקשים (כולל Esc) מגיעים לעורך
		if ce.ac != nil && ce.ac.visible {
			return win.DLGC_WANTALLKEYS
		}
		if wParam == win.VK_RETURN || wParam == win.VK_ESCAPE || wParam == win.VK_TAB {
			return win.DLGC_WANTALLKEYS
		}
		// WANTTAB — Tab להזחה, לא מעבר בין שגיאות/פלט/כפתורים
		return win.DLGC_HASSETSEL | win.DLGC_WANTARROWS | win.DLGC_WANTCHARS | win.DLGC_WANTTAB

	case win.WM_KILLFOCUS:
		if ce.ac != nil {
			ce.ac.hide()
		}

	case win.WM_KEYDOWN:
		key := walk.Key(wParam)
		ce.ensureAutoComplete()
		// Escape — תמיד נסה לסגור השלמה (גם לפי קוד מקש גולמי)
		if wParam == win.VK_ESCAPE || key == walk.KeyEscape {
			if ce.ac != nil && ce.ac.visible {
				ce.ac.hide()
				if ce.acDebounce != nil {
					ce.acDebounce.Stop()
				}
				return 0
			}
		}
		if ce.ac != nil && ce.ac.handleKey(key) {
			return 0
		}
		if key == walk.KeyA && walk.ControlDown() {
			ce.SetTextSelection(0, -1)
			return 0
		}
		if key == walk.KeyZ && walk.ControlDown() && !walk.ShiftDown() {
			ce.Undo()
			return 0
		}
		if key == walk.KeyY && walk.ControlDown() {
			ce.Redo()
			return 0
		}
		if key == walk.KeyOEM2 && walk.ControlDown() { // Ctrl+/
			ce.toggleComment()
			return 0
		}
		if key == walk.KeyD && walk.ControlDown() && walk.ShiftDown() {
			ce.duplicateLines()
			return 0
		}
		if key == walk.KeySpace && walk.ControlDown() {
			ce.ignoreSpaceChar = true
			ce.forceCompletion()
			return 0
		}
		// Enter — הזחה אוטומטית (השלמה פתוחה כבר טופלה ב־handleKey)
		if key == walk.KeyReturn && !walk.ControlDown() && !walk.AltDown() {
			ce.insertNewlineAutoIndent()
			return 0
		}
		// Tab → השלמה אם פתוח, אחרת הזחה; Shift+Tab → החזרת הזחה
		if wParam == win.VK_TAB && !walk.ControlDown() && !walk.AltDown() {
			shift := walk.ShiftDown() || uint16(win.GetKeyState(win.VK_SHIFT))&0x8000 != 0
			if shift {
				if ce.ac != nil && ce.ac.visible {
					ce.ac.hide()
				}
				ce.unindentSelection()
				return 0
			}
			if ce.ac != nil && ce.ac.visible {
				ce.ac.accept()
				return 0
			}
			start, end := ce.TextSelection()
			if end < start {
				start, end = end, start
			}
			if end > start && ce.LineFromChar(start) != ce.LineFromChar(end-1) {
				ce.indentSelection()
			} else {
				// הזחה בתחילת השורה (לא רק ליד הסמן) — כדי ש־Shift+Tab יהפוך אותה
				ce.indentCurrentLine()
			}
			return 0
		}
		if walk.ControlDown() && (key == walk.KeyAdd || key == walk.KeyOEMPlus) {
			ce.zoom(+1)
			return 0
		}
		if walk.ControlDown() && (key == walk.KeySubtract || key == walk.KeyOEMMinus) {
			ce.zoom(-1)
			return 0
		}
		if walk.ControlDown() && key == walk.Key0 {
			codeFontSize = 14
			ce.rehighlight()
			if ce.onZoom != nil {
				ce.onZoom(codeFontSize)
			}
			return 0
		}

	case win.WM_MOUSEWHEEL:
		if ce.ac != nil && ce.ac.visible && !walk.ControlDown() {
			delta := int16(win.HIWORD(uint32(wParam)))
			steps := int(delta) / 120
			if steps == 0 {
				if delta > 0 {
					steps = 1
				} else {
					steps = -1
				}
			}
			ce.ac.scrollBy(-steps)
			return 0
		}
		if walk.ControlDown() {
			delta := int16(win.HIWORD(uint32(wParam)))
			if delta > 0 {
				ce.zoom(+1)
			} else if delta < 0 {
				ce.zoom(-1)
			}
			return 0
		}

	case win.WM_CHAR:
		if wParam == '\t' {
			return 0
		}
		// Ctrl+Space: לא להכניס רווח (גם אם Control כבר שוחרר ב־WM_CHAR)
		if wParam == ' ' && (ce.ignoreSpaceChar || walk.ControlDown()) {
			ce.ignoreSpaceChar = false
			return 0
		}
		if wParam < 32 && wParam != '\r' && wParam != '\n' {
			return 0
		}

	case win.WM_COMMAND:
		// EN_CHANGE מההורה (reflect) או מקומי
		if win.HIWORD(uint32(wParam)) == win.EN_CHANGE {
			ce.onTextEdited()
			return 0
		}

	case win.WM_NOTIFY:
		nmh := (*win.NMHDR)(unsafe.Pointer(lParam))
		if nmh != nil && nmh.Code == uint32(win.EN_SELCHANGE) {
			if ce.onSelChange != nil {
				ce.onSelChange()
			}
			return 0
		}
	}

	result := ce.WidgetBase.WndProc(hwnd, msg, wParam, lParam)

	// גיבוי רק לפעולות בלי EN_CHANGE אמין; הקלדה רגילה — רק EN_CHANGE (פעם אחת)
	switch msg {
	case win.WM_PASTE:
		ce.fullHLOnce = true
		ce.onTextEdited()
	case win.WM_CUT, win.WM_CLEAR:
		ce.onTextEdited()
	case win.WM_KEYDOWN:
		key := walk.Key(wParam)
		switch key {
		case walk.KeyLeft, walk.KeyRight, walk.KeyUp, walk.KeyDown, walk.KeyHome, walk.KeyEnd, walk.KeyPrior, walk.KeyNext:
			if ce.onSelChange != nil {
				ce.onSelChange()
			}
		}
	case win.WM_LBUTTONUP:
		if ce.onSelChange != nil {
			ce.onSelChange()
		}
	}
	return result
}

func (ce *CodeEdit) insertText(s string) {
	ce.insertTextRaw(s)
	ce.onTextEdited()
}

func (ce *CodeEdit) insertTextRaw(s string) {
	ptr := syscall.StringToUTF16Ptr(s)
	ce.SendMessage(win.EM_REPLACESEL, 1, uintptr(unsafe.Pointer(ptr)))
}

func (ce *CodeEdit) deleteSelection() {
	start, end := ce.TextSelection()
	if end == start {
		return
	}
	if end < start {
		start, end = end, start
	}
	ce.SetTextSelection(start, end)
	// WM_CLEAR אמין יותר מ־REPLACESEL עם מחרוזת ריקה ב־RichEdit
	ce.SendMessage(win.WM_CLEAR, 0, 0)
}

// selectionLineRange — טווח שורות להזחה (כולל שורת הסמן)
func (ce *CodeEdit) selectionLineRange() (first, last int) {
	start, end := ce.TextSelection()
	if end < start {
		start, end = end, start
	}
	first = ce.LineFromChar(start)
	if end > start {
		last = ce.LineFromChar(end - 1)
	} else {
		last = first
	}
	if last < first {
		last = first
	}
	return first, last
}

func (ce *CodeEdit) indentCurrentLine() {
	start, end := ce.TextSelection()
	if end < start {
		start, end = end, start
	}
	ls := ce.LineIndex(ce.LineFromChar(start))
	if ls < 0 {
		ce.insertText("  ")
		return
	}
	ce.SetTextSelection(ls, ls)
	ce.insertTextRaw("  ")
	ce.SetTextSelection(start+2, end+2)
	ce.onTextEdited()
}

func (ce *CodeEdit) indentSelection() {
	first, last := ce.selectionLineRange()
	start, end := ce.TextSelection()
	if end < start {
		start, end = end, start
	}
	added := 0
	for line := last; line >= first; line-- {
		ls := ce.LineIndex(line)
		if ls < 0 {
			continue
		}
		ce.SetTextSelection(ls, ls)
		ce.insertTextRaw("  ")
		added += 2
	}
	ce.SetTextSelection(start+2, end+added)
	ce.onTextEdited()
}

func (ce *CodeEdit) unindentSelection() {
	start, end := ce.TextSelection()
	if end < start {
		start, end = end, start
	}

	// בחירה מרובת־שורות — הסרה מתחילת כל שורה
	if end > start && ce.LineFromChar(start) != ce.LineFromChar(end-1) {
		ce.unindentLines()
		return
	}

	// 1) הפוך ל־Tab ישן: מחק עד 2 רווחים / טאב ממש לפני הסמן
	if n := ce.deleteIndentBeforeCaret(start); n > 0 {
		return
	}

	// 2) הסרה מתחילת השורה הנוכחית
	ce.unindentLines()
}

func (ce *CodeEdit) deleteIndentBeforeCaret(caret int) int {
	if caret <= 0 {
		return 0
	}
	text := ce.indexText()
	lineStart := ce.LineIndex(ce.LineFromChar(caret))
	n := indentCharsBefore(text, caret, lineStart)
	if n <= 0 {
		return 0
	}
	ce.SetTextSelection(caret-n, caret)
	ce.deleteSelection()
	ce.onTextEdited()
	return n
}

func (ce *CodeEdit) unindentLines() {
	first, last := ce.selectionLineRange()
	start, end := ce.TextSelection()
	if end < start {
		start, end = end, start
	}
	text := ce.indexText()

	type span struct{ from, n int }
	var spans []span
	for line := first; line <= last; line++ {
		ls := ce.LineIndex(line)
		if ls < 0 {
			continue
		}
		off, n := leadingIndentSpan(text, ls)
		if n > 0 {
			spans = append(spans, span{ls + off, n})
		}
	}
	if len(spans) == 0 {
		return
	}
	for i := len(spans) - 1; i >= 0; i-- {
		s := spans[i]
		ce.SetTextSelection(s.from, s.from+s.n)
		ce.deleteSelection()
	}
	adj := func(pos int) int {
		for _, s := range spans {
			if s.from+s.n <= pos {
				pos -= s.n
			} else if s.from < pos {
				pos = s.from
			}
		}
		if pos < 0 {
			return 0
		}
		return pos
	}
	ce.SetTextSelection(adj(start), adj(end))
	ce.onTextEdited()
}

// indentCharsBefore — כמה תווי UTF-16 למחוק לפני הסמן (בתוך אותה שורה)
func indentCharsBefore(text string, caretUTF16, lineStartUTF16 int) int {
	if caretUTF16 <= lineStartUTF16 {
		return 0
	}
	u := utf16.Encode([]rune(text))
	if caretUTF16 > len(u) {
		caretUTF16 = len(u)
	}
	if lineStartUTF16 < 0 {
		lineStartUTF16 = 0
	}
	i := caretUTF16 - 1
	for i >= lineStartUTF16 && isBidiMark(rune(u[i])) {
		i--
	}
	if i < lineStartUTF16 {
		return 0
	}
	if u[i] == '\t' {
		return caretUTF16 - i
	}
	n := 0
	j := i
	for n < 2 && j >= lineStartUTF16 && u[j] == ' ' {
		n++
		j--
	}
	if n == 0 {
		return 0
	}
	return caretUTF16 - (j + 1)
}

// leadingIndentSpan — היסט וכמות UTF-16 להסרה מתחילת שורה (טאב או עד 2 רווחים)
func leadingIndentSpan(text string, lineStartUTF16 int) (offset, n int) {
	bytePos := bytePosFromUTF16(text, lineStartUTF16)
	if bytePos >= len(text) {
		return 0, 0
	}
	u := utf16.Encode([]rune(text[bytePos:]))
	i := 0
	for i < len(u) && isBidiMark(rune(u[i])) {
		i++
	}
	if i >= len(u) {
		return 0, 0
	}
	if u[i] == '\t' {
		return i, 1
	}
	count := 0
	for count < 2 && i+count < len(u) && u[i+count] == ' ' {
		count++
	}
	return i, count
}

func (ce *CodeEdit) Undo() {
	if ce.historyUndo() {
		return
	}
	// נפילה: Undo מקורי (אם נשאר משהו שימושי)
	before := stripBidiMarks(ce.rawText())
	for i := 0; i < 800; i++ {
		if ce.SendMessage(win.EM_CANUNDO, 0, 0) == 0 {
			break
		}
		if ce.SendMessage(win.EM_UNDO, 0, 0) == 0 {
			break
		}
		if stripBidiMarks(ce.rawText()) != before {
			ce.resetHistoryFromCurrent()
			ce.onTextEdited()
			if ce.onSelChange != nil {
				ce.onSelChange()
			}
			return
		}
	}
}

func (ce *CodeEdit) Redo() {
	if ce.historyRedo() {
		return
	}
	before := stripBidiMarks(ce.rawText())
	for i := 0; i < 800; i++ {
		if ce.SendMessage(win.EM_CANREDO, 0, 0) == 0 {
			break
		}
		if ce.SendMessage(win.EM_REDO, 0, 0) == 0 {
			break
		}
		if stripBidiMarks(ce.rawText()) != before {
			ce.resetHistoryFromCurrent()
			ce.onTextEdited()
			if ce.onSelChange != nil {
				ce.onSelChange()
			}
			return
		}
	}
}

const (
	frDown      = 0x00000001
	frMatchCase = 0x00000004
)

// findTextEx — תואם FINDTEXTEXW (שדות לא מיוצאים ב־lxn/win)
type findTextEx struct {
	chrg      win.CHARRANGE
	lpstrText *uint16
	chrgText  win.CHARRANGE
}

func (ce *CodeEdit) FindNext(needle string, matchCase bool) bool {
	if needle == "" {
		return false
	}
	_, end := ce.TextSelection()
	if ce.findInRange(needle, matchCase, int32(end), -1) {
		return true
	}
	return ce.findInRange(needle, matchCase, 0, int32(end))
}

func (ce *CodeEdit) findInRange(needle string, matchCase bool, min, max int32) bool {
	ptr := syscall.StringToUTF16Ptr(needle)
	ft := findTextEx{
		chrg:      win.CHARRANGE{CpMin: min, CpMax: max},
		lpstrText: ptr,
	}
	flags := uintptr(frDown)
	if matchCase {
		flags |= frMatchCase
	}
	r := ce.SendMessage(win.EM_FINDTEXTEXW, flags, uintptr(unsafe.Pointer(&ft)))
	if int32(r) < 0 {
		return false
	}
	ce.SetTextSelection(int(ft.chrgText.CpMin), int(ft.chrgText.CpMax))
	ce.ScrollCaret()
	if ce.onSelChange != nil {
		ce.onSelChange()
	}
	return true
}

func (ce *CodeEdit) ReplaceSelection(repl string) {
	ce.insertText(repl)
}

func (ce *CodeEdit) ReplaceAll(needle, repl string, matchCase bool) int {
	if needle == "" {
		return 0
	}
	count := 0
	pos := int32(0)
	for count < 50000 {
		if !ce.findInRange(needle, matchCase, pos, -1) {
			break
		}
		start, _ := ce.TextSelection()
		ce.insertTextRaw(repl)
		count++
		pos = int32(start + utf16Len(repl))
		ce.SetTextSelection(int(pos), int(pos))
	}
	if count > 0 {
		ce.onTextEdited()
	}
	return count
}

func (ce *CodeEdit) lineRangeUTF16(first, last int) (start, end int) {
	start = ce.LineIndex(first)
	if start < 0 {
		start = 0
	}
	if last+1 < ce.LineCount() {
		end = ce.LineIndex(last + 1)
	} else {
		end = utf16Len(ce.indexText())
	}
	return start, end
}

func (ce *CodeEdit) toggleComment() {
	first, last := ce.selectionLineRange()
	start, end := ce.lineRangeUTF16(first, last)
	text := ce.indexText()
	chunk := substringUTF16(text, start, end)
	// שמירת ירידת שורה סופית אם הייתה
	trailingCR := strings.HasSuffix(chunk, "\r")
	body := chunk
	if trailingCR {
		body = chunk[:len(chunk)-1]
	}
	lines := strings.Split(body, "\r")
	newLines := toggleCommentLines(lines)
	repl := strings.Join(newLines, "\r")
	if trailingCR {
		repl += "\r"
	}
	ce.SetTextSelection(start, end)
	ce.insertText(repl)
	ce.SetTextSelection(start, start+utf16Len(repl))
}

func (ce *CodeEdit) duplicateLines() {
	first, last := ce.selectionLineRange()
	start, end := ce.lineRangeUTF16(first, last)
	text := ce.indexText()
	chunk := substringUTF16(text, start, end)
	if chunk == "" {
		return
	}
	insert := chunk
	if !strings.HasSuffix(insert, "\r") {
		// סוף קובץ בלי ירידת שורה — הוסף מפריד לפני השכפול
		insert = "\r" + insert
	}
	ce.SetTextSelection(end, end)
	ce.insertText(insert)
}

func (ce *CodeEdit) currentLineText() string {
	start, _ := ce.TextSelection()
	line := ce.LineFromChar(start)
	ls := ce.LineIndex(line)
	var le int
	if line+1 < ce.LineCount() {
		le = ce.LineIndex(line + 1)
		// בלי ה־\r
		if le > ls {
			le--
		}
	} else {
		le = utf16Len(ce.indexText())
	}
	return substringUTF16(ce.indexText(), ls, le)
}

func (ce *CodeEdit) insertNewlineAutoIndent() {
	line := ce.currentLineText()
	indent := leadingSpacesOfLine(line)
	if lineSuggestsBlockIndent(line) {
		indent += "  "
	}
	ce.insertText("\r" + indent)
}

func (ce *CodeEdit) zoom(delta int) {
	codeFontSize += delta
	if codeFontSize < 10 {
		codeFontSize = 10
	}
	if codeFontSize > 28 {
		codeFontSize = 28
	}
	ce.rehighlight()
	if ce.onZoom != nil {
		ce.onZoom(codeFontSize)
	}
}

func (*CodeEdit) CreateLayoutItem(ctx *walk.LayoutContext) walk.LayoutItem {
	return walk.NewGreedyLayoutItem()
}

func stripBidiMarks(s string) string {
	return strings.Map(func(r rune) rune {
		if isBidiMark(r) {
			return -1
		}
		return r
	}, s)
}
