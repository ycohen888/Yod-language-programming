package draw2d

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

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

func sdRoundedRect(px, py, x, y, w, h, r float64) float64 {
	cx := x + w/2
	cy := y + h/2
	bx := w/2 - r
	by := h/2 - r
	dx := math.Abs(px-cx) - bx
	dy := math.Abs(py-cy) - by
	return math.Hypot(math.Max(dx, 0), math.Max(dy, 0)) + math.Min(math.Max(dx, dy), 0) - r
}

func softCoverage(insideDist float64) float64 {
	const soft = 1.4
	v := (insideDist + soft*0.5) / soft
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v * v * (3 - 2*v)
}

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

func drawEllipseOutline(img *image.RGBA, cx, cy, rx, ry, thickness int, c color.RGBA) {
	if rx < 1 && ry < 1 {
		setPx(img, cx, cy, c)
		return
	}
	if thickness < 1 {
		thickness = 1
	}
	half := float64(thickness) / 2
	pad := int(half) + 3
	minY, maxY, minX, maxX := clipDiskBounds(img, cx, cy, int(math.Max(float64(rx), float64(ry)))+pad)
	fcx, fcy := float64(cx), float64(cy)
	frx, fry := float64(rx), float64(ry)
	if frx < 0.5 {
		frx = 0.5
	}
	if fry < 0.5 {
		fry = 0.5
	}
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			cov := sampleAA2x2(px, py, func(fx, fy float64) float64 {
				nx := (fx - fcx) / frx
				ny := (fy - fcy) / fry
				d := math.Hypot(nx, ny)
				// מרחק בקירוב לפיקסל בצד החיצוני
				grad := math.Hypot(nx*frx, ny*fry)
				if grad < 1e-6 {
					grad = 1
				}
				distPx := (d - 1) * grad / d
				if d < 1e-6 {
					distPx = -math.Min(frx, fry)
				}
				return softCoverage(half - math.Abs(distPx))
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

func floodFill(img *image.RGBA, x, y int, repl color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
		return
	}
	target := img.RGBAAt(x, y)
	if target == repl {
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
		if img.RGBAAt(p.x, p.y) != target {
			continue
		}
		img.SetRGBA(p.x, p.y, repl)
		stack = append(stack, pt{p.x + 1, p.y}, pt{p.x - 1, p.y}, pt{p.x, p.y + 1}, pt{p.x, p.y - 1})
	}
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	dst := image.NewRGBA(b)
	copy(dst.Pix, src.Pix)
	return dst
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

func prepareSprite(src image.Image, dw, dh int, angleDeg float64, flipH bool) *image.RGBA {
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
	corners := [][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}}
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
