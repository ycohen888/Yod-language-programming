//go:build windows

package editor

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

const histMax = 200

type textSnap struct {
	text  string
	start int
	end   int
}

// קבועי TOM — השעיית Undo המקורי בזמן צביעה
const (
	tomSuspend int32 = -9999995
	tomResume  int32 = -9999994
)

var iidITextDocument = win.IID{
	Data1: 0x8CC497C0,
	Data2: 0xA1DF,
	Data3: 0x11CE,
	Data4: [8]byte{0x80, 0x98, 0x00, 0xAA, 0x00, 0x61, 0x44, 0x5E},
}

func (ce *CodeEdit) snapNow() textSnap {
	start, end := ce.TextSelection()
	return textSnap{text: ce.indexText(), start: start, end: end}
}

func (ce *CodeEdit) clearHistory() {
	ce.undoStack = nil
	ce.redoStack = nil
	ce.lastSnap = textSnap{}
	ce.haveLast = false
}

func (ce *CodeEdit) resetHistoryFromCurrent() {
	ce.undoStack = nil
	ce.redoStack = nil
	ce.lastSnap = ce.snapNow()
	ce.haveLast = true
}

func (ce *CodeEdit) recordHistory() {
	if ce.histLock || ce.suppress || ce.highlighting {
		return
	}
	// בתחילת רצף הקלדה — שומרים נקודת Undo אחת (לפני השינויים)
	if !ce.histBurst {
		if ce.haveLast {
			ce.undoStack = append(ce.undoStack, ce.lastSnap)
			if len(ce.undoStack) > histMax {
				ce.undoStack = ce.undoStack[len(ce.undoStack)-histMax:]
			}
		}
		ce.histBurst = true
		ce.redoStack = nil
	}
	if ce.histDebounce != nil {
		ce.histDebounce.Stop()
	}
	ce.histDebounce = time.AfterFunc(historyIdle, func() {
		ce.Synchronize(func() {
			if ce.suppress {
				return
			}
			ce.lastSnap = ce.snapNow()
			ce.haveLast = true
			ce.histBurst = false
		})
	})
}

func (ce *CodeEdit) flushHistory() {
	if ce.histDebounce != nil {
		ce.histDebounce.Stop()
		ce.histDebounce = nil
	}
	if !ce.histBurst {
		return
	}
	ce.lastSnap = ce.snapNow()
	ce.haveLast = true
	ce.histBurst = false
}

func (ce *CodeEdit) applySnap(s textSnap) {
	ce.histLock = true
	ce.suppress = true
	defer func() {
		ce.suppress = false
		ce.histLock = false
	}()

	ptr := syscall.StringToUTF16Ptr(s.text)
	ce.SendMessage(win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(ptr)))
	ce.applyDefaultFormat()
	ce.setCodePara()
	if s.start < 0 {
		s.start = 0
	}
	if s.end < s.start {
		s.end = s.start
	}
	ce.SetTextSelection(s.start, s.end)
	ce.lastSnap = s
	ce.haveLast = true

	// צביעה מחדש בלי להיכנס להיסטוריה
	ce.rehighlight()
	ce.ScrollCaret()
	if ce.onSelChange != nil {
		ce.onSelChange()
	}
	ce.textChangedPublisher.Publish()
}

func (ce *CodeEdit) historyUndo() bool {
	ce.flushHistory()
	if len(ce.undoStack) == 0 {
		return false
	}
	if ce.haveLast {
		ce.redoStack = append(ce.redoStack, ce.lastSnap)
	}
	prev := ce.undoStack[len(ce.undoStack)-1]
	ce.undoStack = ce.undoStack[:len(ce.undoStack)-1]
	ce.applySnap(prev)
	return true
}

func (ce *CodeEdit) historyRedo() bool {
	ce.flushHistory()
	if len(ce.redoStack) == 0 {
		return false
	}
	if ce.haveLast {
		ce.undoStack = append(ce.undoStack, ce.lastSnap)
	}
	next := ce.redoStack[len(ce.redoStack)-1]
	ce.redoStack = ce.redoStack[:len(ce.redoStack)-1]
	ce.applySnap(next)
	return true
}

func (ce *CodeEdit) textDocument() *win.ITextDocument {
	var punk unsafe.Pointer
	if ce.SendMessage(win.EM_GETOLEINTERFACE, 0, uintptr(unsafe.Pointer(&punk))) == 0 || punk == nil {
		return nil
	}
	ole := (*win.IRichEditOle)(punk)
	var pdoc unsafe.Pointer
	hr := ole.QueryInterface(&iidITextDocument, &pdoc)
	if hr != win.S_OK || pdoc == nil {
		hr = ole.QueryInterface(&win.IID_ITextDocument, &pdoc)
	}
	ole.Release()
	if hr != win.S_OK || pdoc == nil {
		return nil
	}
	return (*win.ITextDocument)(pdoc)
}

func (ce *CodeEdit) withUndoSuspended(fn func()) {
	// חשוב: לא קוראים ל־Freeze/TOM כאן — זה שיבש הקלדה ב־msftedit.
	// יש מחסנית Undo משלנו (recordHistory), ולכן אין צורך להשעות את Undo של RichEdit.
	fn()
}
