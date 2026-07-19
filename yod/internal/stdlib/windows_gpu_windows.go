//go:build windows

package stdlib

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/gpu"
	"yod/internal/object"
)

// חלונות.משטח_GPU(רוחב, גובה) — משטח OpenGL מקומי (לא WebView).
func winCreateGPUSurface(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלונות.משטח_GPU מצפה לרוחב וגובה")
	}
	wn, ok1 := args[0].(*object.Number)
	hn, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("חלונות.משטח_GPU מצפה למספרים")
	}
	ww, hh := int(wn.Value), int(hn.Value)
	if ww < 40 || hh < 40 || ww > 4000 || hh > 4000 {
		return errObj("גודל משטח_GPU לא תקין (40–4000)")
	}
	surf := gpu.NewSurface(ww, hh)
	st := &controlState{
		kind:          "משטח_GPU",
		canvasW:       ww,
		canvasH:       hh,
		gpuSurface:    surf,
		stretchFactor: 2,
	}
	w := &object.GuiWidget{Kind: "משטח_GPU", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["רענן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("משטח_GPU.רענן מצפה ל־0 ארגומנטים")
		}
		if st.gpuSurface == nil {
			return errObj("משטח GPU לא מוכן")
		}
		if err := st.gpuSurface.Present(); err != nil {
			return errObj(err.Error())
		}
		return object.Nil
	}}
	w.Attrs["נקה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 3 {
			return errObj("משטח_GPU.נקה מצפה ל־R,G,B (0–1)")
		}
		r, ok1 := a[0].(*object.Number)
		g, ok2 := a[1].(*object.Number)
		b, ok3 := a[2].(*object.Number)
		if !ok1 || !ok2 || !ok3 {
			return errObj("משטח_GPU.נקה מצפה למספרים")
		}
		st.gpuSurface.SetClear(float32(r.Value), float32(g.Value), float32(b.Value))
		return object.Nil
	}}
	w.Attrs["הצג_משולש"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("משטח_GPU.הצג_משולש מצפה ל־0 ארגומנטים")
		}
		st.gpuSurface.SetMode("משולש")
		return object.Nil
	}}
	w.Attrs["קבע_סצנה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("משטח_GPU.קבע_סצנה מצפה לסצנת GPU")
		}
		sc, err := sceneFromObject(a[0])
		if err != nil {
			return errObj(err.Error())
		}
		st.gpuSurface.SetScene(sc)
		return object.Nil
	}}
	w.Attrs["קרא_FPS"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if st.gpuSurface == nil {
			return &object.Number{Value: 0}
		}
		return &object.Number{Value: st.gpuSurface.FPS()}
	}}
	w.Attrs["קרא_backend"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		name := ""
		if st.gpuSurface != nil {
			name = st.gpuSurface.BackendName()
		}
		return &object.String{Value: name}
	}}
	w.Attrs["קבע_מתיחה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setStretchFactor(st, "משטח_GPU.קבע_מתיחה", a...)
	}}
	w.Attrs["בעכבר_למטה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseDown, "בעכבר_למטה", a)
	}}
	w.Attrs["בעכבר_גרירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseDrag, "בעכבר_גרירה", a)
	}}
	w.Attrs["בעכבר_למעלה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseUp, "בעכבר_למעלה", a)
	}}
	w.Attrs["בעכבר_גלגל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setMouseHandler(&st.onMouseWheel, "בעכבר_גלגל", a)
	}}
	return w
}

func surfaceFromWidget(obj object.Object) (*gpu.Surface, error) {
	gw, ok := obj.(*object.GuiWidget)
	if !ok {
		return nil, fmt.Errorf("מצפה למשטח_GPU")
	}
	st, ok := gw.Data.(*controlState)
	if !ok || st.kind != "משטח_GPU" || st.gpuSurface == nil {
		return nil, fmt.Errorf("מצפה למשטח_GPU")
	}
	return st.gpuSurface, nil
}

func sceneFromObject(obj object.Object) (*gpu.Scene, error) {
	gw, ok := obj.(*object.GuiWidget)
	if !ok {
		return nil, fmt.Errorf("מצפה לסצנת GPU")
	}
	if gw.Kind != "סצנת_GPU" {
		return nil, fmt.Errorf("מצפה לסצנת GPU")
	}
	sc, ok := gw.Data.(*gpu.Scene)
	if !ok || sc == nil {
		return nil, fmt.Errorf("סצנת GPU לא תקינה")
	}
	return sc, nil
}

func buildGPUSurfaceWidget(ch *controlState) Widget {
	ww, hh := ch.canvasW, ch.canvasH
	if ww < 40 {
		ww = 40
	}
	if hh < 40 {
		hh = 40
	}
	sf := stretchOr(ch.stretchFactor, 2)
	return Composite{
		AssignTo:      &ch.host,
		StretchFactor: sf,
		MinSize:       Size{Width: ww, Height: hh},
		Layout:        VBox{MarginsZero: true},
	}
}

func wireGPUSurfacesRecursive(ch *controlState, mw *walk.MainWindow) {
	if ch.kind == "משטח_GPU" && ch.host != nil && ch.gpuSurface != nil {
		hwnd := uintptr(ch.host.Handle())
		if err := ch.gpuSurface.Attach(hwnd); err != nil {
			walk.MsgBox(mw, "שגיאת GPU", err.Error(), walk.MsgBoxIconError)
			return
		}
		bounds := ch.host.ClientBounds()
		if bounds.Width > 0 && bounds.Height > 0 {
			ch.gpuSurface.Resize(bounds.Width, bounds.Height)
		}
		_ = ch.gpuSurface.Present()
		ch.host.SizeChanged().Attach(func() {
			b := ch.host.ClientBounds()
			if b.Width > 0 && b.Height > 0 {
				ch.gpuSurface.Resize(b.Width, b.Height)
			}
		})
	}
	for _, c := range ch.children {
		wireGPUSurfacesRecursive(c, mw)
	}
}

func resizeGPUSurfacesRecursive(children []*controlState) {
	for _, ch := range children {
		if ch.kind == "משטח_GPU" && ch.host != nil && ch.gpuSurface != nil {
			b := ch.host.ClientBounds()
			if b.Width > 0 && b.Height > 0 {
				ch.gpuSurface.Resize(b.Width, b.Height)
			}
		}
		resizeGPUSurfacesRecursive(ch.children)
	}
}
