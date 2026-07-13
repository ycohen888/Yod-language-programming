//go:build windows

package stdlib

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/object"
)

type paintTool int

const (
	toolPencil paintTool = iota
	toolEraser
	toolLine
	toolRect
	toolEllipse
	toolFill
)

type paintEditor struct {
	img       *image.RGBA
	tool      paintTool
	fg        color.RGBA
	thickness int
	drawing   bool
	startX    int
	startY    int
	lastX     int
	lastY     int
	preview   bool
	prevX2    int
	prevY2    int
	undo      []*image.RGBA
	status    *walk.Label
	canvas    *walk.CustomWidget
	fgSwatch  *walk.CustomWidget
	mw        *walk.MainWindow
}

var paintPalette = []color.RGBA{
	{0, 0, 0, 255}, {128, 128, 128, 255}, {128, 0, 0, 255}, {128, 128, 0, 255},
	{0, 128, 0, 255}, {0, 128, 128, 255}, {0, 0, 128, 255}, {128, 0, 128, 255},
	{128, 128, 64, 255}, {0, 64, 64, 255}, {0, 128, 255, 255}, {0, 64, 128, 255},
	{64, 0, 255, 255}, {128, 64, 0, 255},
	{255, 255, 255, 255}, {192, 192, 192, 255}, {255, 0, 0, 255}, {255, 255, 0, 255},
	{0, 255, 0, 255}, {0, 255, 255, 255}, {0, 0, 255, 255}, {255, 0, 255, 255},
	{255, 255, 128, 255}, {64, 255, 192, 255}, {128, 255, 255, 255}, {128, 128, 255, 255},
	{255, 0, 128, 255}, {255, 128, 64, 255},
}

func drawOpenEditor(args ...object.Object) object.Object {
	w, h := 800, 560
	title := "צייר — יוד"
	if len(args) >= 1 {
		if n, ok := args[0].(*object.Number); ok {
			w = int(n.Value)
		} else if s, ok := asString(args[0]); ok {
			title = s
		} else {
			return errObj("ציור.עורך: ארגומנטים — [רוחב], [גובה], [כותרת]")
		}
	}
	if len(args) >= 2 {
		if n, ok := args[1].(*object.Number); ok {
			h = int(n.Value)
		} else if s, ok := asString(args[1]); ok {
			title = s
		}
	}
	if len(args) >= 3 {
		if s, ok := asString(args[2]); ok {
			title = s
		}
	}
	if w < 80 || h < 80 || w > 4000 || h > 4000 {
		return errObj("ציור.עורך: גודל לוח לא תקין (80–4000)")
	}
	return runPaintEditor(w, h, title)
}

func runPaintEditor(w, h int, title string) object.Object {
	ed := &paintEditor{
		img:       image.NewRGBA(image.Rect(0, 0, w, h)),
		tool:      toolPencil,
		fg:        color.RGBA{0, 0, 0, 255},
		thickness: 2,
	}
	draw.Draw(ed.img, ed.img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	ed.pushUndo()

	setTool := func(t paintTool) {
		ed.tool = t
		ed.refreshStatus()
	}
	setSize := func(n int) {
		ed.thickness = n
		ed.refreshStatus()
	}

	colorRow1 := make([]Widget, 0, 14)
	colorRow2 := make([]Widget, 0, 14)
	for i, c := range paintPalette {
		sw := paintColorSwatch(ed, c)
		if i < 14 {
			colorRow1 = append(colorRow1, sw)
		} else {
			colorRow2 = append(colorRow2, sw)
		}
	}

	winW := w + 40
	winH := h + 180
	if winW < 720 {
		winW = 720
	}
	if winH < 520 {
		winH = 520
	}

	if err := (MainWindow{
		AssignTo: &ed.mw,
		Title:    title,
		MinSize:  Size{Width: 640, Height: 480},
		Size:     Size{Width: winW, Height: winH},
		Layout:   VBox{Margins: Margins{Left: 6, Top: 6, Right: 6, Bottom: 4}, Spacing: 4},
		Children: []Widget{
			Composite{
				Layout:     HBox{Margins: Margins{Left: 4, Top: 2, Right: 4, Bottom: 2}, Spacing: 3},
				Background: SolidColorBrush{Color: walk.RGB(245, 245, 245)},
				MaxSize:    Size{Height: 34},
				Children: []Widget{
					Label{Text: "כלים"},
					PushButton{Text: "עיפרון", MaxSize: Size{Width: 64}, OnClicked: func() { setTool(toolPencil) }},
					PushButton{Text: "מחק", MaxSize: Size{Width: 52}, OnClicked: func() { setTool(toolEraser) }},
					PushButton{Text: "קו", MaxSize: Size{Width: 44}, OnClicked: func() { setTool(toolLine) }},
					PushButton{Text: "מלבן", MaxSize: Size{Width: 56}, OnClicked: func() { setTool(toolRect) }},
					PushButton{Text: "אליפסה", MaxSize: Size{Width: 64}, OnClicked: func() { setTool(toolEllipse) }},
					PushButton{Text: "מילוי", MaxSize: Size{Width: 52}, OnClicked: func() { setTool(toolFill) }},
					VSeparator{},
					Label{Text: "עובי"},
					PushButton{Text: "1", MaxSize: Size{Width: 28}, OnClicked: func() { setSize(1) }},
					PushButton{Text: "2", MaxSize: Size{Width: 28}, OnClicked: func() { setSize(2) }},
					PushButton{Text: "4", MaxSize: Size{Width: 28}, OnClicked: func() { setSize(4) }},
					PushButton{Text: "8", MaxSize: Size{Width: 28}, OnClicked: func() { setSize(8) }},
					VSeparator{},
					PushButton{Text: "בטל", MaxSize: Size{Width: 44}, OnClicked: func() { ed.undoLast() }},
					PushButton{Text: "נקה", MaxSize: Size{Width: 44}, OnClicked: func() { ed.clearCanvas() }},
					PushButton{Text: "שמור…", MaxSize: Size{Width: 64}, OnClicked: func() { ed.saveDialog() }},
					HSpacer{},
				},
			},
			Composite{
				Layout:        HBox{MarginsZero: true, Spacing: 0},
				Background:    SolidColorBrush{Color: walk.RGB(160, 160, 160)},
				StretchFactor: 1,
				Children: []Widget{
					HSpacer{},
					CustomWidget{
						AssignTo:            &ed.canvas,
						MinSize:             Size{Width: w, Height: h},
						MaxSize:             Size{Width: w, Height: h},
						InvalidatesOnResize: true,
						PaintMode:           PaintBuffered,
						Paint:               ed.paintCanvas,
					},
					HSpacer{},
				},
			},
			Composite{
				Layout:     HBox{Margins: Margins{Left: 4, Top: 2, Right: 4, Bottom: 2}, Spacing: 6},
				Background: SolidColorBrush{Color: walk.RGB(245, 245, 245)},
				MaxSize:    Size{Height: 56},
				Children: []Widget{
					Label{Text: "צבע"},
					CustomWidget{
						AssignTo:            &ed.fgSwatch,
						MinSize:             Size{Width: 36, Height: 36},
						MaxSize:             Size{Width: 36, Height: 36},
						InvalidatesOnResize: true,
						PaintMode:           PaintBuffered,
						Paint:               ed.paintFgSwatch,
					},
					VSeparator{},
					Composite{
						Layout: VBox{MarginsZero: true, Spacing: 2},
						Children: []Widget{
							Composite{Layout: HBox{MarginsZero: true, Spacing: 2}, Children: colorRow1},
							Composite{Layout: HBox{MarginsZero: true, Spacing: 2}, Children: colorRow2},
						},
					},
					HSpacer{},
				},
			},
			Label{
				AssignTo:  &ed.status,
				Text:      "עיפרון · שחור · עובי 2 — גררו עם העכבר לציור",
				TextColor: walk.RGB(50, 50, 50),
			},
		},
	}).Create(); err != nil {
		return errObj("פתיחת עורך ציור נכשלה: " + err.Error())
	}

	ed.wireMouse()
	ed.refreshStatus()
	ed.mw.Run()
	return object.Nil
}

func paintColorSwatch(ed *paintEditor, c color.RGBA) Widget {
	var w *walk.CustomWidget
	return CustomWidget{
		AssignTo:            &w,
		MinSize:             Size{Width: 18, Height: 18},
		MaxSize:             Size{Width: 18, Height: 18},
		InvalidatesOnResize: true,
		PaintMode:           PaintBuffered,
		Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
			br, err := walk.NewSolidColorBrush(walk.RGB(c.R, c.G, c.B))
			if err != nil {
				return err
			}
			defer br.Dispose()
			if err := canvas.FillRectanglePixels(br, bounds); err != nil {
				return err
			}
			pen, err := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(80, 80, 80))
			if err != nil {
				return nil
			}
			defer pen.Dispose()
			return canvas.DrawRectanglePixels(pen, bounds)
		},
		OnMouseDown: func(x, y int, button walk.MouseButton) {
			if button == walk.LeftButton {
				ed.fg = c
				if ed.fgSwatch != nil {
					ed.fgSwatch.Invalidate()
				}
				ed.refreshStatus()
			}
		},
	}
}

func (ed *paintEditor) paintFgSwatch(canvas *walk.Canvas, bounds walk.Rectangle) error {
	br, err := walk.NewSolidColorBrush(walk.RGB(ed.fg.R, ed.fg.G, ed.fg.B))
	if err != nil {
		return err
	}
	defer br.Dispose()
	if err := canvas.FillRectanglePixels(br, bounds); err != nil {
		return err
	}
	pen, err := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	if err != nil {
		return nil
	}
	defer pen.Dispose()
	return canvas.DrawRectanglePixels(pen, bounds)
}

func (ed *paintEditor) wireMouse() {
	if ed.canvas == nil {
		return
	}
	ed.canvas.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		ed.onDown(x, y)
	})
	ed.canvas.MouseMove().Attach(func(x, y int, button walk.MouseButton) {
		ed.onMove(x, y, button == walk.LeftButton)
	})
	ed.canvas.MouseUp().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		ed.onUp(x, y)
	})
}

func (ed *paintEditor) paintCanvas(canvas *walk.Canvas, bounds walk.Rectangle) error {
	view := ed.img
	if ed.preview && ed.drawing {
		view = cloneRGBA(ed.img)
		ed.drawShape(view, ed.startX, ed.startY, ed.prevX2, ed.prevY2, ed.fg, ed.thickness)
	}
	bmp, err := walk.NewBitmapFromImageForDPI(view, ed.canvas.DPI())
	if err != nil {
		return err
	}
	defer bmp.Dispose()
	dest := walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: ed.img.Bounds().Dx(), Height: ed.img.Bounds().Dy()}
	return canvas.DrawImageStretchedPixels(bmp, dest)
}

func (ed *paintEditor) onDown(x, y int) {
	ed.pushUndo()
	ed.drawing = true
	ed.startX, ed.startY = x, y
	ed.lastX, ed.lastY = x, y
	ed.prevX2, ed.prevY2 = x, y

	switch ed.tool {
	case toolPencil:
		r := ed.thickness / 2
		drawDisk(ed.img, x, y, r, ed.fg)
		if ed.thickness < 2 {
			ed.img.Set(x, y, ed.fg)
		}
		ed.invalidate()
	case toolEraser:
		drawDisk(ed.img, x, y, maxInt(ed.thickness, 6)/2, color.RGBA{255, 255, 255, 255})
		ed.invalidate()
	case toolFill:
		floodFill(ed.img, x, y, ed.fg)
		ed.drawing = false
		ed.invalidate()
	case toolLine, toolRect, toolEllipse:
		ed.preview = true
	}
}

func (ed *paintEditor) onMove(x, y int, leftDown bool) {
	if !ed.drawing || !leftDown {
		return
	}
	switch ed.tool {
	case toolPencil:
		drawThickLine(ed.img, ed.lastX, ed.lastY, x, y, ed.thickness, ed.fg)
		ed.lastX, ed.lastY = x, y
		ed.invalidate()
	case toolEraser:
		th := maxInt(ed.thickness, 6)
		drawThickLine(ed.img, ed.lastX, ed.lastY, x, y, th, color.RGBA{255, 255, 255, 255})
		ed.lastX, ed.lastY = x, y
		ed.invalidate()
	case toolLine, toolRect, toolEllipse:
		ed.prevX2, ed.prevY2 = x, y
		ed.invalidate()
	}
}

func (ed *paintEditor) onUp(x, y int) {
	if !ed.drawing {
		return
	}
	ed.drawing = false
	switch ed.tool {
	case toolLine, toolRect, toolEllipse:
		ed.preview = false
		ed.drawShape(ed.img, ed.startX, ed.startY, x, y, ed.fg, ed.thickness)
		ed.invalidate()
	default:
		ed.preview = false
	}
}

func (ed *paintEditor) drawShape(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, th int) {
	switch ed.tool {
	case toolLine:
		drawThickLine(img, x0, y0, x1, y1, th, c)
	case toolRect:
		x, y := minInt(x0, x1), minInt(y0, y1)
		w, h := absInt(x1-x0), absInt(y1-y0)
		drawRectOutline(img, x, y, w, h, th, c)
	case toolEllipse:
		cx := (x0 + x1) / 2
		cy := (y0 + y1) / 2
		rx := absInt(x1-x0) / 2
		ry := absInt(y1-y0) / 2
		drawEllipseOutline(img, cx, cy, rx, ry, th, c)
	}
}

func (ed *paintEditor) invalidate() {
	if ed.canvas != nil {
		ed.canvas.Invalidate()
	}
}

func (ed *paintEditor) refreshStatus() {
	if ed.status == nil {
		return
	}
	tool := "עיפרון"
	switch ed.tool {
	case toolEraser:
		tool = "מחק"
	case toolLine:
		tool = "קו"
	case toolRect:
		tool = "מלבן"
	case toolEllipse:
		tool = "אליפסה"
	case toolFill:
		tool = "מילוי"
	}
	ed.status.SetText(fmt.Sprintf("%s · RGB(%d,%d,%d) · עובי %d — גררו עם העכבר לציור",
		tool, ed.fg.R, ed.fg.G, ed.fg.B, ed.thickness))
}

func (ed *paintEditor) clearCanvas() {
	ed.pushUndo()
	draw.Draw(ed.img, ed.img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	ed.invalidate()
	if ed.status != nil {
		ed.status.SetText("הלוח נוקה")
	}
}

func (ed *paintEditor) pushUndo() {
	ed.undo = append(ed.undo, cloneRGBA(ed.img))
	if len(ed.undo) > 25 {
		ed.undo = ed.undo[len(ed.undo)-25:]
	}
}

func (ed *paintEditor) undoLast() {
	if len(ed.undo) == 0 {
		return
	}
	last := ed.undo[len(ed.undo)-1]
	ed.undo = ed.undo[:len(ed.undo)-1]
	draw.Draw(ed.img, ed.img.Bounds(), last, image.Point{}, draw.Src)
	ed.invalidate()
	if ed.status != nil {
		ed.status.SetText("בוטל הצעד האחרון")
	}
}

func (ed *paintEditor) saveDialog() {
	dlg := new(walk.FileDialog)
	dlg.Title = "שמירת ציור"
	dlg.Filter = "תמונת PNG (*.png)|*.png|JPEG (*.jpg)|*.jpg"
	dlg.FilePath = "ציור_שלי.png"
	ok, err := dlg.ShowSave(ed.mw)
	if err != nil || !ok {
		return
	}
	path := dlg.FilePath
	if filepath.Ext(path) == "" {
		path += ".png"
	}
	if err := saveImageFile(ed.img, path); err != nil {
		walk.MsgBox(ed.mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
		return
	}
	if ed.status != nil {
		ed.status.SetText("נשמר: " + filepath.Base(path))
	}
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

func floodFill(img *image.RGBA, x, y int, repl color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
		return
	}
	target := img.RGBAAt(x, y)
	if rgbaEq(target, repl) {
		return
	}
	type pt struct{ x, y int }
	stack := []pt{{x, y}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if p.x < b.Min.X || p.y < b.Min.Y || p.x >= b.Max.X || p.y >= b.Max.Y {
			continue
		}
		if !rgbaEq(img.RGBAAt(p.x, p.y), target) {
			continue
		}
		img.SetRGBA(p.x, p.y, repl)
		stack = append(stack, pt{p.x + 1, p.y}, pt{p.x - 1, p.y}, pt{p.x, p.y + 1}, pt{p.x, p.y - 1})
	}
}

func rgbaEq(a, b color.RGBA) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}

func drawEllipseOutline(img *image.RGBA, cx, cy, rx, ry, thickness int, c color.RGBA) {
	if rx < 1 {
		rx = 1
	}
	if ry < 1 {
		ry = 1
	}
	steps := (rx + ry) * 4
	if steps < 48 {
		steps = 48
	}
	for i := 0; i <= steps; i++ {
		ang := 2 * math.Pi * float64(i) / float64(steps)
		x := cx + int(float64(rx)*math.Cos(ang))
		y := cy + int(float64(ry)*math.Sin(ang))
		drawDisk(img, x, y, thickness/2, c)
		if thickness < 2 {
			img.Set(x, y, c)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
