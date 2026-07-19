package draw2d

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// CPUBackend — ציור ב־RGBA (fallback / !windows / כשל D2D).
type CPUBackend struct {
	mu     sync.Mutex
	img    *image.RGBA
	dpi    int
	face   font.Face
	fontSz float64
}

func NewCPUBackend(physW, physH, dpi int) *CPUBackend {
	if physW < 1 {
		physW = 1
	}
	if physH < 1 {
		physH = 1
	}
	if dpi < 96 {
		dpi = 96
	}
	img := image.NewRGBA(image.Rect(0, 0, physW, physH))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	return &CPUBackend{img: img, dpi: dpi, fontSz: 16}
}

func (b *CPUBackend) Name() string { return "cpu" }

func (b *CPUBackend) BindHWND(hwnd uintptr) error { return nil }

func (b *CPUBackend) Resize(physW, physH, dpi int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if physW < 1 {
		physW = 1
	}
	if physH < 1 {
		physH = 1
	}
	if dpi >= 96 {
		b.dpi = dpi
	}
	ow, oh := 0, 0
	var bg color.RGBA
	if b.img != nil {
		ow, oh = b.img.Bounds().Dx(), b.img.Bounds().Dy()
		if ow > 0 && oh > 0 {
			bg = b.img.RGBAAt(0, 0)
		}
	}
	if ow == physW && oh == physH {
		return nil
	}
	if bg.A == 0 && bg.R == 0 && bg.G == 0 && bg.B == 0 {
		bg = color.RGBA{255, 255, 255, 255}
	}
	neu := image.NewRGBA(image.Rect(0, 0, physW, physH))
	draw.Draw(neu, neu.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	if b.img != nil && ow > 0 && oh > 0 {
		xdraw.CatmullRom.Scale(neu, neu.Bounds(), b.img, b.img.Bounds(), draw.Src, nil)
	}
	b.img = neu
	b.face = nil
	return nil
}

func (b *CPUBackend) Clear(c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	draw.Draw(b.img, b.img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func (b *CPUBackend) FillRect(x, y, w, h int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawRectFill(b.img, x, y, w, h, c)
}

func (b *CPUBackend) StrokeRect(x, y, w, h, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawRectOutline(b.img, x, y, w, h, thickness, c)
}

func (b *CPUBackend) FillRoundedRect(x, y, w, h, radius int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawRoundedRectFill(b.img, x, y, w, h, radius, c)
}

func (b *CPUBackend) StrokeLine(x0, y0, x1, y1, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawThickLine(b.img, x0, y0, x1, y1, thickness, c)
}

func (b *CPUBackend) FillCircle(cx, cy, radius int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawDisk(b.img, cx, cy, radius, c)
}

func (b *CPUBackend) StrokeCircle(cx, cy, radius, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawCircleOutline(b.img, cx, cy, radius, thickness, c)
}

func (b *CPUBackend) StrokeEllipse(cx, cy, rx, ry, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	drawEllipseOutline(b.img, cx, cy, rx, ry, thickness, c)
}

func (b *CPUBackend) DrawImage(src image.Image, x, y, dw, dh int, angleDeg float64, flipH bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if dw < 1 || dh < 1 {
		return
	}
	spr := prepareSprite(src, dw, dh, angleDeg, flipH)
	sb := spr.Bounds()
	dst := image.Rect(x, y, x+sb.Dx(), y+sb.Dy())
	draw.Draw(b.img, dst, spr, sb.Min, draw.Over)
}

func (b *CPUBackend) DrawText(s string, x, y int, fontSize float64, c color.RGBA, align int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if fontSize < 8 {
		fontSize = 8
	}
	face := b.ensureFaceLocked(fontSize)
	adv := float64(font.MeasureString(face, s)) / 64.0
	drawX := x
	switch align {
	case AlignRight:
		drawX = x - int(adv+0.5)
	case AlignCenter:
		drawX = x - int(adv/2+0.5)
	}
	d := &font.Drawer{
		Dst:  b.img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(drawX, y+int(fontSize)),
	}
	d.DrawString(s)
}

func (b *CPUBackend) ensureFaceLocked(sz float64) font.Face {
	if b.face != nil && b.fontSz == sz {
		return b.face
	}
	b.fontSz = sz
	// basicfont — מספיק ל־fallback; משטח Windows משתמש בדרך כלל ב־D2D/DirectWrite
	b.face = basicfont.Face7x13
	return b.face
}

func (b *CPUBackend) FloodFill(x, y int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	floodFill(b.img, x, y, c)
}

func (b *CPUBackend) Present() error { return nil }

func (b *CPUBackend) SnapshotRGBA() (*image.RGBA, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneRGBA(b.img), nil
}

func (b *CPUBackend) ReplacePixels(src *image.RGBA) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if src == nil {
		return fmt.Errorf("ReplacePixels: nil")
	}
	b.img = cloneRGBA(src)
	return nil
}

func (b *CPUBackend) Buffer() *image.RGBA {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.img
}

func (b *CPUBackend) Destroy() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.img = nil
	b.face = nil
}
