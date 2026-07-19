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
	"sync"
	"unicode"

	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/unicode/bidi"

	"yod/internal/draw2d"
	"yod/internal/object"
)

func NewDrawingModule() *object.Module {
	m := &object.Module{Name: "ציור", Attrs: map[string]object.Object{}}
	m.Attrs["לוח"] = &object.Builtin{Fn: drawCreateBoard}
	m.Attrs["טען_תמונה"] = &object.Builtin{Fn: drawLoadImage}
	m.Attrs["טען"] = &object.Builtin{Fn: drawLoadImage} // כינוי ל־טען_תמונה / תמונות.טען
	m.Attrs["צבע"] = &object.Builtin{Fn: drawMakeColor}
	return m
}

type drawBoard struct {
	img    *image.RGBA
	stroke color.RGBA
	fill   color.RGBA
	width  int // עובי קו (ב־DIP)
	fontSz float64
	face   font.Face
	// align: 0 שמאל, 1 ימין, 2 מרכז — קובע איך מפרשים את x בטקסט
	align int
	// dpr — יחס פיקסלים/DIP; ציור/טקסט מוכפלים לחדות ב־HiDPI
	dpr float64
	// backend — ציור משטח (CPU / Direct2D); לוח ציור רגיל יכול להיות nil
	backend draw2d.Backend2D
}

func (st *drawBoard) syncImgFromBackend() {
	if st == nil || st.backend == nil {
		return
	}
	if buf := st.backend.Buffer(); buf != nil {
		st.img = buf
	}
}

func (st *drawBoard) clearBackend(c color.RGBA) {
	if st.backend != nil {
		st.backend.Clear(c)
		st.syncImgFromBackend()
		return
	}
	draw.Draw(st.img, st.img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

// sp — המרת יחידת לוגיקה (DIP) לפיקסל התקן.
func (st *drawBoard) sp(v int) int {
	if st == nil || st.dpr <= 1.001 {
		return v
	}
	return int(math.Round(float64(v) * st.dpr))
}

func (st *drawBoard) scaleVals(vals []int) []int {
	if st == nil || st.dpr <= 1.001 || len(vals) == 0 {
		return vals
	}
	out := make([]int, len(vals))
	for i, v := range vals {
		out[i] = st.sp(v)
	}
	return out
}

func (st *drawBoard) strokePx() int {
	w := st.width
	if w < 1 {
		w = 1
	}
	return st.sp(w)
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
	w.Attrs["קבע_יישור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setBoardTextAlign(st, a...)
	}}
	w.Attrs["קרא_יישור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardReadTextAlign(st)
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
	w.Attrs["מלבן_מעוגל_מלא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		vals, err := nums("מלבן_מעוגל_מלא", a, 5)
		if err != nil {
			return err
		}
		drawRoundedRectFill(st.img, vals[0], vals[1], vals[2], vals[3], vals[4], st.fill)
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
	w.Attrs["טקסט_בתיבה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardDrawTextBox(st, a...)
	}}
	w.Attrs["רוחב_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardTextWidth(st, a...)
	}}
	w.Attrs["מיקום_סמן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardTextCaretInset(st, a...)
	}}
	drawImg := &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return boardDrawImage(st, a...)
	}}
	w.Attrs["תמונה"] = drawImg
	w.Attrs["צייר_תמונה"] = drawImg
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
	return imageLoadFromPath(path)
}

// imageLoadFromPath — ליבת טעינה משותפת לציור ולתמונות.
func imageLoadFromPath(path string) object.Object {
	img, err := loadImageFile(path)
	if err != nil {
		return errObj(err.Error())
	}
	return wrapImage(img)
}

// imageSaveWidget — ליבת שמירה משותפת (תמונות.שמור / מתודת שמור על רכיב).
func imageSaveWidget(obj object.Object, path string, errPrefix string) object.Object {
	gw, ok := obj.(*object.GuiWidget)
	if !ok {
		return errObj(errPrefix + " מצפה לרכיב תמונה")
	}
	di, ok := gw.Data.(*drawImage)
	if !ok || di.img == nil {
		return errObj(errPrefix + " מצפה לרכיב תמונה")
	}
	if err := saveImageFile(di.img, resolveAppPath(path)); err != nil {
		return errObj(err.Error())
	}
	return object.Nil
}

func wrapImage(img image.Image) *object.GuiWidget {
	st := &drawImage{img: img}
	w := &object.GuiWidget{Kind: "תמונה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["רוחב"] = &object.Number{Value: float64(img.Bounds().Dx())}
	w.Attrs["גובה"] = &object.Number{Value: float64(img.Bounds().Dy())}
	w.Attrs["שמור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("שמור מצפה לנתיב")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("שמור מצפה לנתיב מחרוזת")
		}
		return imageSaveWidget(w, path, "שמור")
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
		return errObj("תמונה מצפה לתמונה/נתיב, x, y [, רוחב, גובה [, זווית [, הפוך]]]")
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
	vals, errV := nums("תמונה", args[1:], len(args)-1)
	if errV != nil {
		var perr error
		vals, perr = parseImageDrawNums(args[1:])
		if perr != nil {
			return errObj(perr.Error())
		}
	}
	x, y := vals[0], vals[1]
	dw, dh := src.Bounds().Dx(), src.Bounds().Dy()
	angle := 0.0
	flipH := false
	if len(vals) >= 4 {
		dw, dh = vals[2], vals[3]
	}
	if len(vals) >= 5 {
		angle = float64(vals[4])
	}
	if len(vals) >= 6 {
		flipH = vals[5] != 0
	}
	if len(args) >= 7 {
		if b, ok := args[6].(*object.Boolean); ok {
			flipH = b.Value
		}
	} else if len(args) == 6 {
		if b, ok := args[5].(*object.Boolean); ok {
			flipH = b.Value
		}
	}
	// HiDPI: קואורדינטות וגודל ב־DIP → פיקסלים
	x, y = st.sp(x), st.sp(y)
	dw, dh = st.sp(dw), st.sp(dh)
	if dw < 1 || dh < 1 {
		return errObj("גודל תמונה לא תקין")
	}
	if st.backend != nil {
		st.backend.DrawImage(src, x, y, dw, dh, angle, flipH)
		st.syncImgFromBackend()
		return object.Nil
	}
	src = prepareSprite(src, dw, dh, angle, flipH)
	sb := src.Bounds()
	dst := image.Rect(x, y, x+sb.Dx(), y+sb.Dy())
	draw.Draw(st.img, dst, src, sb.Min, draw.Over)
	return object.Nil
}

func parseImageDrawNums(args []object.Object) ([]int, error) {
	out := make([]int, 0, len(args))
	for _, a := range args {
		switch v := a.(type) {
		case *object.Number:
			out = append(out, int(v.Value))
		case *object.Boolean:
			if v.Value {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
		default:
			return nil, fmt.Errorf("תמונה מצפה למספרים אחרי הנתיב")
		}
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("תמונה מצפה לפחות ל־x, y")
	}
	return out, nil
}

// prepareSprite — שינוי גודל, הפוך אופקי וסיבוב במעלות (סביב המרכז).
func prepareSprite(src image.Image, dw, dh int, angleDeg float64, flipH bool) image.Image {
	scaled := image.NewRGBA(image.Rect(0, 0, dw, dh))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, src.Bounds(), draw.Over, nil)
	if flipH {
		scaled = flipImageHorizontal(scaled)
	}
	if angleDeg == 0 || math.Mod(angleDeg, 360) == 0 {
		return scaled
	}
	return rotateImage(scaled, angleDeg)
}

func flipImageHorizontal(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.Set(w-1-x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

func rotateImage(src *image.RGBA, angleDeg float64) *image.RGBA {
	rad := angleDeg * math.Pi / 180
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)
	b := src.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	cx, cy := w/2, h/2
	corners := [][2]float64{
		{0, 0}, {w, 0}, {0, h}, {w, h},
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range corners {
		dx, dy := c[0]-cx, c[1]-cy
		rx := dx*cosA - dy*sinA
		ry := dx*sinA + dy*cosA
		if rx < minX {
			minX = rx
		}
		if ry < minY {
			minY = ry
		}
		if rx > maxX {
			maxX = rx
		}
		if ry > maxY {
			maxY = ry
		}
	}
	outW := int(math.Ceil(maxX-minX)) + 1
	outH := int(math.Ceil(maxY-minY)) + 1
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}
	out := image.NewRGBA(image.Rect(0, 0, outW, outH))
	ocx, ocy := float64(outW)/2, float64(outH)/2
	cosB := math.Cos(-rad)
	sinB := math.Sin(-rad)
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			dx, dy := float64(x)-ocx, float64(y)-ocy
			sx := dx*cosB - dy*sinB + cx
			sy := dx*sinB + dy*cosB + cy
			out.Set(x, y, sampleBilinear(src, sx, sy))
		}
	}
	return out
}

func sampleBilinear(src *image.RGBA, fx, fy float64) color.Color {
	b := src.Bounds()
	if fx < -1 || fy < -1 || fx > float64(b.Dx()) || fy > float64(b.Dy()) {
		return color.RGBA{}
	}
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := fx - float64(x0)
	ty := fy - float64(y0)
	c00 := rgbaAtClamped(src, x0, y0)
	c10 := rgbaAtClamped(src, x0+1, y0)
	c01 := rgbaAtClamped(src, x0, y0+1)
	c11 := rgbaAtClamped(src, x0+1, y0+1)
	return color.RGBA{
		R: lerpByte(lerpByte(c00.R, c10.R, tx), lerpByte(c01.R, c11.R, tx), ty),
		G: lerpByte(lerpByte(c00.G, c10.G, tx), lerpByte(c01.G, c11.G, tx), ty),
		B: lerpByte(lerpByte(c00.B, c10.B, tx), lerpByte(c01.B, c11.B, tx), ty),
		A: lerpByte(lerpByte(c00.A, c10.A, tx), lerpByte(c01.A, c11.A, tx), ty),
	}
}

func rgbaAtClamped(src *image.RGBA, x, y int) color.RGBA {
	b := src.Bounds()
	if x < 0 || y < 0 || x >= b.Dx() || y >= b.Dy() {
		return color.RGBA{}
	}
	return src.RGBAAt(b.Min.X+x, b.Min.Y+y)
}

func lerpByte(a, b uint8, t float64) uint8 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return uint8(math.Round(float64(a)*(1-t) + float64(b)*t))
}

var (
	imageCacheMu sync.Mutex
	imageCache   = map[string]image.Image{}
)

func resolveAppPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	base := AppBaseDir()
	if base == "" {
		if wd, err := os.Getwd(); err == nil {
			base = wd
		}
	}
	if base != "" {
		return filepath.Join(base, path)
	}
	return path
}

func loadImageFile(path string) (image.Image, error) {
	key := filepath.Clean(resolveAppPath(path))
	imageCacheMu.Lock()
	if img, ok := imageCache[key]; ok {
		imageCacheMu.Unlock()
		return img, nil
	}
	imageCacheMu.Unlock()

	f, err := os.Open(key)
	if err != nil {
		return nil, fmt.Errorf("לא הצלחתי לפתוח תמונה: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("פורמט תמונה לא נתמך או פגום (png/jpg/gif…): %v", err)
	}
	imageCacheMu.Lock()
	imageCache[key] = img
	imageCacheMu.Unlock()
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
	rf := float64(radius) - float64(thickness-1)/2
	half := float64(thickness) / 2
	pad := int(half) + 3
	minY, maxY, minX, maxX := clipDiskBounds(img, cx, cy, radius+pad)
	fcx := float64(cx)
	fcy := float64(cy)
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			cov := sampleAA2x2(px, py, func(fx, fy float64) float64 {
				d := math.Hypot(fx-fcx, fy-fcy)
				return softCoverage(half - math.Abs(d-rf))
			})
			if cov <= 0 {
				continue
			}
			if cov >= 0.999 {
				setPx(img, px, py, c)
			} else {
				setPx(img, px, py, alphaScale(c, cov))
			}
		}
	}
}

func drawDisk(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	if radius < 1 {
		setPx(img, cx, cy, c)
		return
	}
	rf := float64(radius)
	pad := 2
	minY, maxY, minX, maxX := clipDiskBounds(img, cx, cy, radius+pad)
	fcx := float64(cx)
	fcy := float64(cy)
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			cov := sampleAA2x2(px, py, func(fx, fy float64) float64 {
				d := math.Hypot(fx-fcx, fy-fcy)
				return softCoverage(rf - d)
			})
			if cov <= 0 {
				continue
			}
			if cov >= 0.999 {
				setPx(img, px, py, c)
			} else {
				setPx(img, px, py, alphaScale(c, cov))
			}
		}
	}
}

// drawRoundedRectFill — מלבן מעוגל אטום עם אנטי־אליאסינג (SDF + דגימת־על 2×2).
func drawRoundedRectFill(img *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	if w < 0 {
		x, w = x+w, -w
	}
	if h < 0 {
		y, h = y+h, -h
	}
	if w < 1 || h < 1 {
		return
	}
	r := float64(radius)
	if r < 0 {
		r = 0
	}
	if r*2 > float64(w) {
		r = float64(w) / 2
	}
	if r*2 > float64(h) {
		r = float64(h) / 2
	}
	if r < 0.5 {
		drawRectFill(img, x, y, w, h, c)
		return
	}
	fx, fy := float64(x), float64(y)
	fw, fh := float64(w), float64(h)
	pad := 2
	b := img.Bounds()
	minY := y - pad
	maxY := y + h + pad
	minX := x - pad
	maxX := x + w + pad
	if minY < b.Min.Y {
		minY = b.Min.Y
	}
	if maxY >= b.Max.Y {
		maxY = b.Max.Y - 1
	}
	if minX < b.Min.X {
		minX = b.Min.X
	}
	if maxX >= b.Max.X {
		maxX = b.Max.X - 1
	}
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			cov := sampleAA2x2(px, py, func(sx, sy float64) float64 {
				return softCoverage(-sdRoundedRect(sx, sy, fx, fy, fw, fh, r))
			})
			if cov <= 0 {
				continue
			}
			if cov >= 0.999 {
				setPx(img, px, py, c)
			} else {
				setPx(img, px, py, alphaScale(c, cov))
			}
		}
	}
}

// sdRoundedRect — מרחק חתום למלבן מעוגל (שלילי בפנים).
func sdRoundedRect(px, py, x, y, w, h, r float64) float64 {
	cx := x + w/2
	cy := y + h/2
	bx := w/2 - r
	by := h/2 - r
	dx := math.Abs(px-cx) - bx
	dy := math.Abs(py-cy) - by
	return math.Hypot(math.Max(dx, 0), math.Max(dy, 0)) + math.Min(math.Max(dx, dy), 0) - r
}

// softCoverage — מעבר רך ~1.4 פיקסל סביב הקצה (מבוסס מרחק מהגבול; חיובי = בפנים).
func softCoverage(insideDist float64) float64 {
	const soft = 1.4
	v := (insideDist + soft*0.5) / soft
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	// smoothstep — פחות "פס" חד מכיסוי ליניארי
	return v * v * (3 - 2*v)
}

// sampleAA2x2 — ממוצע 4 דגימות בתוך הפיקסל.
func sampleAA2x2(px, py int, sample func(fx, fy float64) float64) float64 {
	const o0 = 0.25
	const o1 = 0.75
	fx0 := float64(px)
	fy0 := float64(py)
	return (sample(fx0+o0, fy0+o0) + sample(fx0+o1, fy0+o0) +
		sample(fx0+o0, fy0+o1) + sample(fx0+o1, fy0+o1)) * 0.25
}

func clipDiskBounds(img *image.RGBA, cx, cy, extent int) (minY, maxY, minX, maxX int) {
	b := img.Bounds()
	minY = cy - extent
	maxY = cy + extent
	minX = cx - extent
	maxX = cx + extent
	if minY < b.Min.Y {
		minY = b.Min.Y
	}
	if maxY >= b.Max.Y {
		maxY = b.Max.Y - 1
	}
	if minX < b.Min.X {
		minX = b.Min.X
	}
	if maxX >= b.Max.X {
		maxX = b.Max.X - 1
	}
	return
}

// alphaScale — משנה שקיפות לפי כיסוי (0..1) לאנטי־אליאסינג.
func alphaScale(c color.RGBA, cov float64) color.RGBA {
	if cov >= 1 {
		return c
	}
	if cov <= 0 {
		return color.RGBA{}
	}
	a := float64(c.A) * cov
	if a > 255 {
		a = 255
	}
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: uint8(a + 0.5)}
}

func setPx(img *image.RGBA, x, y int, c color.RGBA) {
	if c.A == 0 {
		return
	}
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


func setBoardTextAlign(st *drawBoard, a ...object.Object) object.Object {
	if len(a) != 1 {
		return errObj("קבע_יישור מצפה ל־שמאל / ימין / מרכז")
	}
	s, ok := asString(a[0])
	if !ok {
		return errObj("קבע_יישור מצפה למחרוזת: שמאל, ימין או מרכז")
	}
	switch s {
	case "שמאל", "לשמאל", "left":
		st.align = 0
	case "ימין", "לימין", "right":
		st.align = 1
	case "מרכז", "center":
		st.align = 2
	default:
		return errObj("קבע_יישור: ערך לא מוכר — השתמשו ב־שמאל, ימין או מרכז")
	}
	return object.Nil
}

func boardReadTextAlign(st *drawBoard) object.Object {
	switch st.align {
	case 1:
		return &object.String{Value: "ימין"}
	case 2:
		return &object.String{Value: "מרכז"}
	default:
		return &object.String{Value: "שמאל"}
	}
}

func boardDrawTextBox(st *drawBoard, a ...object.Object) object.Object {
	if len(a) != 4 {
		return errObj("טקסט_בתיבה מצפה למחרוזת, x, y, רוחב")
	}
	s, ok := asString(a[0])
	if !ok {
		s = a[0].Inspect()
	}
	vals, err := nums("טקסט_בתיבה", a[1:], 3)
	if err != nil {
		return err
	}
	x, y, boxW := vals[0], vals[1], vals[2]
	if boxW < 1 {
		return errObj("טקסט_בתיבה: רוחב חייב להיות חיובי")
	}
	prev := st.align
	st.align = 1 // יישור לימין בתוך התיבה
	defer func() { st.align = prev }()
	if e := drawTextOnBoard(st, s, x+boxW, y); e != nil {
		return errObj(e.Error())
	}
	return object.Nil
}

func measureBoardText(st *drawBoard, text string) (float64, string, error) {
	face, err := st.ensureFace()
	if err != nil {
		return 0, "", err
	}
	// יישור ימין = פסקה RTL (כמו עורך יוד); שמאל/מרכז = LTR
	visual := visualOrderForAlign(text, st.align)
	if visual == "" {
		return 0, visual, nil
	}
	adv := font.MeasureString(face, visual)
	return float64(adv) / 64.0, visual, nil
}

func drawTextOnBoard(st *drawBoard, text string, x, y int) error {
	face, err := st.ensureFace()
	if err != nil {
		return err
	}
	w, visual, err := measureBoardText(st, text)
	if err != nil {
		return err
	}
	x, y = st.sp(x), st.sp(y)
	drawX := x
	switch st.align {
	case 1: // ימין — x הוא הקצה הימני של הטקסט
		drawX = x - int(math.Ceil(w))
	case 2: // מרכז — x הוא מרכז הטקסט
		drawX = x - int(math.Ceil(w/2))
	}
	d := &font.Drawer{
		Dst:  st.img,
		Src:  image.NewUniform(st.stroke),
		Face: face,
		Dot:  fixed.P(drawX, y+int(sizeForFace(st))),
	}
	d.DrawString(visual)
	return nil
}

func boardTextWidth(st *drawBoard, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רוחב_טקסט מצפה למחרוזת")
	}
	s, ok := asString(args[0])
	if !ok {
		if args[0] == nil || args[0] == object.Nil {
			s = ""
		} else {
			return errObj("רוחב_טקסט מצפה למחרוזת")
		}
	}
	if s == "" {
		return &object.Number{Value: 0}
	}
	w, _, err := measureBoardText(st, s)
	if err != nil {
		return errObj(err.Error())
	}
	if st.dpr > 1.001 {
		w = w / st.dpr
	}
	return &object.Number{Value: w}
}

// boardTextCaretInset — מרחק הסמן מצד תחילת הפסקה (ימין ב־RTL, שמאל ב־LTR).
// תואם בדיוק לציור BiDi (אותו סדר חזותי) — עברית + אנגלית + מספרים כמו וורד.
func boardTextCaretInset(st *drawBoard, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("מיקום_סמן מצפה למחרוזת ולאינדקס")
	}
	s, ok := asString(args[0])
	if !ok {
		if args[0] == nil || args[0] == object.Nil {
			s = ""
		} else {
			return errObj("מיקום_סמן מצפה למחרוזת ולאינדקס")
		}
	}
	n, ok := args[1].(*object.Number)
	if !ok {
		return errObj("מיקום_סמן מצפה למחרוזת ולאינדקס")
	}
	rs := []rune(s)
	caret := int(n.Value)
	if caret < 0 {
		caret = 0
	}
	if caret > len(rs) {
		caret = len(rs)
	}
	rtlPara := st.align == 1

	face, err := st.ensureFace()
	if err != nil {
		return errObj(err.Error())
	}
	fromLeft := caretOffsetFromTextLeft(face, s, caret, rtlPara)
	if !rtlPara {
		return &object.Number{Value: fromLeft}
	}
	disp, _, _ := buildVisualMap(s, true)
	fullW := measureFaceAdvance(face, disp)
	inset := fullW - fromLeft
	if inset < 0 {
		inset = 0
	}
	return &object.Number{Value: inset}
}

func measureFaceAdvance(face font.Face, s string) float64 {
	if s == "" {
		return 0
	}
	return float64(font.MeasureString(face, s)) / 64.0
}

func paragraphOrder(text string, rtlPara bool) (bidi.Ordering, bool) {
	var p bidi.Paragraph
	opts := []bidi.Option{}
	if rtlPara {
		opts = append(opts, bidi.DefaultDirection(bidi.RightToLeft))
	} else {
		opts = append(opts, bidi.DefaultDirection(bidi.LeftToRight))
	}
	_, err := p.SetString(text, opts...)
	if err != nil {
		return bidi.Ordering{}, false
	}
	ord, err := p.Order()
	if err != nil || ord.NumRuns() == 0 {
		return bidi.Ordering{}, false
	}
	return ord, true
}

// buildVisualMap — סדר חזותי שמאלי→ימין כמו וורד/פסקה RTL.
// Order() של x/text מחזיר ריצות לפי סדר לוגי; בפסקת RTL הופכים את סדר הריצות
// כדי שאנגלית/מספרים יופיעו משמאל לעברית (כמו בוורד).
func buildVisualMap(text string, rtlPara bool) (display string, logicalOfVisual []int, fromRTL []bool) {
	rs := []rune(text)
	n := len(rs)
	if n == 0 {
		return "", nil, nil
	}
	ord, ok := paragraphOrder(text, rtlPara)
	if !ok {
		logicalOfVisual = make([]int, n)
		fromRTL = make([]bool, n)
		if rtlPara && hasRTLRune(text) {
			out := make([]rune, n)
			for i := 0; i < n; i++ {
				out[i] = rs[n-1-i]
				logicalOfVisual[i] = n - 1 - i
				fromRTL[i] = true
			}
			return string(out), logicalOfVisual, fromRTL
		}
		for i := 0; i < n; i++ {
			logicalOfVisual[i] = i
		}
		return text, logicalOfVisual, fromRTL
	}

	type runInfo struct {
		start, end int
		rtl        bool
	}
	runs := make([]runInfo, 0, ord.NumRuns())
	for i := 0; i < ord.NumRuns(); i++ {
		run := ord.Run(i)
		start, endIncl := run.Pos()
		end := endIncl + 1
		if start < 0 {
			start = 0
		}
		if end > n {
			end = n
		}
		runs = append(runs, runInfo{start: start, end: end, rtl: run.Direction() == bidi.RightToLeft})
	}
	// פסקת RTL: הריצה הראשונה לוגית יושבת מימין — הופכים לסדר ציור משמאל לימין
	if rtlPara {
		for i, j := 0, len(runs)-1; i < j; i, j = i+1, j-1 {
			runs[i], runs[j] = runs[j], runs[i]
		}
	}

	var out []rune
	logicalOfVisual = make([]int, 0, n)
	fromRTL = make([]bool, 0, n)
	for _, run := range runs {
		if run.rtl {
			for j := run.end - 1; j >= run.start; j-- {
				out = append(out, rs[j])
				logicalOfVisual = append(logicalOfVisual, j)
				fromRTL = append(fromRTL, true)
			}
		} else {
			for j := run.start; j < run.end; j++ {
				out = append(out, rs[j])
				logicalOfVisual = append(logicalOfVisual, j)
				fromRTL = append(fromRTL, false)
			}
		}
	}
	return string(out), logicalOfVisual, fromRTL
}

// caretOffsetFromTextLeft — מרחק הסמן מקצה שמאל של המחרוזת החזותית (אותה מחרוזת שצוירים).
// מדידה רק על קידומות של הסדר החזותי — תואם ל־MeasureString של הציור המלא.
func caretOffsetFromTextLeft(face font.Face, text string, caret int, rtlPara bool) float64 {
	rs := []rune(text)
	n := len(rs)
	if n == 0 {
		return 0
	}
	if caret < 0 {
		caret = 0
	}
	if caret > n {
		caret = n
	}

	disp, mapping, fromRTL := buildVisualMap(text, rtlPara)
	drunes := []rune(disp)
	if len(drunes) != len(mapping) || len(mapping) != n || len(fromRTL) != n {
		if !rtlPara {
			return measureFaceAdvance(face, string(rs[:caret]))
		}
		return measureFaceAdvance(face, disp)
	}

	visOf := func(log int) int {
		for vi, li := range mapping {
			if li == log {
				return vi
			}
		}
		return -1
	}

	measurePrefix := func(visCount int) float64 {
		if visCount <= 0 {
			return 0
		}
		if visCount >= len(drunes) {
			return measureFaceAdvance(face, disp)
		}
		return measureFaceAdvance(face, string(drunes[:visCount]))
	}

	if caret == 0 {
		vi := visOf(0)
		if vi < 0 {
			return 0
		}
		if fromRTL[vi] {
			return measurePrefix(vi + 1) // leading RTL = ימין הגליף
		}
		return measurePrefix(vi) // leading LTR = שמאל
	}

	log := caret - 1
	vi := visOf(log)
	if vi < 0 {
		return measurePrefix(len(drunes))
	}
	if fromRTL[vi] {
		return measurePrefix(vi) // trailing RTL = שמאל הגליף
	}
	return measurePrefix(vi + 1) // trailing LTR/ספרות = ימין
}

// visualOrderForAlign — סדר חזותי לציור; זהה ל־buildVisualMap.
func visualOrderForAlign(s string, align int) string {
	if s == "" {
		return s
	}
	rtlPara := align == 1
	if !rtlPara && !hasRTLRune(s) {
		return s
	}
	disp, _, _ := buildVisualMap(s, rtlPara)
	return disp
}

// visualOrderRTL — תאימות לשם ישן; ברירת פסקה RTL כשיש תווים עבריים/ערביים
func visualOrderRTL(s string) string {
	if hasRTLRune(s) {
		return visualOrderForAlign(s, 1)
	}
	return visualOrderForAlign(s, 0)
}

// visualOrderLTR — תאימות לשם ישן
func visualOrderLTR(s string) string {
	return visualOrderForAlign(s, 0)
}

func hasRTLRune(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hebrew, r) || unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

func hasLatinLetter(s string) bool {
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			return true
		}
		if unicode.Is(unicode.Latin, r) {
			return true
		}
	}
	return false
}

func hasASCIIDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func sizeForFace(st *drawBoard) float64 {
	sz := st.fontSz
	if sz < 8 {
		sz = 8
	}
	if st.dpr > 1.001 {
		sz = sz * st.dpr
	}
	return sz
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
			Size: sizeForFace(st),
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

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
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
