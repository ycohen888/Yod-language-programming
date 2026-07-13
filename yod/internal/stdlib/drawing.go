package stdlib

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/unicode/bidi"

	"yod/internal/object"
)

func NewDrawingModule() *object.Module {
	m := &object.Module{Name: "ציור", Attrs: map[string]object.Object{}}
	m.Attrs["לוח"] = &object.Builtin{Fn: drawCreateBoard}
	m.Attrs["טען_תמונה"] = &object.Builtin{Fn: drawLoadImage}
	m.Attrs["צבע"] = &object.Builtin{Fn: drawMakeColor}
	return m
}

type drawBoard struct {
	img    *image.RGBA
	stroke color.RGBA
	fill   color.RGBA
	width  int // עובי קו
	fontSz float64
	face   font.Face
}

type drawImage struct {
	img image.Image
}

func drawCreateBoard(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("ציור.לוח מצפה לרוחב וגובה")
	}
	w, ok1 := args[0].(*object.Number)
	h, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("ציור.לוח מצפה למספרים")
	}
	ww, hh := int(w.Value), int(h.Value)
	if ww < 1 || hh < 1 || ww > 8000 || hh > 8000 {
		return errObj("גודל לוח לא תקין (1–8000)")
	}
	img := image.NewRGBA(image.Rect(0, 0, ww, hh))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	st := &drawBoard{
		img:    img,
		stroke: color.RGBA{0, 0, 0, 255},
		fill:   color.RGBA{200, 200, 200, 255},
		width:  2,
		fontSz: 16,
	}
	return wrapBoard(st)
}

func wrapBoard(st *drawBoard) *object.GuiWidget {
	w := &object.GuiWidget{Kind: "לוח", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_צבע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c, err := parseDrawColor(a...)
		if err != nil {
			return errObj(err.Error())
		}
		st.stroke = c
		return object.Nil
	}}
	w.Attrs["קבע_מילוי"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		c, err := parseDrawColor(a...)
		if err != nil {
			return errObj(err.Error())
		}
		st.fill = c
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
		st.width = int(n.Value)
		if st.width < 1 {
			st.width = 1
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
		st.fontSz = n.Value
		if st.fontSz < 8 {
			st.fontSz = 8
		}
		st.face = nil
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
		draw.Draw(st.img, st.img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
		return object.Nil
	}}
	w.Attrs["נקודה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, err := twoInts("נקודה", a)
		if err != nil {
			return err
		}
		drawDisk(st.img, x, y, st.width/2, st.stroke)
		if st.width < 2 {
			st.img.Set(x, y, st.stroke)
		}
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
		drawThickLine(st.img, vals[0], vals[1], vals[2], vals[3], st.width, st.stroke)
		return object.Nil
	}}
	w.Attrs["מלבן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן", a, 4)
		if err != nil {
			return err
		}
		drawRectOutline(st.img, vals[0], vals[1], vals[2], vals[3], st.width, st.stroke)
		return object.Nil
	}}
	w.Attrs["מלבן_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן_מלא", a, 4)
		if err != nil {
			return err
		}
		drawRectFill(st.img, vals[0], vals[1], vals[2], vals[3], st.fill)
		return object.Nil
	}}
	w.Attrs["עיגול"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("עיגול", a, 3)
		if err != nil {
			return err
		}
		drawCircleOutline(st.img, vals[0], vals[1], vals[2], st.width, st.stroke)
		return object.Nil
	}}
	w.Attrs["עיגול_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("עיגול_מלא", a, 3)
		if err != nil {
			return err
		}
		drawDisk(st.img, vals[0], vals[1], vals[2], st.fill)
		return object.Nil
	}}
	w.Attrs["טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 3 {
			return errObj("טקסט מצפה למחרוזת, x, y")
		}
		s, ok := asString(a[0])
		if !ok {
			s = a[0].Inspect()
		}
		vals, err := nums("טקסט", a[1:], 2)
		if err != nil {
			return err
		}
		if e := drawTextOnBoard(st, s, vals[0], vals[1]); e != nil {
			return errObj(e.Error())
		}
		return object.Nil
	}}
	w.Attrs["תמונה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardDrawImage(st, a...)
	}}
	w.Attrs["שמור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("שמור מצפה לנתיב")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("שמור מצפה לנתיב מחרוזת")
		}
		if err := saveImageFile(st.img, path); err != nil {
			return errObj(err.Error())
		}
		return object.Nil
	}}
	w.Attrs["הצג"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		title := "ציור יוד"
		if len(a) >= 1 {
			if s, ok := asString(a[0]); ok {
				title = s
			}
		}
		return showDrawing(st.img, title)
	}}
	w.Attrs["רוחב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(st.img.Bounds().Dx())}
	}}
	w.Attrs["גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(st.img.Bounds().Dy())}
	}}
	return w
}

func drawLoadImage(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("ציור.טען_תמונה מצפה לנתיב")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("ציור.טען_תמונה מצפה לנתיב מחרוזת")
	}
	img, err := loadImageFile(path)
	if err != nil {
		return errObj(err.Error())
	}
	return wrapImage(img)
}

func wrapImage(img image.Image) *object.GuiWidget {
	st := &drawImage{img: img}
	w := &object.GuiWidget{Kind: "תמונה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["רוחב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(img.Bounds().Dx())}
	}}
	w.Attrs["גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.Number{Value: float64(img.Bounds().Dy())}
	}}
	w.Attrs["שמור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("שמור מצפה לנתיב")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("שמור מצפה לנתיב מחרוזת")
		}
		if err := saveImageFile(img, path); err != nil {
			return errObj(err.Error())
		}
		return object.Nil
	}}
	return w
}

func drawMakeColor(args ...object.Object) object.Object {
	c, err := parseDrawColor(args...)
	if err != nil {
		return errObj(err.Error())
	}
	return &object.Hash{Pairs: map[string]object.Object{
		"אדום":   &object.Number{Value: float64(c.R)},
		"ירוק":   &object.Number{Value: float64(c.G)},
		"כחול":   &object.Number{Value: float64(c.B)},
		"שקיפות": &object.Number{Value: float64(c.A)},
	}}
}

func boardDrawImage(st *drawBoard, args ...object.Object) object.Object {
	if len(args) < 3 {
		return errObj("תמונה מצפה לתמונה/נתיב, x, y [, רוחב, גובה]")
	}
	var src image.Image
	switch v := args[0].(type) {
	case *object.String:
		im, err := loadImageFile(v.Value)
		if err != nil {
			return errObj(err.Error())
		}
		src = im
	case *object.GuiWidget:
		di, ok := v.Data.(*drawImage)
		if !ok {
			return errObj("תמונה מצפה לרכיב תמונה או לנתיב")
		}
		src = di.img
	default:
		return errObj("תמונה מצפה לרכיב תמונה או לנתיב")
	}
	vals, err := nums("תמונה", args[1:], len(args)-1)
	if err != nil {
		return err
	}
	x, y := vals[0], vals[1]
	dw, dh := src.Bounds().Dx(), src.Bounds().Dy()
	if len(vals) >= 4 {
		dw, dh = vals[2], vals[3]
	}
	if dw < 1 || dh < 1 {
		return errObj("גודל תמונה לא תקין")
	}
	dst := image.Rect(x, y, x+dw, y+dh)
	xdraw.CatmullRom.Scale(st.img, dst, src, src.Bounds(), draw.Over, nil)
	return object.Nil
}

func loadImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("לא הצלחתי לפתוח תמונה: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("פורמט תמונה לא נתמך או פגום (png/jpg/gif…): %v", err)
	}
	return img, nil
}

func saveImageFile(img image.Image, path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("לא הצלחתי ליצור קובץ: %v", err)
	}
	defer f.Close()

	// אותו מפת סיביות כמו בתצוגה — RGBA אטום מראשית (0,0)
	rgba := imageToRGBA(img)

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return png.Encode(f, rgba)
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, rgba, &jpeg.Options{Quality: 95})
	case ".gif":
		return gif.Encode(f, rgba, nil)
	default:
		return fmt.Errorf("סיומת לא נתמכת לשמירה: %s (השתמשו ב־.PNG / .JPG / .GIF)", ext)
	}
}

func imageToRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok && r.Bounds().Min.Eq(image.Pt(0, 0)) {
		return r
	}
	b := img.Bounds()
	r := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(r, r.Bounds(), img, b.Min, draw.Src)
	return r
}

func parseDrawColor(args ...object.Object) (color.RGBA, error) {
	if len(args) == 1 {
		if s, ok := asString(args[0]); ok {
			return namedDrawColor(s)
		}
		if h, ok := args[0].(*object.Hash); ok {
			r := hashNum(h, "אדום")
			g := hashNum(h, "ירוק")
			b := hashNum(h, "כחול")
			a := hashNum(h, "שקיפות")
			if a < 0 {
				a = 255
			}
			return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
		}
		return color.RGBA{}, fmt.Errorf("צבע לא תקין")
	}
	if len(args) == 3 || len(args) == 4 {
		vals, err := nums("צבע", args, len(args))
		if err != nil {
			return color.RGBA{}, fmt.Errorf("%s", err.(*object.Error).Message)
		}
		a := 255
		if len(vals) == 4 {
			a = vals[3]
		}
		return color.RGBA{uint8(vals[0]), uint8(vals[1]), uint8(vals[2]), uint8(a)}, nil
	}
	return color.RGBA{}, fmt.Errorf("צבע: שם, או 3/4 מספרי RGB")
}

func namedDrawColor(s string) (color.RGBA, error) {
	switch strings.TrimSpace(s) {
	case "שחור", "black":
		return color.RGBA{0, 0, 0, 255}, nil
	case "לבן", "white":
		return color.RGBA{255, 255, 255, 255}, nil
	case "אדום", "red":
		return color.RGBA{239, 68, 68, 255}, nil
	case "ירוק", "green":
		return color.RGBA{34, 197, 94, 255}, nil
	case "כחול", "blue":
		return color.RGBA{37, 99, 235, 255}, nil
	case "צהוב", "yellow":
		return color.RGBA{250, 204, 21, 255}, nil
	case "כתום", "orange":
		return color.RGBA{249, 115, 22, 255}, nil
	case "סגול", "purple":
		return color.RGBA{168, 85, 247, 255}, nil
	case "ורוד", "pink":
		return color.RGBA{236, 72, 153, 255}, nil
	case "אפור", "gray", "grey":
		return color.RGBA{148, 163, 184, 255}, nil
	case "תכלת", "cyan":
		return color.RGBA{34, 211, 238, 255}, nil
	case "חום", "brown":
		return color.RGBA{146, 64, 14, 255}, nil
	default:
		return color.RGBA{}, fmt.Errorf("צבע לא מוכר: %s", s)
	}
}

func hashNum(h *object.Hash, key string) int {
	v, ok := h.Pairs[key]
	if !ok {
		return -1
	}
	n, ok := v.(*object.Number)
	if !ok {
		return -1
	}
	return int(n.Value)
}

func twoInts(name string, args []object.Object) (int, int, object.Object) {
	vals, err := nums(name, args, 2)
	if err != nil {
		return 0, 0, err
	}
	return vals[0], vals[1], nil
}

func nums(name string, args []object.Object, n int) ([]int, object.Object) {
	if len(args) != n {
		return nil, errObj(fmt.Sprintf("%s מצפה ל־%d מספרים", name, n))
	}
	out := make([]int, n)
	for i, a := range args {
		num, ok := a.(*object.Number)
		if !ok {
			return nil, errObj(fmt.Sprintf("%s מצפה למספרים", name))
		}
		out[i] = int(num.Value)
	}
	return out, nil
}

func drawThickLine(img *image.RGBA, x0, y0, x1, y1, thickness int, c color.RGBA) {
	if thickness < 1 {
		thickness = 1
	}
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length := math.Hypot(dx, dy)
	steps := int(length) + 1
	r := thickness / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(float64(x0) + dx*t)
		y := int(float64(y0) + dy*t)
		drawDisk(img, x, y, r, c)
		if r < 1 {
			setPx(img, x, y, c)
		}
	}
}

func drawRectOutline(img *image.RGBA, x, y, w, h, thickness int, c color.RGBA) {
	if w < 0 {
		x, w = x+w, -w
	}
	if h < 0 {
		y, h = y+h, -h
	}
	drawThickLine(img, x, y, x+w, y, thickness, c)
	drawThickLine(img, x+w, y, x+w, y+h, thickness, c)
	drawThickLine(img, x+w, y+h, x, y+h, thickness, c)
	drawThickLine(img, x, y+h, x, y, thickness, c)
}

func drawRectFill(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w < 0 {
		x, w = x+w, -w
	}
	if h < 0 {
		y, h = y+h, -h
	}
	rect := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
}

func drawCircleOutline(img *image.RGBA, cx, cy, radius, thickness int, c color.RGBA) {
	if radius < 1 {
		setPx(img, cx, cy, c)
		return
	}
	if thickness < 1 {
		thickness = 1
	}
	for t := 0; t < thickness; t++ {
		r := radius - t
		if r < 1 {
			break
		}
		x, y, d := r-1, 0, 1-r
		for x >= y {
			plot8(img, cx, cy, x, y, c)
			y++
			if d < 0 {
				d += 2*y + 1
			} else {
				x--
				d += 2*(y-x) + 1
			}
		}
	}
}

func plot8(img *image.RGBA, cx, cy, x, y int, c color.RGBA) {
	setPx(img, cx+x, cy+y, c)
	setPx(img, cx+y, cy+x, c)
	setPx(img, cx-y, cy+x, c)
	setPx(img, cx-x, cy+y, c)
	setPx(img, cx-x, cy-y, c)
	setPx(img, cx-y, cy-x, c)
	setPx(img, cx+y, cy-x, c)
	setPx(img, cx+x, cy-y, c)
}

func drawDisk(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	if radius < 1 {
		setPx(img, cx, cy, c)
		return
	}
	r2 := radius * radius
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= r2 {
				setPx(img, cx+x, cy+y, c)
			}
		}
	}
}

func setPx(img *image.RGBA, x, y int, c color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	i := img.PixOffset(x, y)
	if c.A == 255 {
		img.Pix[i+0] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = 255
		return
	}
	sr := float64(c.R) * float64(c.A) / 255
	sg := float64(c.G) * float64(c.A) / 255
	sb := float64(c.B) * float64(c.A) / 255
	sa := float64(c.A) / 255
	dr := float64(img.Pix[i+0])
	dg := float64(img.Pix[i+1])
	db := float64(img.Pix[i+2])
	da := float64(img.Pix[i+3]) / 255
	outA := sa + da*(1-sa)
	if outA < 1e-6 {
		return
	}
	img.Pix[i+0] = uint8((sr + dr*da*(1-sa)) / outA)
	img.Pix[i+1] = uint8((sg + dg*da*(1-sa)) / outA)
	img.Pix[i+2] = uint8((sb + db*da*(1-sa)) / outA)
	img.Pix[i+3] = uint8(outA * 255)
}

func drawTextOnBoard(st *drawBoard, text string, x, y int) error {
	face, err := st.ensureFace()
	if err != nil {
		return err
	}
	visual := visualOrderLTR(text)
	d := &font.Drawer{
		Dst:  st.img,
		Src:  image.NewUniform(st.stroke),
		Face: face,
		Dot:  fixed.P(x, y+int(st.fontSz)),
	}
	d.DrawString(visual)
	return nil
}

// visualOrderLTR — סדר חזותי לציור LTR (עברית לא תצא הפוכה)
func visualOrderLTR(s string) string {
	if s == "" {
		return s
	}
	var p bidi.Paragraph
	opts := []bidi.Option{}
	if hasRTLRune(s) {
		opts = append(opts, bidi.DefaultDirection(bidi.RightToLeft))
	}
	_, _ = p.SetString(s, opts...)
	ord, err := p.Order()
	if err != nil || ord.NumRuns() == 0 {
		if hasRTLRune(s) {
			return bidi.ReverseString(s)
		}
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < ord.NumRuns(); i++ {
		run := ord.Run(i)
		part := run.String()
		if run.Direction() == bidi.RightToLeft {
			part = bidi.ReverseString(part)
		}
		b.WriteString(part)
	}
	return b.String()
}

func hasRTLRune(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hebrew, r) || unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

func (st *drawBoard) ensureFace() (font.Face, error) {
	if st.face != nil {
		return st.face, nil
	}
	paths := []string{
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\tahoma.ttf`,
	}
	var last error
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			last = err
			continue
		}
		f, err := opentype.Parse(data)
		if err != nil {
			last = err
			continue
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: st.fontSz,
			DPI:  72,
		})
		if err != nil {
			last = err
			continue
		}
		st.face = face
		return face, nil
	}
	if last == nil {
		last = fmt.Errorf("לא נמצא גופן")
	}
	return nil, fmt.Errorf("לא הצלחתי לטעון גופן לטקסט: %v", last)
}
