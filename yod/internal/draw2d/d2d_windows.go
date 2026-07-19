package draw2d

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	d2d1DLL   = windows.NewLazySystemDLL("d2d1.dll")
	dwriteDLL = windows.NewLazySystemDLL("dwrite.dll")
	procD2D1CreateFactory = d2d1DLL.NewProc("D2D1CreateFactory")
	procDWriteCreateFactory = dwriteDLL.NewProc("DWriteCreateFactory")
)

// GUIDs
var (
	iidID2D1Factory = windows.GUID{Data1: 0x06152247, Data2: 0x6f50, Data3: 0x465a, Data4: [8]byte{0x92, 0x45, 0x11, 0x8b, 0xfd, 0x3b, 0x60, 0x07}}
	iidIDWriteFactory = windows.GUID{Data1: 0xb859ee5a, Data2: 0xd838, Data3: 0x4b5b, Data4: [8]byte{0xa2, 0xe8, 0x1a, 0xdc, 0x7d, 0x93, 0xdb, 0x48}}
)

const (
	d2d1FactoryTypeSingleThreaded = 0
	dwriteFactoryTypeShared       = 0

	d2d1AlphaModePremultiplied = 1
	d2d1AlphaModeIgnore        = 3
	d2d1PixelFormatUnknown     = 0
	dxgiFormatB8G8R8A8UNORM    = 87

	d2d1RenderTargetTypeDefault     = 0
	d2d1RenderTargetUsageNone       = 0
	d2d1FeatureLevelDefault         = 0
	d2d1PresentOptionsNone          = 0
	d2d1WindowStateNone             = 0
	d2d1BitmapInterpolationLinear   = 1
	d2d1DrawTextOptionsNone         = 0
	d2d1AntialiasModePerPrimitive  = 0
	dwriteFontWeightRegular         = 400
	dwriteFontStyleNormal           = 0
	dwriteFontStretchNormal         = 5
	dwriteTextAlignmentLeading      = 0
	dwriteTextAlignmentTrailing     = 1
	dwriteTextAlignmentCenter       = 2
	dwriteParagraphAlignmentNear    = 0
	dwriteReadingDirectionRTL       = 1
	dwriteReadingDirectionLTR       = 0
	dwriteWordWrappingNoWrap        = 1
)

type d2dColorF struct{ R, G, B, A float32 }
type d2dSizeU struct{ Width, Height uint32 }
type d2dSizeF struct{ Width, Height float32 }
type d2dPoint2F struct{ X, Y float32 }
type d2dRectF struct{ Left, Top, Right, Bottom float32 }
type d2dRoundedRect struct {
	Rect   d2dRectF
	RadiusX, RadiusY float32
}
type d2dEllipse struct {
	Point      d2dPoint2F
	RadiusX, RadiusY float32
}
type d2dMatrix3x2F struct {
	M11, M12 float32
	M21, M22 float32
	M31, M32 float32
}
type d2dPixelFormat struct {
	Format    uint32
	AlphaMode uint32
}
type d2dRenderTargetProps struct {
	Type        uint32
	PixelFormat d2dPixelFormat
	DpiX, DpiY  float32
	Usage       uint32
	MinLevel    uint32
}
type d2dHwndRTProps struct {
	Hwnd           uintptr
	PixelSize      d2dSizeU
	PresentOptions uint32
}
type d2dBitmapProps struct {
	PixelFormat d2dPixelFormat
	DpiX, DpiY  float32
}

func comVtbl(obj uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(obj))
}

func comCall(obj uintptr, idx int, a ...uintptr) (uintptr, uintptr, syscall.Errno) {
	vt := comVtbl(obj)
	fn := *(*uintptr)(unsafe.Pointer(vt + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	var args []uintptr
	args = append(args, obj)
	args = append(args, a...)
	switch len(args) {
	case 1:
		return syscall.Syscall(fn, 1, args[0], 0, 0)
	case 2:
		return syscall.Syscall(fn, 2, args[0], args[1], 0)
	case 3:
		return syscall.Syscall(fn, 3, args[0], args[1], args[2])
	case 4:
		return syscall.Syscall6(fn, 4, args[0], args[1], args[2], args[3], 0, 0)
	case 5:
		return syscall.Syscall6(fn, 5, args[0], args[1], args[2], args[3], args[4], 0)
	case 6:
		return syscall.Syscall6(fn, 6, args[0], args[1], args[2], args[3], args[4], args[5])
	case 7:
		return syscall.Syscall9(fn, 7, args[0], args[1], args[2], args[3], args[4], args[5], args[6], 0, 0)
	case 8:
		return syscall.Syscall9(fn, 8, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], 0)
	case 9:
		return syscall.Syscall9(fn, 9, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8])
	default:
		return syscall.SyscallN(fn, args...)
	}
}

func comRelease(obj uintptr) {
	if obj != 0 {
		comCall(obj, 2)
	}
}

func hrOK(r uintptr) bool { return int32(r) >= 0 }

func rgbaToD2D(c color.RGBA) d2dColorF {
	return d2dColorF{
		R: float32(c.R) / 255,
		G: float32(c.G) / 255,
		B: float32(c.B) / 255,
		A: float32(c.A) / 255,
	}
}

// D2DBackend — Direct2D + DirectWrite; שומר צל CPU ל־Snapshot/FloodFill.
type D2DBackend struct {
	mu sync.Mutex
	cpu *CPUBackend

	factory uintptr
	hwndRT  uintptr
	dwFact  uintptr
	hwnd    uintptr
	dpi     int
	w, h    int

	drawing bool // BeginDraw פתוח על hwndRT
	ok      bool
	cpuBlit bool // צריך העלאת בופר CPU ב־Present (אחרי הצפה/Replace/יצירה)
}

// NewD2DBackend יוצר באקאנד D2D; נכשל → שגיאה (הקורא יעבור ל־CPU).
func NewD2DBackend(physW, physH, dpi int) (*D2DBackend, error) {
	if err := d2d1DLL.Load(); err != nil {
		return nil, err
	}
	if dpi < 96 {
		dpi = 96
	}
	var factory uintptr
	hr, _, _ := procD2D1CreateFactory.Call(
		uintptr(d2d1FactoryTypeSingleThreaded),
		uintptr(unsafe.Pointer(&iidID2D1Factory)),
		0,
		uintptr(unsafe.Pointer(&factory)),
	)
	if !hrOK(hr) || factory == 0 {
		return nil, fmt.Errorf("D2D1CreateFactory: 0x%08x", uint32(hr))
	}
	b := &D2DBackend{
		cpu:     NewCPUBackend(physW, physH, dpi),
		factory: factory,
		dpi:     dpi,
		w:       physW,
		h:       physH,
		ok:      true,
		cpuBlit: true,
	}
	if err := dwriteDLL.Load(); err == nil {
		var dw uintptr
		hr, _, _ = procDWriteCreateFactory.Call(
			uintptr(dwriteFactoryTypeShared),
			uintptr(unsafe.Pointer(&iidIDWriteFactory)),
			uintptr(unsafe.Pointer(&dw)),
		)
		if hrOK(hr) && dw != 0 {
			b.dwFact = dw
		}
	}
	return b, nil
}

func (b *D2DBackend) Name() string { return "direct2d" }

func (b *D2DBackend) BindHWND(hwnd uintptr) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.ok || hwnd == 0 {
		return nil
	}
	if b.hwnd == hwnd && b.hwndRT != 0 {
		return nil
	}
	b.releaseRTLocked()
	b.hwnd = hwnd
	return b.createRTLocked()
}

func (b *D2DBackend) releaseRTLocked() {
	b.endDrawLocked()
	if b.hwndRT != 0 {
		comRelease(b.hwndRT)
		b.hwndRT = 0
	}
}

func (b *D2DBackend) createRTLocked() error {
	if b.factory == 0 || b.hwnd == 0 {
		return nil
	}
	// DPI 96 — יחידות ציור = פיקסלים טבעיים (ה־API של יוד כבר המיר DIP→פיקסל).
	props := d2dRenderTargetProps{
		Type: d2d1RenderTargetTypeDefault,
		PixelFormat: d2dPixelFormat{
			Format:    d2d1PixelFormatUnknown,
			AlphaMode: 0, // UNKNOWN — מתאים ל־HWND
		},
		DpiX:     96,
		DpiY:     96,
		Usage:    d2d1RenderTargetUsageNone,
		MinLevel: d2d1FeatureLevelDefault,
	}
	hwndProps := d2dHwndRTProps{
		Hwnd: b.hwnd,
		PixelSize: d2dSizeU{
			Width:  uint32(b.w),
			Height: uint32(b.h),
		},
		PresentOptions: d2d1PresentOptionsNone,
	}
	var rt uintptr
	// ID2D1Factory::CreateHwndRenderTarget = vtable index 14
	hr, _, _ := comCall(b.factory, 14,
		uintptr(unsafe.Pointer(&props)),
		uintptr(unsafe.Pointer(&hwndProps)),
		uintptr(unsafe.Pointer(&rt)),
	)
	if !hrOK(hr) || rt == 0 {
		return fmt.Errorf("CreateHwndRenderTarget: 0x%08x", uint32(hr))
	}
	b.hwndRT = rt
	comCall(rt, 51, uintptr(math.Float32bits(96)), uintptr(math.Float32bits(96)))
	// אנטי־אליאסינג לצורות + טקסט ClearType
	comCall(rt, 32, uintptr(d2d1AntialiasModePerPrimitive)) // SetAntialiasMode
	const d2d1TextAntialiasCleartype = 1
	comCall(rt, 34, uintptr(d2d1TextAntialiasCleartype)) // SetTextAntialiasMode
	b.cpuBlit = true
	return nil
}

func (b *D2DBackend) Resize(physW, physH, dpi int) error {
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
	_ = b.cpu.Resize(physW, physH, b.dpi)
	b.w, b.h = physW, physH
	if b.hwndRT != 0 {
		b.endDrawLocked()
		sz := d2dSizeU{Width: uint32(physW), Height: uint32(physH)}
		// ID2D1HwndRenderTarget::Resize = index 58 (after RT methods + CheckWindowState)
		hr, _, _ := comCall(b.hwndRT, 58, uintptr(unsafe.Pointer(&sz)))
		if !hrOK(hr) {
			b.releaseRTLocked()
			return b.createRTLocked()
		}
		comCall(b.hwndRT, 51, uintptr(math.Float32bits(96)), uintptr(math.Float32bits(96)))
	}
	return nil
}

func packSizeU(w, h int) uintptr {
	return uintptr(uint32(w)) | (uintptr(uint32(h)) << 32)
}

func packPointF(x, y float32) uintptr {
	return uintptr(math.Float32bits(x)) | (uintptr(math.Float32bits(y)) << 32)
}

func (b *D2DBackend) beginDrawLocked() {
	if b.hwndRT == 0 || b.drawing {
		return
	}
	comCall(b.hwndRT, 48) // BeginDraw
	b.drawing = true
}

func (b *D2DBackend) endDrawLocked() {
	if b.hwndRT == 0 || !b.drawing {
		b.drawing = false
		return
	}
	comCall(b.hwndRT, 49, 0, 0) // EndDraw
	b.drawing = false
}

func (b *D2DBackend) solidBrush(c color.RGBA) uintptr {
	if b.hwndRT == 0 {
		return 0
	}
	col := rgbaToD2D(c)
	var brush uintptr
	// CreateSolidColorBrush = 8
	hr, _, _ := comCall(b.hwndRT, 8, uintptr(unsafe.Pointer(&col)), 0, uintptr(unsafe.Pointer(&brush)))
	if !hrOK(hr) {
		return 0
	}
	return brush
}

func (b *D2DBackend) Clear(c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.Clear(c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	col := rgbaToD2D(c)
	comCall(b.hwndRT, 47, uintptr(unsafe.Pointer(&col))) // Clear
}

func (b *D2DBackend) FillRect(x, y, w, h int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.FillRect(x, y, w, h, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	r := d2dRectF{float32(x), float32(y), float32(x + w), float32(y + h)}
	comCall(b.hwndRT, 17, uintptr(unsafe.Pointer(&r)), br) // FillRectangle
}

func (b *D2DBackend) StrokeRect(x, y, w, h, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.StrokeRect(x, y, w, h, thickness, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	r := d2dRectF{float32(x), float32(y), float32(x + w), float32(y + h)}
	// DrawRectangle(rect, brush, strokeWidth, strokeStyle)
	comCall(b.hwndRT, 16, uintptr(unsafe.Pointer(&r)), br, uintptr(math.Float32bits(float32(thickness))), 0)
}

func (b *D2DBackend) FillRoundedRect(x, y, w, h, radius int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.FillRoundedRect(x, y, w, h, radius, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	rr := d2dRoundedRect{
		Rect:    d2dRectF{float32(x), float32(y), float32(x + w), float32(y + h)},
		RadiusX: float32(radius),
		RadiusY: float32(radius),
	}
	comCall(b.hwndRT, 19, uintptr(unsafe.Pointer(&rr)), br) // FillRoundedRectangle
}

func (b *D2DBackend) StrokeLine(x0, y0, x1, y1, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.StrokeLine(x0, y0, x1, y1, thickness, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	comCall(b.hwndRT, 15, packPointF(float32(x0), float32(y0)), packPointF(float32(x1), float32(y1)), br, uintptr(math.Float32bits(float32(thickness))), 0)
}

func (b *D2DBackend) FillCircle(cx, cy, radius int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.FillCircle(cx, cy, radius, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	e := d2dEllipse{Point: d2dPoint2F{float32(cx), float32(cy)}, RadiusX: float32(radius), RadiusY: float32(radius)}
	comCall(b.hwndRT, 21, uintptr(unsafe.Pointer(&e)), br) // FillEllipse
}

func (b *D2DBackend) StrokeCircle(cx, cy, radius, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.StrokeCircle(cx, cy, radius, thickness, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	e := d2dEllipse{Point: d2dPoint2F{float32(cx), float32(cy)}, RadiusX: float32(radius), RadiusY: float32(radius)}
	comCall(b.hwndRT, 20, uintptr(unsafe.Pointer(&e)), br, uintptr(math.Float32bits(float32(thickness))), 0)
}

func (b *D2DBackend) StrokeEllipse(cx, cy, rx, ry, thickness int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.StrokeEllipse(cx, cy, rx, ry, thickness, c)
	if b.hwndRT == 0 {
		return
	}
	b.beginDrawLocked()
	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)
	e := d2dEllipse{Point: d2dPoint2F{float32(cx), float32(cy)}, RadiusX: float32(rx), RadiusY: float32(ry)}
	comCall(b.hwndRT, 20, uintptr(unsafe.Pointer(&e)), br, uintptr(math.Float32bits(float32(thickness))), 0)
}

func (b *D2DBackend) DrawImage(src image.Image, x, y, dw, dh int, angleDeg float64, flipH bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.DrawImage(src, x, y, dw, dh, angleDeg, flipH)
	if b.hwndRT == 0 || dw < 1 || dh < 1 {
		b.cpuBlit = true
		return
	}
	spr := prepareSprite(src, dw, dh, angleDeg, flipH)
	b.beginDrawLocked()
	bmp := b.createBitmapFromRGBALocked(spr)
	if bmp == 0 {
		b.cpuBlit = true
		return
	}
	defer comRelease(bmp)
	sb := spr.Bounds()
	dest := d2dRectF{float32(x), float32(y), float32(x + sb.Dx()), float32(y + sb.Dy())}
	comCall(b.hwndRT, 26, bmp, uintptr(unsafe.Pointer(&dest)), uintptr(math.Float32bits(1)),
		uintptr(d2d1BitmapInterpolationLinear), 0, 0)
}

func (b *D2DBackend) createBitmapFromRGBALocked(src *image.RGBA) uintptr {
	if b.hwndRT == 0 || src == nil {
		return 0
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w < 1 || h < 1 {
		return 0
	}
	bgra := rgbaToBGRAPremul(src)
	props := d2dBitmapProps{
		PixelFormat: d2dPixelFormat{Format: dxgiFormatB8G8R8A8UNORM, AlphaMode: d2d1AlphaModePremultiplied},
		DpiX:        96,
		DpiY:        96,
	}
	var bmp uintptr
	// CreateBitmap = 4: size (by value), srcData, pitch, props*, bitmap**
	hr, _, _ := comCall(b.hwndRT, 4,
		packSizeU(w, h),
		uintptr(unsafe.Pointer(&bgra[0])),
		uintptr(w*4),
		uintptr(unsafe.Pointer(&props)),
		uintptr(unsafe.Pointer(&bmp)),
	)
	if !hrOK(hr) {
		return 0
	}
	return bmp
}

func rgbaToBGRAPremul(src *image.RGBA) []byte {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.RGBAAt(x, y)
			i := (y*w + x) * 4
			a := float64(c.A) / 255
			out[i+0] = uint8(float64(c.B)*a + 0.5) // B
			out[i+1] = uint8(float64(c.G)*a + 0.5)
			out[i+2] = uint8(float64(c.R)*a + 0.5)
			out[i+3] = c.A
		}
	}
	return out
}

func (b *D2DBackend) DrawText(s string, x, y int, fontSize float64, c color.RGBA, align int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.hwndRT == 0 || b.dwFact == 0 || s == "" {
		// בלי DirectWrite — נשאר על צל ה־CPU
		b.cpu.DrawText(s, x, y, fontSize, c, align)
		b.cpuBlit = true
		return
	}
	b.beginDrawLocked()
	fmt16, err := windows.UTF16PtrFromString("Segoe UI")
	if err != nil {
		b.cpu.DrawText(s, x, y, fontSize, c, align)
		b.cpuBlit = true
		return
	}
	locale, _ := windows.UTF16PtrFromString("he-IL")
	var textFormat uintptr
	hr, _, _ := comCall(b.dwFact, 15,
		uintptr(unsafe.Pointer(fmt16)),
		0,
		uintptr(dwriteFontWeightRegular),
		uintptr(dwriteFontStyleNormal),
		uintptr(dwriteFontStretchNormal),
		uintptr(math.Float32bits(float32(fontSize))),
		uintptr(unsafe.Pointer(locale)),
		uintptr(unsafe.Pointer(&textFormat)),
	)
	if !hrOK(hr) || textFormat == 0 {
		b.cpu.DrawText(s, x, y, fontSize, c, align)
		b.cpuBlit = true
		return
	}
	defer comRelease(textFormat)

	ta := dwriteTextAlignmentLeading
	rd := dwriteReadingDirectionLTR
	switch align {
	case AlignRight:
		ta = dwriteTextAlignmentTrailing
		rd = dwriteReadingDirectionRTL
	case AlignCenter:
		ta = dwriteTextAlignmentCenter
	}
	comCall(textFormat, 3, uintptr(ta))
	comCall(textFormat, 4, uintptr(dwriteParagraphAlignmentNear))
	comCall(textFormat, 5, uintptr(dwriteWordWrappingNoWrap))
	comCall(textFormat, 6, uintptr(rd))

	br := b.solidBrush(c)
	if br == 0 {
		return
	}
	defer comRelease(br)

	layoutW := float32(b.w)
	if layoutW < 64 {
		layoutW = 64
	}
	layoutH := float32(fontSize*2 + 8)
	left := float32(x)
	switch align {
	case AlignRight:
		left = float32(x) - layoutW
	case AlignCenter:
		left = float32(x) - layoutW/2
	}
	// baseline של DirectWrite ≈ y; תיבה מעט מעל/מתחת לנקודה
	top := float32(y) - float32(fontSize)*0.15
	rect := d2dRectF{left, top, left + layoutW, top + layoutH}
	u16, err := windows.UTF16FromString(s)
	if err != nil || len(u16) < 1 {
		return
	}
	nchars := uint32(len(u16) - 1)
	comCall(b.hwndRT, 27,
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(nchars),
		textFormat,
		uintptr(unsafe.Pointer(&rect)),
		br,
		uintptr(d2d1DrawTextOptionsNone),
		0,
	)
}

func (b *D2DBackend) FloodFill(x, y int, c color.RGBA) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cpu.FloodFill(x, y, c)
	b.cpuBlit = true
}

func (b *D2DBackend) Present() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.hwndRT == 0 {
		return fmt.Errorf("אין HwndRenderTarget")
	}
	needBlit := b.cpuBlit || !b.drawing
	if needBlit {
		b.endDrawLocked()
		b.beginDrawLocked()
		img := b.cpu.Buffer()
		if img == nil {
			b.endDrawLocked()
			return fmt.Errorf("בופר ציור ריק")
		}
		bmp := b.createBitmapFromRGBALocked(img)
		if bmp == 0 {
			b.endDrawLocked()
			return fmt.Errorf("CreateBitmap נכשל")
		}
		dest := d2dRectF{0, 0, float32(b.w), float32(b.h)}
		const d2d1BitmapInterpolationNearest = 0
		comCall(b.hwndRT, 26, bmp, uintptr(unsafe.Pointer(&dest)), uintptr(math.Float32bits(1)),
			uintptr(d2d1BitmapInterpolationNearest), 0, 0)
		comRelease(bmp)
		b.cpuBlit = false
	}
	hr, _, _ := comCall(b.hwndRT, 49, 0, 0) // EndDraw
	b.drawing = false
	if !hrOK(hr) {
		b.releaseRTLocked()
		return fmt.Errorf("EndDraw: 0x%08x", uint32(hr))
	}
	return nil
}

func (b *D2DBackend) SnapshotRGBA() (*image.RGBA, error) {
	return b.cpu.SnapshotRGBA()
}

func (b *D2DBackend) ReplacePixels(src *image.RGBA) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.cpu.ReplacePixels(src); err != nil {
		return err
	}
	b.cpuBlit = true
	return nil
}

func (b *D2DBackend) Buffer() *image.RGBA {
	return b.cpu.Buffer()
}

func (b *D2DBackend) Destroy() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.releaseRTLocked()
	if b.dwFact != 0 {
		comRelease(b.dwFact)
		b.dwFact = 0
	}
	if b.factory != 0 {
		comRelease(b.factory)
		b.factory = 0
	}
	if b.cpu != nil {
		b.cpu.Destroy()
	}
	b.ok = false
}
