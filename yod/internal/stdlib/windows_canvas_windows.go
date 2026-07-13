//go:build windows

package stdlib

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/object"
)

// חלונות.שורה() — מקבץ רכיבים אופקית (סרגל כלים / צבעים)
func winCreateRow(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלונות.שורה מצפה ל־0 ארגומנטים")
	}
	st := &controlState{kind: "שורה", children: nil}
	w := &object.GuiWidget{Kind: "שורה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["הוסף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("שורה.הוסף מצפה לרכיב אחד")
		}
		gw, ok := a[0].(*object.GuiWidget)
		if !ok {
			return errObj("שורה.הוסף מצפה לרכיב ממשק")
		}
		cs, ok := gw.Data.(*controlState)
		if !ok {
			return errObj("שורה.הוסף: רכיב לא תקין")
		}
		st.children = append(st.children, cs)
		return object.Nil
	}}
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
	img := image.NewRGBA(image.Rect(0, 0, ww, hh))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	board := &drawBoard{
		img:    img,
		stroke: color.RGBA{0, 0, 0, 255},
		fill:   color.RGBA{200, 200, 200, 255},
		width:  2,
		fontSz: 16,
	}
	st := &controlState{
		kind:      "משטח",
		board:     board,
		canvasW:   ww,
		canvasH:   hh,
		dragging:  false,
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
	w.Attrs["נקה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c := color.RGBA{255, 255, 255, 255}
		if len(a) > 0 {
			parsed, err := parseDrawColor(a...)
			if err != nil {
				return errObj(err.Error())
			}
			c = parsed
		}
		draw.Draw(board.img, board.img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["נקודה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, err := twoInts("נקודה", a)
		if err != nil {
			return err
		}
		drawDisk(board.img, x, y, board.width/2, board.stroke)
		if board.width < 2 {
			board.img.Set(x, y, board.stroke)
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
		drawThickLine(board.img, vals[0], vals[1], vals[2], vals[3], board.width, board.stroke)
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלבן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן", a, 4)
		if err != nil {
			return err
		}
		drawRectOutline(board.img, vals[0], vals[1], vals[2], vals[3], board.width, board.stroke)
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלבן_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן_מלא", a, 4)
		if err != nil {
			return err
		}
		drawRectFill(board.img, vals[0], vals[1], vals[2], vals[3], board.fill)
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["עיגול"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("עיגול", a, 3)
		if err != nil {
			return err
		}
		drawCircleOutline(board.img, vals[0], vals[1], vals[2], board.width, board.stroke)
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["עיגול_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("עיגול_מלא", a, 3)
		if err != nil {
			return err
		}
		drawDisk(board.img, vals[0], vals[1], vals[2], board.fill)
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["מלא_באזור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, err := twoInts("מלא_באזור", a)
		if err != nil {
			return err
		}
		floodFill(board.img, x, y, board.stroke)
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
		if err := saveImageFile(board.img, path); err != nil {
			return errObj(err.Error())
		}
		return object.Nil
	}}
	w.Attrs["רענן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		invalidateCanvas(st)
		return object.Nil
	}}
	w.Attrs["רוחב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(board.img.Bounds().Dx())}
	}}
	w.Attrs["גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(board.img.Bounds().Dy())}
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
	if st != nil && st.canvas != nil {
		st.canvas.Invalidate()
	}
}

func invokeYod(fn object.Object, args []object.Object) {
	if fn == nil {
		return
	}
	var res object.Object
	if object.InvokeCallable != nil {
		res = object.InvokeCallable(fn, args)
	} else if f, ok := fn.(*object.Function); ok && object.InvokeFunction != nil {
		res = object.InvokeFunction(f, args)
	}
	if err, ok := res.(*object.Error); ok {
		walk.MsgBox(nil, "שגיאה", err.Message, walk.MsgBoxIconError)
	}
}

func numObj(v int) object.Object {
	return &object.Number{Value: float64(v)}
}

func buildControlWidget(ch *controlState) Widget {
	switch ch.kind {
	case "כפתור":
		cb := ch.onClick
		return PushButton{
			Text: ch.text,
			OnClicked: func() {
				invokeYod(cb, nil)
			},
		}
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
		return LineEdit{
			AssignTo: &ch.edit,
			Text:     ch.text,
		}
	case "דפדפן":
		return Composite{
			AssignTo:      &ch.host,
			StretchFactor: 1,
			MinSize:       Size{Width: 120, Height: 180},
			Layout:        VBox{MarginsZero: true},
		}
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
	case "משטח":
		ww, hh := ch.canvasW, ch.canvasH
		return CustomWidget{
			AssignTo:            &ch.canvas,
			MinSize:             Size{Width: ww, Height: hh},
			StretchFactor:       1,
			InvalidatesOnResize: true,
			PaintMode:           PaintBuffered,
			Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
				return paintSurface(ch, canvas, bounds)
			},
			OnMouseDown: func(x, y int, button walk.MouseButton) {
				if button != walk.LeftButton {
					return
				}
				ch.dragging = true
				invokeYod(ch.onMouseDown, []object.Object{numObj(x), numObj(y)})
			},
			OnMouseMove: func(x, y int, button walk.MouseButton) {
				if !ch.dragging {
					return
				}
				invokeYod(ch.onMouseDrag, []object.Object{numObj(x), numObj(y)})
			},
			OnMouseUp: func(x, y int, button walk.MouseButton) {
				if button != walk.LeftButton {
					return
				}
				ch.dragging = false
				invokeYod(ch.onMouseUp, []object.Object{numObj(x), numObj(y)})
			},
		}
	default:
		return Label{Text: "?"}
	}
}

func paintSurface(st *controlState, canvas *walk.Canvas, bounds walk.Rectangle) error {
	if st.board == nil || st.canvas == nil {
		return nil
	}
	bmp, err := walk.NewBitmapFromImageForDPI(st.board.img, st.canvas.DPI())
	if err != nil {
		return err
	}
	defer bmp.Dispose()
	dest := walk.Rectangle{
		X: bounds.X, Y: bounds.Y,
		Width:  st.board.img.Bounds().Dx(),
		Height: st.board.img.Bounds().Dy(),
	}
	return canvas.DrawImageStretchedPixels(bmp, dest)
}

