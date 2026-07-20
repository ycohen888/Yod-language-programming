//go:build windows

package stdlib

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	xdraw "golang.org/x/image/draw"

	"yod/internal/draw2d"
	"yod/internal/object"
)

const canvasUndoMax = 40

func addChildControl(st *controlState, a []object.Object, owner string) object.Object {
	if len(a) != 1 {
		return errObj(owner + ".הוסף מצפה לרכיב אחד")
	}
	gw, ok := a[0].(*object.GuiWidget)
	if !ok {
		return errObj(owner + ".הוסף מצפה לרכיב ממשק")
	}
	cs, ok := gw.Data.(*controlState)
	if !ok {
		return errObj(owner + ".הוסף: רכיב לא תקין")
	}
	st.children = append(st.children, cs)
	return object.Nil
}

// חלונות.שורה() — מקבץ רכיבים אופקית (סרגל כלים / צבעים)
func winCreateRow(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלונות.שורה מצפה ל־0 ארגומנטים")
	}
	st := &controlState{kind: "שורה", children: nil}
	w := &object.GuiWidget{Kind: "שורה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["הוסף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return addChildControl(st, a, "שורה")
	}}
	return w
}

// חלונות.עמודה() — סרגל כלים אנכי
func winCreateColumn(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלונות.עמודה מצפה ל־0 ארגומנטים")
	}
	st := &controlState{kind: "עמודה", children: nil}
	w := &object.GuiWidget{Kind: "עמודה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["הוסף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return addChildControl(st, a, "עמודה")
	}}
	return w
}

// חלונות.מסגרת(כיוון) — "אופקי" | "אנכי", בלי הגבלת גובה
func winCreateFrame(args ...object.Object) object.Object {
	dir := "אנכי"
	if len(args) >= 1 {
		s, ok := asString(args[0])
		if !ok {
			return errObj("חלונות.מסגרת מצפה לכיוון מחרוזת: אופקי / אנכי")
		}
		switch s {
		case "אופקי", "אנכי":
			dir = s
		default:
			return errObj("חלונות.מסגרת: כיוון חייב להיות אופקי או אנכי")
		}
	}
	if len(args) > 1 {
		return errObj("חלונות.מסגרת מצפה ל־0 או 1 ארגומנטים")
	}
	st := &controlState{kind: "מסגרת", frameDir: dir, children: nil, stretchFactor: -1}
	w := &object.GuiWidget{Kind: "מסגרת", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["הוסף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return addChildControl(st, a, "מסגרת")
	}}
	w.Attrs["קבע_רקע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c, err := parseWalkColor(a...)
		if err != nil {
			return errObj("מסגרת.קבע_רקע: " + err.Error())
		}
		st.bgColor = c
		st.hasBg = true
		return object.Nil
	}}
	w.Attrs["קבע_מתיחה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setStretchFactor(st, "מסגרת.קבע_מתיחה", a...)
	}}
	attachVisible(w, st, "מסגרת")
	return w
}

func attachClick(w *object.GuiWidget, st *controlState) {
	w.Attrs["בלחיצה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("בלחיצה מצפה לפונקציה אחת")
		}
		switch a[0].(type) {
		case *object.Function, *object.Closure, *object.CompiledFunction:
			st.onClick = a[0]
			return object.Nil
		default:
			return errObj("בלחיצה מצפה לפונקציה")
		}
	}}
}

// חלונות.דגם(צבע…) — ריבוע צבע לחיץ (לוח צבעים)
func winCreateSwatch(args ...object.Object) object.Object {
	c, err := parseDrawColor(args...)
	if err != nil {
		return errObj("חלונות.דגם: " + err.Error())
	}
	st := &controlState{kind: "דגם", swatchColor: c}
	w := &object.GuiWidget{Kind: "דגם", Data: st, Attrs: map[string]object.Object{}}
	attachClick(w, st)
	return w
}

// חלונות.סמל(סוג) — כפתור כלי עם אייקון מצויר
func winCreateIcon(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חלונות.סמל מצפה לשם כלי (עיפרון/מחק/קו/…)")
	}
	name, ok := asString(args[0])
	if !ok || name == "" {
		return errObj("חלונות.סמל מצפה למחרוזת")
	}
	st := &controlState{kind: "סמל", iconKind: name}
	w := &object.GuiWidget{Kind: "סמל", Data: st, Attrs: map[string]object.Object{}}
	attachClick(w, st)
	return w
}

// חלונות.משטח(רוחב, גובה) — לוח ציור אינטראקטיבי לעכבר (הלוגיקה ביוד)
func winCreateCanvas(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלונות.משטח מצפה לרוחב וגובה")
	}
	wn, ok1 := args[0].(*object.Number)
	hn, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("חלונות.משטח מצפה למספרים")
	}
	ww, hh := int(wn.Value), int(hn.Value)
	if ww < 40 || hh < 40 || ww > 4000 || hh > 4000 {
		return errObj("גודל משטח לא תקין (40–4000)")
	}
	dpi := screenDPI()
	pw, ph := dipToPixels(ww, dpi), dipToPixels(hh, dpi)
	if pw < 40 {
		pw = 40
	}
	if ph < 40 {
		ph = 40
	}
	if pw > 8000 {
		pw = 8000
	}
	if ph > 8000 {
		ph = 8000
	}
	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	dpr := float64(dpi) / 96.0
	if dpr < 1 {
		dpr = 1
	}
	backend := draw2d.NewBackend(pw, ph, dpi)
	board := &drawBoard{
		img:     backend.Buffer(),
		backend: backend,
		stroke:  color.RGBA{0, 0, 0, 255},
		fill:    color.RGBA{200, 200, 200, 255},
		width:   2,
		fontSz:  16,
		dpr:     dpr,
	}
	if board.img == nil {
		board.img = img
	}
	// אימות חד־פעמי בקונסול — כדי לדעת שהבינארי החדש רץ
	fmt.Fprintf(os.Stderr, "יוד: משטח באקאנד=%s (%dx%d @%d)\n", backend.Name(), pw, ph, dpi)
	st := &controlState{
		kind:          "משטח",
		board:         board,
		canvasW:       ww,
		canvasH:       hh,
		canvasDipW:    ww,
		canvasDipH:    hh,
		dragging:      false,
		undoStack:     nil,
		canvasLockH:   true,
		stretchFactor: -1,
	}
	w := &object.GuiWidget{Kind: "משטח", Data: st, Attrs: map[string]object.Object{}}

	w.Attrs["בעכבר_למטה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseDown, "בעכבר_למטה", a)
	}}
	w.Attrs["בעכבר_גרירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseDrag, "בעכבר_גרירה", a)
	}}
	w.Attrs["בעכבר_למעלה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseUp, "בעכבר_למעלה", a)
	}}
	w.Attrs["בעכבר_זוז"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseHover, "בעכבר_זוז", a)
	}}
	w.Attrs["בעכבר_גלגל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseWheel, "בעכבר_גלגל", a)
	}}
	w.Attrs["בעת_תו"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onKeyChar, "בעת_תו", a)
	}}
	w.Attrs["בעת_מקש"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onKeyCmd, "בעת_מקש", a)
	}}
	w.Attrs["בשינוי_גודל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onSizeChange, "בשינוי_גודל", a)
	}}
	w.Attrs["בקש_מיקוד"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if st.canvas != nil {
			_ = st.canvas.SetFocus()
		}
		return object.Nil
	}}
	w.Attrs["קבע_רמז"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setSurfaceHint(st, a...)
	}}
	w.Attrs["קבע_מתיחה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setStretchFactor(st, "משטח.קבע_מתיחה", a...)
	}}
	w.Attrs["קבע_נעילת_גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("משטח.קבע_נעילת_גובה מצפה לערך בוליאני")
		}
		on, errV := parseDarkBool("משטח.קבע_נעילת_גובה", a...)
		if errV != nil {
			return errV
		}
		st.canvasLockH = on
		return object.Nil
	}}
	w.Attrs["קבע_נעילת_רוחב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("משטח.קבע_נעילת_רוחב מצפה לערך בוליאני")
		}
		on, errV := parseDarkBool("משטח.קבע_נעילת_רוחב", a...)
		if errV != nil {
			return errV
		}
		st.canvasLockW = on
		return object.Nil
	}}
	w.Attrs["רוחב_נוכחי"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(st.canvasW)}
	}}
	w.Attrs["גובה_נוכחי"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(st.canvasH)}
	}}

	w.Attrs["קבע_צבע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c, err := parseDrawColor(a...)
		if err != nil {
			return errObj(err.Error())
		}
		board.stroke = c
		return object.Nil
	}}
	w.Attrs["קבע_מילוי"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c, err := parseDrawColor(a...)
		if err != nil {
			return errObj(err.Error())
		}
		board.fill = c
		return object.Nil
	}}
	w.Attrs["קבע_עובי"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("קבע_עובי מצפה למספר")
		}
		n, ok := a[0].(*object.Number)
		if !ok {
			return errObj("קבע_עובי מצפה למספר")
		}
		board.width = int(n.Value)
		if board.width < 1 {
			board.width = 1
		}
		return object.Nil
	}}
	w.Attrs["קבע_גודל_גופן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("קבע_גודל_גופן מצפה למספר")
		}
		n, ok := a[0].(*object.Number)
		if !ok {
			return errObj("קבע_גודל_גופן מצפה למספר")
		}
		board.fontSz = n.Value
		if board.fontSz < 8 {
			board.fontSz = 8
		}
		board.face = nil
		return object.Nil
	}}
	w.Attrs["קבע_יישור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setBoardTextAlign(board, a...)
	}}
	w.Attrs["קרא_יישור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardReadTextAlign(board)
	}}

	w.Attrs["צלם"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		var snap *image.RGBA
		if board.backend != nil {
			if s, err := board.backend.SnapshotRGBA(); err == nil {
				snap = s
			}
		}
		if snap == nil {
			snap = cloneRGBA(board.img)
		}
		st.undoStack = append(st.undoStack, snap)
		if len(st.undoStack) > canvasUndoMax {
			st.undoStack = st.undoStack[len(st.undoStack)-canvasUndoMax:]
		}
		return object.Nil
	}}
	w.Attrs["בטל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(st.undoStack) == 0 {
			return object.Nil
		}
		last := st.undoStack[len(st.undoStack)-1]
		st.undoStack = st.undoStack[:len(st.undoStack)-1]
		if board.backend != nil {
			_ = board.backend.ReplacePixels(last)
			board.syncImgFromBackend()
		} else {
			draw.Draw(board.img, board.img.Bounds(), last, last.Bounds().Min, draw.Src)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["גיבוי"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if board.backend != nil {
			if s, err := board.backend.SnapshotRGBA(); err == nil {
				st.backup = s
				return object.Nil
			}
		}
		st.backup = cloneRGBA(board.img)
		return object.Nil
	}}
	w.Attrs["שחזר"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if st.backup == nil {
			return object.Nil
		}
		if board.backend != nil {
			_ = board.backend.ReplacePixels(st.backup)
			board.syncImgFromBackend()
		} else {
			draw.Draw(board.img, board.img.Bounds(), st.backup, st.backup.Bounds().Min, draw.Src)
		}
		invalidateCanvas(st)
		return object.Nil
	}}

	w.Attrs["נקה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c := color.RGBA{255, 255, 255, 255}
		if len(a) > 0 {
			parsed, err := parseDrawColor(a...)
			if err != nil {
				return errObj(err.Error())
			}
			c = parsed
		}
		board.clearBackend(c)
		// אצווה: ציורים עד רענן() בלי Invalidate חוזר ונשנה
		st.canvasBatch = true
		st.canvasDirty = true
		return object.Nil
	}}
	w.Attrs["נקודה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, err := twoInts("נקודה", a)
		if err != nil {
			return err
		}
		x, y = board.sp(x), board.sp(y)
		r := board.strokePx() / 2
		if board.backend != nil {
			board.backend.FillCircle(x, y, r, board.stroke)
			if board.strokePx() < 2 {
				board.backend.FillRect(x, y, 1, 1, board.stroke)
			}
			board.syncImgFromBackend()
		} else {
			drawDisk(board.img, x, y, r, board.stroke)
			if board.strokePx() < 2 {
				board.img.Set(x, y, board.stroke)
			}
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["קו"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 4 {
			return errObj("קו מצפה ל־x1, y1, x2, y2")
		}
		vals, err := nums("קו", a, 4)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.StrokeLine(vals[0], vals[1], vals[2], vals[3], board.strokePx(), board.stroke)
			board.syncImgFromBackend()
		} else {
			drawThickLine(board.img, vals[0], vals[1], vals[2], vals[3], board.strokePx(), board.stroke)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלבן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן", a, 4)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.StrokeRect(vals[0], vals[1], vals[2], vals[3], board.strokePx(), board.stroke)
			board.syncImgFromBackend()
		} else {
			drawRectOutline(board.img, vals[0], vals[1], vals[2], vals[3], board.strokePx(), board.stroke)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלבן_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן_מלא", a, 4)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.FillRect(vals[0], vals[1], vals[2], vals[3], board.fill)
			board.syncImgFromBackend()
		} else {
			drawRectFill(board.img, vals[0], vals[1], vals[2], vals[3], board.fill)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלבן_מעוגל_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן_מעוגל_מלא", a, 5)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.FillRoundedRect(vals[0], vals[1], vals[2], vals[3], vals[4], board.fill)
			board.syncImgFromBackend()
		} else {
			drawRoundedRectFill(board.img, vals[0], vals[1], vals[2], vals[3], vals[4], board.fill)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["עיגול"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("עיגול", a, 3)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.StrokeCircle(vals[0], vals[1], vals[2], board.strokePx(), board.stroke)
			board.syncImgFromBackend()
		} else {
			drawCircleOutline(board.img, vals[0], vals[1], vals[2], board.strokePx(), board.stroke)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["עיגול_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("עיגול_מלא", a, 3)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.FillCircle(vals[0], vals[1], vals[2], board.fill)
			board.syncImgFromBackend()
		} else {
			drawDisk(board.img, vals[0], vals[1], vals[2], board.fill)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["אליפסה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("אליפסה", a, 4)
		if err != nil {
			return err
		}
		vals = board.scaleVals(vals)
		if board.backend != nil {
			board.backend.StrokeEllipse(vals[0], vals[1], vals[2], vals[3], board.strokePx(), board.stroke)
			board.syncImgFromBackend()
		} else {
			drawEllipseOutline(board.img, vals[0], vals[1], vals[2], vals[3], board.strokePx(), board.stroke)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלא_באזור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, err := twoInts("מלא_באזור", a)
		if err != nil {
			return err
		}
		if board.backend != nil {
			board.backend.FloodFill(board.sp(x), board.sp(y), board.stroke)
			board.syncImgFromBackend()
		} else {
			floodFill(board.img, board.sp(x), board.sp(y), board.stroke)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 3 {
			return errObj("טקסט מצפה למחרוזת, x, y")
		}
		s, ok := asString(a[0])
		if !ok {
			return errObj("טקסט מצפה למחרוזת, x, y")
		}
		vals, err := nums("טקסט", a[1:], 2)
		if err != nil {
			return err
		}
		if err := drawTextOnBoard(board, s, vals[0], vals[1]); err != nil {
			return errObj(err.Error())
		}
		if board.backend != nil {
			fs := board.fontSz
			if board.dpr > 1.001 {
				fs *= board.dpr
			}
			board.backend.DrawText(s, board.sp(vals[0]), board.sp(vals[1]), fs, board.stroke, board.align)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["טקסט_בתיבה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		res := boardDrawTextBox(board, a...)
		if res != nil && res.Type() == object.ErrorObj {
			return res
		}
		board.syncImgFromBackend()
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["רוחב_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardTextWidth(board, a...)
	}}
	w.Attrs["מיקום_סמן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardTextCaretInset(board, a...)
	}}
	drawImg := &object.Builtin{Fn: func(a ...object.Object) object.Object {
		res := boardDrawImage(board, a...)
		if res != nil && res.Type() == object.ErrorObj {
			return res
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["תמונה"] = drawImg
	w.Attrs["צייר_תמונה"] = drawImg
	w.Attrs["קרא_צבע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, err := twoInts("קרא_צבע", a)
		if err != nil {
			return err
		}
		x, y = board.sp(x), board.sp(y)
		board.syncImgFromBackend()
		b := board.img.Bounds()
		if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
			return errObj("קרא_צבע: נקודה מחוץ למשטח")
		}
		c := board.img.RGBAAt(x, y)
		return &object.Hash{Pairs: map[string]object.Object{
			"אדום":   &object.Number{Value: float64(c.R)},
			"ירוק":   &object.Number{Value: float64(c.G)},
			"כחול":   &object.Number{Value: float64(c.B)},
			"שקיפות": &object.Number{Value: float64(c.A)},
		}}
	}}
	w.Attrs["טען"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("טען מצפה לנתיב")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("טען מצפה לנתיב מחרוזת")
		}
		src, err := loadImageFile(path)
		if err != nil {
			return errObj(err.Error())
		}
		dst := board.img.Bounds()
		xdraw.CatmullRom.Scale(board.img, dst, src, src.Bounds(), draw.Src, nil)
		if board.backend != nil {
			_ = board.backend.ReplacePixels(board.img)
		}
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["שמור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("שמור מצפה לנתיב")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("שמור מצפה לנתיב מחרוזת")
		}
		board.syncImgFromBackend()
		src := board.img
		if board.backend != nil {
			if snap, err := board.backend.SnapshotRGBA(); err == nil {
				src = snap
			}
		}
		if err := saveImageFile(src, path); err != nil {
			return errObj(err.Error())
		}
		return object.Nil
	}}
	w.Attrs["רענן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		st.canvasDirty = true
		flushCanvasInvalidate(st)
		return object.Nil
	}}
	w.Attrs["השהה_הצגה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		// ציורים הבאים בלי Invalidate עד רענן — להקלדה מהירה / רענון חלקי
		st.canvasBatch = true
		st.canvasDirty = true
		return object.Nil
	}}
	w.Attrs["רוחב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(st.canvasW)}
	}}
	w.Attrs["גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(st.canvasH)}
	}}
	return w
}

func setMouseHandler(dst *object.Object, name string, a []object.Object) object.Object {
	if len(a) != 1 {
		return errObj(name + " מצפה לפונקציה אחת")
	}
	switch a[0].(type) {
	case *object.Function, *object.Closure, *object.CompiledFunction:
		*dst = a[0]
		return object.Nil
	default:
		return errObj(name + " מצפה לפונקציה")
	}
}

func invalidateCanvas(st *controlState) {
	if st == nil {
		return
	}
	st.canvasDirty = true
	if st.canvasBatch {
		return
	}
	flushCanvasInvalidate(st)
}

func flushCanvasInvalidate(st *controlState) {
	if st == nil || st.canvas == nil {
		return
	}
	st.canvasDirty = false
	st.canvasBatch = false
	st.canvas.Invalidate()
}

func mapSurfaceMouse(ch *controlState, x, y int) (int, int) {
	if ch == nil || ch.canvas == nil || ch.board == nil || ch.board.img == nil {
		return x, y
	}
	b := ch.canvas.ClientBoundsPixels()
	if b.Width < 1 || b.Height < 1 {
		return x, y
	}
	iw := ch.board.img.Bounds().Dx()
	ih := ch.board.img.Bounds().Dy()
	if iw < 1 || ih < 1 {
		return x, y
	}
	// עכבר בפיקסלים טבעיים → קואורדינטות בלוח הפיזי
	nx := x * iw / b.Width
	ny := y * ih / b.Height
	// ואז חזרה ל־DIP ללוגיקת יוד
	if ch.board.dpr > 1.001 {
		nx = int(float64(nx)/ch.board.dpr + 0.5)
		ny = int(float64(ny)/ch.board.dpr + 0.5)
	}
	if nx < 0 {
		nx = 0
	}
	if ny < 0 {
		ny = 0
	}
	maxX, maxY := ch.canvasW, ch.canvasH
	if maxX < 1 {
		maxX = iw
	}
	if maxY < 1 {
		maxY = ih
	}
	if nx >= maxX {
		nx = maxX - 1
	}
	if ny >= maxY {
		ny = maxY - 1
	}
	return nx, ny
}

func invokeYodMouse(fn object.Object, x, y int, button string) {
	if fn == nil {
		return
	}
	args := []object.Object{numObj(x), numObj(y)}
	if mouseHandlerArity(fn) >= 3 {
		args = append(args, &object.String{Value: button})
	}
	invokeYod(fn, args)
}

func mouseHandlerArity(fn object.Object) int {
	switch f := fn.(type) {
	case *object.Function:
		return len(f.Parameters)
	case *object.Closure:
		if f.Fn != nil {
			return f.Fn.NumParameters
		}
	case *object.CompiledFunction:
		return f.NumParameters
	}
	return 2
}

func walkButtonName(button walk.MouseButton) string {
	switch button {
	case walk.LeftButton:
		return mouseLeft
	case walk.RightButton:
		return mouseRight
	case walk.MiddleButton:
		return mouseMiddle
	default:
		return mouseOther
	}
}

func numObj(v int) object.Object {
	return &object.Number{Value: float64(v)}
}

func winFileSave(args ...object.Object) object.Object {
	return winFileDialog(true, args...)
}

func winFileOpen(args ...object.Object) object.Object {
	return winFileDialog(false, args...)
}

func winBrowseFolder(args ...object.Object) object.Object {
	title := "בחירת תיקייה"
	if len(args) > 1 {
		return errObj("חלונות.בחר_תיקייה מצפה ל־0 או 1 ארגומנטים")
	}
	if len(args) == 1 {
		if s, ok := asString(args[0]); ok && s != "" {
			title = s
		} else if args[0] != nil && args[0].Type() != object.NullObj {
			return errObj("חלונות.בחר_תיקייה: כותרת חייבת להיות מחרוזת")
		}
	}
	path, ok, err := pickFolderPath(title)
	if err != nil {
		owner := walk.App().ActiveForm()
		walk.MsgBox(owner, "בחירת תיקייה", "לא ניתן לפתוח דיאלוג תיקייה:\n"+err.Error(), walk.MsgBoxIconError)
		return object.Nil
	}
	if !ok || path == "" {
		return object.Nil
	}
	return &object.String{Value: path}
}

func winFileDialog(save bool, args ...object.Object) object.Object {
	title := "בחירת קובץ"
	filter := "תמונות (*.png;*.jpg;*.jpeg;*.gif;*.bmp)|*.png;*.jpg;*.jpeg;*.gif;*.bmp|כל הקבצים (*.*)|*.*"
	defaultName := ""
	imageDefaults := false
	if save {
		title = "שמירת קובץ"
		filter = "PNG (*.png)|*.png|JPEG (*.jpg)|*.jpg|כל הקבצים (*.*)|*.*"
		defaultName = "ציור.png"
		imageDefaults = true
	}
	if len(args) >= 1 {
		if s, ok := asString(args[0]); ok && s != "" {
			title = s
		} else if args[0] != nil && args[0].Type() != object.NullObj {
			return errObj("כותרת הדיאלוג חייבת להיות מחרוזת")
		}
	}
	if len(args) >= 2 {
		if s, ok := asString(args[1]); ok && s != "" {
			filter = s
			if imageDefaults {
				defaultName = ""
				imageDefaults = false
			}
		}
	}
	if len(args) >= 3 {
		if s, ok := asString(args[2]); ok {
			defaultName = s
		} else if args[2] != nil && args[2].Type() != object.NullObj {
			return errObj("שם קובץ ברירת מחדל חייב להיות מחרוזת")
		}
	}
	if len(args) > 3 {
		return errObj("דיאלוג קובץ מצפה ל־0–3 ארגומנטים (כותרת, סינון, שם)")
	}
	dlg := new(walk.FileDialog)
	dlg.Title = title
	dlg.Filter = filter
	if save && defaultName != "" {
		dlg.FilePath = defaultName
	}
	var ok bool
	var err error
	if save {
		ok, err = dlg.ShowSave(nil)
	} else {
		ok, err = dlg.ShowOpen(nil)
	}
	if err != nil {
		return errObj(err.Error())
	}
	if !ok {
		return object.Nil
	}
	path := dlg.FilePath
	if save && filepath.Ext(path) == "" {
		ext := extensionFromFileFilter(filter)
		if ext == "" && imageDefaults {
			ext = ".png"
		}
		if ext != "" {
			path += ext
		}
	}
	return &object.String{Value: path}
}

// extensionFromFileFilter מחזיר סיומת ראשונה מסינון Windows (למשל *.json → .json).
func extensionFromFileFilter(filter string) string {
	for _, part := range strings.Split(filter, "|") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "*.") || part == "*.*" {
			continue
		}
		first := strings.Split(part, ";")[0]
		ext := strings.TrimPrefix(strings.TrimSpace(first), "*")
		if ext != "" && ext != ".*" && !strings.ContainsAny(ext, "*?") {
			return ext
		}
	}
	return ""
}

func buildControlWidget(ch *controlState) Widget {
	switch ch.kind {
	case "כפתור":
		cb := ch.onClick
		btn := PushButton{
			AssignTo: &ch.button,
			Text:     ch.text,
			Enabled:  !ch.ctrlDisabled,
			OnClicked: func() {
				invokeYod(cb, nil)
			},
		}
		if ch.ctrlHint != "" {
			btn.ToolTipText = ch.ctrlHint
		}
		if ch.ctrlDark {
			btn.Background = SolidColorBrush{Color: darkBtnBG()}
			btn.Font = Font{Family: "Segoe UI", PointSize: 10}
		}
		return btn
	case "תווית":
		return Label{
			AssignTo:  &ch.label,
			Text:      ch.text,
			TextColor: ch.textColor,
		}
	case "נורית":
		return CustomWidget{
			AssignTo:            &ch.ledWidget,
			MinSize:             Size{Width: 140, Height: 28},
			MaxSize:             Size{Height: 28},
			InvalidatesOnResize: true,
			PaintMode:           PaintBuffered,
			Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
				return paintLED(ch, canvas, bounds)
			},
		}
	case "שדה":
		le := LineEdit{
			AssignTo:           &ch.edit,
			Text:               ch.text,
			Enabled:            !ch.ctrlDisabled,
			PasswordMode:       ch.passwordMode,
			TextAlignment:      AlignFar,
			RightToLeftReading: true,
		}
		if ch.ctrlHint != "" {
			le.ToolTipText = ch.ctrlHint
			le.CueBanner = ch.ctrlHint
		}
		if ch.ctrlDark {
			le.Background = SolidColorBrush{Color: darkFieldBG()}
			le.TextColor = darkCtlText()
			le.Font = Font{Family: "Segoe UI", PointSize: 10}
		}
		return le
	case "רשימה":
		minH := ch.listMinH
		if minH < 40 {
			minH = 200
		}
		items := ch.listItems
		if items == nil {
			items = []string{}
		}
		lb := ListBox{
			AssignTo:      &ch.listBox,
			Model:         items,
			StretchFactor: 1,
			MinSize:       Size{Width: 200, Height: minH},
			Font:          Font{Family: "Consolas", PointSize: 10},
			OnCurrentIndexChanged: func() {
				if ch.onSelect != nil {
					invokeYod(ch.onSelect, nil)
				}
			},
		}
		if ch.ctrlDark {
			lb.Background = SolidColorBrush{Color: darkPanelBG()}
		}
		return lb
	case "טבלה":
		return buildTableWidget(ch)
	case "גרף":
		return buildChartWidget(ch)
	case "דפדפן", "וידאו":
		// רקע כהה תואם לווידג׳ט — בלי לבן מאחורי WebView; בלי Color Key
		return Composite{
			AssignTo:      &ch.host,
			StretchFactor: stretchOr(ch.stretchFactor, 2),
			MinSize:       Size{Width: 200, Height: 120},
			Layout:        VBox{MarginsZero: true, Spacing: 0},
			Background:    SolidColorBrush{Color: walk.RGB(16, 26, 43)},
		}
	case "משטח_GPU":
		return buildGPUSurfaceWidget(ch)
	case "שורה":
		kids := make([]Widget, 0, len(ch.children)+1)
		for _, child := range ch.children {
			child := child
			kids = append(kids, buildControlWidget(child))
		}
		kids = append(kids, HSpacer{})
		return Composite{
			Layout:     HBox{Margins: Margins{Left: 4, Top: 4, Right: 4, Bottom: 4}, Spacing: 4},
			Background: SolidColorBrush{Color: walk.RGB(245, 245, 245)},
			MaxSize:    Size{Height: 48},
			Children:   kids,
		}
	case "דגם":
		cb := ch.onClick
		return CustomWidget{
			AssignTo:            &ch.toolWidget,
			MinSize:             Size{Width: 28, Height: 28},
			MaxSize:             Size{Width: 28, Height: 28},
			InvalidatesOnResize: true,
			PaintMode:           PaintBuffered,
			Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
				return paintSwatch(ch, canvas, bounds)
			},
			OnMouseDown: func(x, y int, button walk.MouseButton) {
				if button == walk.LeftButton {
					invokeYod(cb, nil)
				}
			},
		}
	case "סמל":
		cb := ch.onClick
		return CustomWidget{
			AssignTo:            &ch.toolWidget,
			MinSize:             Size{Width: 40, Height: 36},
			MaxSize:             Size{Width: 48, Height: 40},
			InvalidatesOnResize: true,
			PaintMode:           PaintBuffered,
			ToolTipText:         ch.iconKind,
			Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
				return paintToolIcon(ch, canvas, bounds)
			},
			OnMouseDown: func(x, y int, button walk.MouseButton) {
				if button == walk.LeftButton {
					invokeYod(cb, nil)
				}
			},
		}
	case "עמודה":
		kids := make([]Widget, 0, len(ch.children)+1)
		for _, child := range ch.children {
			child := child
			kids = append(kids, buildControlWidget(child))
		}
		kids = append(kids, VSpacer{})
		return Composite{
			Layout:     VBox{Margins: Margins{Left: 4, Top: 4, Right: 4, Bottom: 4}, Spacing: 3},
			Background: SolidColorBrush{Color: walk.RGB(236, 236, 236)},
			MinSize:    Size{Width: 52},
			MaxSize:    Size{Width: 56},
			Children:   kids,
		}
	case "מסגרת":
		kids := make([]Widget, 0, len(ch.children))
		for _, child := range ch.children {
			child := child
			kids = append(kids, buildControlWidget(child))
		}
		bg := SolidColorBrush{Color: walk.RGB(0, 0, 0)}
		hasBg := ch.hasBg
		if hasBg {
			bg = SolidColorBrush{Color: ch.bgColor}
		}
		sf := stretchOr(ch.stretchFactor, 1)
		if ch.frameDir == "אופקי" {
			margins := Margins{Left: 2, Top: 0, Right: 2, Bottom: 0}
			arranged := kids
			if sf == 0 {
				// סרגל דק: כותרת בצד אחד, פעולות בצד השני
				margins = Margins{Left: 2, Top: 0, Right: 2, Bottom: 0}
				if len(kids) == 0 {
					arranged = kids
				} else if len(kids) == 1 {
					arranged = append([]Widget{HSpacer{}}, kids...)
				} else {
					arranged = make([]Widget, 0, len(kids)+1)
					arranged = append(arranged, kids[0], HSpacer{})
					arranged = append(arranged, kids[1:]...)
				}
			}
			comp := Composite{
				AssignTo:      &ch.panel,
				Layout:        HBox{Margins: margins, Spacing: 4},
				StretchFactor: sf,
				Visible:       !ch.hidden,
				Children:      arranged,
			}
			if sf == 0 {
				comp.MaxSize = Size{Height: 36}
			}
			if hasBg {
				comp.Background = bg
			}
			return comp
		}
		// מסגרת אנכית שממלאת מקום — בלי שולים כפולים סביב דפדפן/תוכן
		vMargins := Margins{Left: 4, Top: 4, Right: 4, Bottom: 4}
		vSpacing := 4
		if sf > 0 {
			vMargins = Margins{}
			vSpacing = 0
		}
		comp := Composite{
			AssignTo:      &ch.panel,
			Layout:        VBox{Margins: vMargins, Spacing: vSpacing},
			StretchFactor: sf,
			Visible:       !ch.hidden,
			Children:      kids,
		}
		if hasBg {
			comp.Background = bg
		}
		return comp
	case "משטח":
		hh := ch.canvasH
		if hh < 40 {
			hh = 40
		}
		ww := ch.canvasW
		if ww < 40 {
			ww = 40
		}
		sf := stretchOr(ch.stretchFactor, 0)
		cw := CustomWidget{
			AssignTo:            &ch.canvas,
			MinSize:             Size{Width: ww, Height: hh},
			StretchFactor:       sf,
			InvalidatesOnResize: true,
			PaintMode:           PaintNormal, // Direct2D Present ל־HWND — בלי באפר GDI שדורס
			Style:               0x00010000, // WS_TABSTOP — מיקוד מקלדת
			Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
				return paintSurface(ch, canvas, bounds)
			},
			OnMouseDown: func(x, y int, button walk.MouseButton) {
				if ch.canvas != nil {
					_ = ch.canvas.SetFocus()
				}
				x, y = mapSurfaceMouse(ch, x, y)
				name := walkButtonName(button)
				ch.dragging = true
				ch.dragButton = name
				invokeYodMouse(ch.onMouseDown, x, y, name)
			},
			OnMouseMove: func(x, y int, button walk.MouseButton) {
				x, y = mapSurfaceMouse(ch, x, y)
				if ch.dragging {
					btn := ch.dragButton
					if btn == "" {
						btn = walkButtonName(button)
					}
					invokeYodMouse(ch.onMouseDrag, x, y, btn)
					return
				}
				invokeYodMouse(ch.onMouseHover, x, y, walkButtonName(button))
			},
			OnMouseUp: func(x, y int, button walk.MouseButton) {
				x, y = mapSurfaceMouse(ch, x, y)
				name := walkButtonName(button)
				ch.dragging = false
				btn := ch.dragButton
				if btn == "" {
					btn = name
				}
				ch.dragButton = ""
				invokeYodMouse(ch.onMouseUp, x, y, btn)
			},
			OnKeyDown: func(key walk.Key) {
				handleSurfaceKeyDown(ch, key)
			},
			OnKeyPress: func(key walk.Key) {
				handleSurfaceKeyPress(ch, key)
			},
		}
		if ch.canvasLockH || ch.canvasLockW {
			max := Size{}
			if ch.canvasLockW {
				max.Width = ww
			}
			if ch.canvasLockH {
				max.Height = hh
			}
			cw.MaxSize = max
		}
		return cw
	default:
		return Label{Text: "?"}
	}
}

func paintSurface(st *controlState, canvas *walk.Canvas, bounds walk.Rectangle) error {
	if st.board == nil || st.canvas == nil {
		return nil
	}
	st.board.syncImgFromBackend()
	// Direct2D — Present ל־HWND (בלי מתיחת GDI)
	if st.board.backend != nil && st.board.backend.Name() == "direct2d" {
		hwnd := uintptr(st.canvas.Handle())
		if hwnd != 0 {
			_ = st.board.backend.BindHWND(hwnd)
			if err := st.board.backend.Present(); err == nil {
				return nil
			}
		}
	}
	// מילוי כל השטח — מונע רקע שחור כשהווידג'ט רחב/גבוה מהביטמאפ
	bgCol := walk.RGB(22, 27, 34)
	if st.board.img != nil && st.board.img.Bounds().Dx() > 0 && st.board.img.Bounds().Dy() > 0 {
		c := st.board.img.RGBAAt(0, 0)
		bgCol = walk.RGB(c.R, c.G, c.B)
	}
	if br, err := walk.NewSolidColorBrush(bgCol); err == nil {
		_ = canvas.FillRectanglePixels(br, bounds)
		br.Dispose()
	}
	src := st.board.img
	iw, ih := src.Bounds().Dx(), src.Bounds().Dy()
	if iw < 1 || ih < 1 {
		return nil
	}
	// ציור 1:1 — בלי מתיחה שמטשטשת ב־HiDPI
	dest := walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: iw, Height: ih}
	if dest.Width > bounds.Width {
		dest.Width = bounds.Width
	}
	if dest.Height > bounds.Height {
		dest.Height = bounds.Height
	}
	// אם עדיין יש פער (למשל לפני resize) — הגדלה איכותית במקום stretch גס
	if (iw != bounds.Width || ih != bounds.Height) && bounds.Width > 0 && bounds.Height > 0 &&
		(iw*ih < bounds.Width*bounds.Height) {
		scaled := image.NewRGBA(image.Rect(0, 0, bounds.Width, bounds.Height))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, src.Bounds(), draw.Src, nil)
		src = scaled
		dest.Width = bounds.Width
		dest.Height = bounds.Height
	}
	dpi := st.canvas.DPI()
	if dpi < 96 {
		dpi = 96
	}
	bmp, err := walk.NewBitmapFromImageForDPI(src, dpi)
	if err != nil {
		return err
	}
	defer bmp.Dispose()
	return canvas.DrawImageStretchedPixels(bmp, dest)
}

func setStretchFactor(st *controlState, name string, a ...object.Object) object.Object {
	if len(a) != 1 {
		return errObj(name + " מצפה למספר >= 0")
	}
	n, ok := a[0].(*object.Number)
	if !ok {
		return errObj(name + " מצפה למספר >= 0")
	}
	v := int(n.Value)
	if v < 0 {
		return errObj(name + " מצפה למספר >= 0")
	}
	st.stretchFactor = v
	return object.Nil
}

func stretchOr(v, def int) int {
	if v < 0 {
		return def
	}
	return v
}

func resizeSurfaceBoard(st *controlState, physW, physH int) {
	if st == nil || st.board == nil || st.sizeBusy {
		return
	}
	dpi := 96
	if st.canvas != nil {
		dpi = st.canvas.DPI()
	}
	if dpi < 96 {
		dpi = 96
	}
	dpr := float64(dpi) / 96.0
	if dpr < 1 {
		dpr = 1
	}

	w, h := physW, physH
	if st.canvasDipW < 40 {
		st.canvasDipW = st.canvasW
	}
	if st.canvasDipH < 40 {
		st.canvasDipH = st.canvasH
	}
	if st.canvasLockW {
		w = dipToPixels(st.canvasDipW, dpi)
	} else {
		st.canvasDipW = pixelsToDIP(physW, dpi)
	}
	if st.canvasLockH {
		h = dipToPixels(st.canvasDipH, dpi)
	} else {
		st.canvasDipH = pixelsToDIP(physH, dpi)
	}
	if w < 40 {
		w = 40
	}
	if h < 40 {
		h = 40
	}
	if w > 8000 {
		w = 8000
	}
	if h > 8000 {
		h = 8000
	}

	ow, oh := st.board.img.Bounds().Dx(), st.board.img.Bounds().Dy()
	st.board.dpr = dpr
	st.board.face = nil // גופן מותאם ל־dpr
	st.canvasW = st.canvasDipW
	st.canvasH = st.canvasDipH
	if ow == w && oh == h {
		return
	}
	st.sizeBusy = true
	defer func() { st.sizeBusy = false }()

	if st.board.backend != nil {
		_ = st.board.backend.Resize(w, h, dpi)
		st.board.syncImgFromBackend()
		st.board.face = nil
		if st.canvas != nil {
			_ = st.board.backend.BindHWND(uintptr(st.canvas.Handle()))
		}
	} else {
		bg := color.RGBA{22, 27, 34, 255}
		if ow > 0 && oh > 0 {
			bg = st.board.img.RGBAAt(0, 0)
		}
		neu := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(neu, neu.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
		if ow > 0 && oh > 0 {
			xdraw.CatmullRom.Scale(neu, neu.Bounds(), st.board.img, st.board.img.Bounds(), draw.Src, nil)
		}
		st.board.img = neu
		st.board.face = nil
	}

	if st.onSizeChange != nil {
		invokeYod(st.onSizeChange, []object.Object{
			&object.Number{Value: float64(st.canvasW)},
			&object.Number{Value: float64(st.canvasH)},
		})
	}
	invalidateCanvas(st)
}

func dipToPixels(dip, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return int(float64(dip)*float64(dpi)/96.0 + 0.5)
}

func wireSurfaceResize(ch *controlState) {
	if ch == nil {
		return
	}
	for _, c := range ch.children {
		wireSurfaceResize(c)
	}
	if ch.kind != "משטח" || ch.canvas == nil {
		return
	}
	enableSurfaceArrowKeys(ch, ch.canvas)
	if !ch.wheelWired && ch.canvas != nil {
		ch.wheelWired = true
		ch.canvas.MouseWheel().Attach(func(x, y int, button walk.MouseButton) {
			_ = x
			_ = y
			delta := walk.MouseWheelEventDelta(button)
			steps := delta / 120
			if steps == 0 {
				if delta > 0 {
					steps = 1
				} else if delta < 0 {
					steps = -1
				}
			}
			if ch.onMouseWheel != nil {
				invokeYod(ch.onMouseWheel, []object.Object{
					&object.Number{Value: float64(steps)},
				})
			}
		})
	}
	if ch.sizeWired {
		return
	}
	ch.sizeWired = true
	ch.canvas.SizeChanged().Attach(func() {
		if ch.canvas == nil {
			return
		}
		b := ch.canvas.ClientBoundsPixels()
		resizeSurfaceBoard(ch, b.Width, b.Height)
	})
}

func syncSurfaceSizesRecursive(ch *controlState) {
	if ch == nil {
		return
	}
	if ch.kind == "משטח" && ch.canvas != nil {
		b := ch.canvas.ClientBoundsPixels()
		resizeSurfaceBoard(ch, b.Width, b.Height)
	}
	for _, c := range ch.children {
		syncSurfaceSizesRecursive(c)
	}
}

func paintSwatch(st *controlState, canvas *walk.Canvas, bounds walk.Rectangle) error {
	c := st.swatchColor
	br, err := walk.NewSolidColorBrush(walk.RGB(c.R, c.G, c.B))
	if err != nil {
		return err
	}
	defer br.Dispose()
	inner := walk.Rectangle{X: bounds.X + 2, Y: bounds.Y + 2, Width: bounds.Width - 4, Height: bounds.Height - 4}
	if err := canvas.FillRectanglePixels(br, inner); err != nil {
		return err
	}
	pen, err := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(60, 60, 60))
	if err != nil {
		return nil
	}
	defer pen.Dispose()
	return canvas.DrawRectanglePixels(pen, inner)
}

func paintToolIcon(st *controlState, canvas *walk.Canvas, bounds walk.Rectangle) error {
	bg, err := walk.NewSolidColorBrush(walk.RGB(250, 250, 250))
	if err != nil {
		return err
	}
	defer bg.Dispose()
	_ = canvas.FillRectanglePixels(bg, bounds)
	border, err := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(180, 180, 180))
	if err != nil {
		return nil
	}
	defer border.Dispose()
	_ = canvas.DrawRectanglePixels(border, bounds)

	ink := walk.RGB(30, 30, 30)
	brush, err := walk.NewSolidColorBrush(ink)
	if err != nil {
		return nil
	}
	defer brush.Dispose()
	pen, err := walk.NewGeometricPen(walk.PenSolid, 2, brush)
	if err != nil {
		return nil
	}
	defer pen.Dispose()
	cx := bounds.X + bounds.Width/2
	cy := bounds.Y + bounds.Height/2
	pad := 8

	switch st.iconKind {
	case "עיפרון":
		_ = canvas.DrawLinePixels(pen, walk.Point{X: bounds.X + pad, Y: bounds.Y + bounds.Height - pad},
			walk.Point{X: bounds.X + bounds.Width - pad, Y: bounds.Y + pad})
		tip, _ := walk.NewSolidColorBrush(walk.RGB(40, 40, 40))
		if tip != nil {
			defer tip.Dispose()
			_ = canvas.FillEllipsePixels(tip, walk.Rectangle{X: bounds.X + bounds.Width - pad - 3, Y: bounds.Y + pad - 2, Width: 6, Height: 6})
		}
	case "מחק":
		er, _ := walk.NewSolidColorBrush(walk.RGB(255, 180, 200))
		if er != nil {
			defer er.Dispose()
			_ = canvas.FillRectanglePixels(er, walk.Rectangle{X: bounds.X + pad, Y: cy - 6, Width: bounds.Width - 2*pad, Height: 12})
		}
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: bounds.X + pad, Y: cy - 6, Width: bounds.Width - 2*pad, Height: 12})
	case "קו":
		_ = canvas.DrawLinePixels(pen, walk.Point{X: bounds.X + pad, Y: bounds.Y + bounds.Height - pad},
			walk.Point{X: bounds.X + bounds.Width - pad, Y: bounds.Y + pad})
	case "מלבן":
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: bounds.X + pad, Y: bounds.Y + pad, Width: bounds.Width - 2*pad, Height: bounds.Height - 2*pad})
	case "עיגול":
		_ = canvas.DrawEllipsePixels(pen, walk.Rectangle{X: bounds.X + pad, Y: bounds.Y + pad, Width: bounds.Width - 2*pad, Height: bounds.Height - 2*pad})
	case "אליפסה":
		_ = canvas.DrawEllipsePixels(pen, walk.Rectangle{X: bounds.X + pad - 2, Y: bounds.Y + pad + 4, Width: bounds.Width - 2*pad + 4, Height: bounds.Height - 2*pad - 8})
	case "מילוי":
		fill, _ := walk.NewSolidColorBrush(walk.RGB(100, 149, 237))
		if fill != nil {
			defer fill.Dispose()
			_ = canvas.FillEllipsePixels(fill, walk.Rectangle{X: cx - 8, Y: cy - 4, Width: 16, Height: 14})
		}
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx, Y: bounds.Y + pad}, walk.Point{X: cx, Y: cy})
	case "טפטפת":
		_ = canvas.DrawLinePixels(pen, walk.Point{X: bounds.X + pad + 2, Y: bounds.Y + bounds.Height - pad},
			walk.Point{X: cx + 4, Y: bounds.Y + pad + 4})
		drop, _ := walk.NewSolidColorBrush(walk.RGB(220, 50, 50))
		if drop != nil {
			defer drop.Dispose()
			_ = canvas.FillEllipsePixels(drop, walk.Rectangle{X: cx + 2, Y: bounds.Y + pad, Width: 8, Height: 10})
		}
	case "בטל":
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx + 8, Y: cy - 8}, walk.Point{X: cx - 8, Y: cy - 8})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx - 8, Y: cy - 8}, walk.Point{X: cx - 2, Y: cy - 14})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: cx - 8, Y: cy - 8}, walk.Point{X: cx - 2, Y: cy - 2})
		_ = canvas.DrawEllipsePixels(pen, walk.Rectangle{X: cx - 10, Y: cy - 10, Width: 20, Height: 18})
	case "נקה":
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: bounds.X + pad, Y: bounds.Y + pad + 2, Width: bounds.Width - 2*pad, Height: bounds.Height - 2*pad - 2})
		_ = canvas.DrawLinePixels(pen, walk.Point{X: bounds.X + pad + 2, Y: bounds.Y + pad + 4},
			walk.Point{X: bounds.X + bounds.Width - pad - 2, Y: bounds.Y + bounds.Height - pad - 2})
	case "פתח":
		_ = canvas.DrawRectanglePixels(pen, walk.Rectangle{X: bounds.X + pad, Y: cy - 2, Width: bounds.Width - 2*pad, Height: bounds.Height/2 - 2})
		tab, _ := walk.NewSolidColorBrush(walk.RGB(255, 220, 100))
		if tab != nil {
			defer tab.Dispose()
			_ = canvas.FillRectanglePixels(tab, walk.Rectangle{X: bounds.X + pad, Y: bounds.Y + pad, Width: 12, Height: 8})
		}
	case "שמור":
		disk, _ := walk.NewSolidColorBrush(walk.RGB(70, 70, 160))
		if disk != nil {
			defer disk.Dispose()
			_ = canvas.FillRectanglePixels(disk, walk.Rectangle{X: bounds.X + pad, Y: bounds.Y + pad, Width: bounds.Width - 2*pad, Height: bounds.Height - 2*pad})
		}
		slot, _ := walk.NewSolidColorBrush(walk.RGB(230, 230, 230))
		if slot != nil {
			defer slot.Dispose()
			_ = canvas.FillRectanglePixels(slot, walk.Rectangle{X: cx - 6, Y: bounds.Y + pad + 2, Width: 12, Height: 8})
		}
	default:
		_ = canvas.DrawLinePixels(pen, walk.Point{X: bounds.X + pad, Y: cy}, walk.Point{X: bounds.X + bounds.Width - pad, Y: cy})
	}
	return nil
}
