//go:build windows

package editor

import (
	"yod/MaterialIcons"

	"github.com/lxn/walk"
)

var (
	colBtn      = walk.RGB(50, 52, 56)
	colBtnHover = walk.RGB(64, 68, 74)
	colBtnPress = walk.RGB(40, 42, 46)
	colBtnAct   = walk.RGB(32, 34, 38)
	colBtnPri   = walk.RGB(14, 124, 112) // teal יוד
	colBtnPriH  = walk.RGB(18, 148, 134)
	colBtnPriPr = walk.RGB(10, 98, 88)
	colBtnText  = walk.RGB(230, 230, 230)
)

// סוגי איקון לכפתורים (Material Icons)
const (
	iconNone = iota
	iconRun
	iconVM
	iconCheck
	iconNew
	iconOpen
	iconSave
	iconHighlight
	iconFormat
	iconError
	iconOutput
	iconPack
)

// גובה כפתור בפיקסלים אמיתיים (לא יחידות 96dpi שמתנפחות עם DPI)
const btnH = 22
const btnFontPt = 8
const btnIconPt = 10

// DarkBtn כפתור שטוח מקצועי עם Material Icon + tooltip
type DarkBtn struct {
	fb      *flatBtn
	text    string
	tip     string
	icon    int
	primary bool
	active  bool
	pressed bool
	hover   bool
	onClick func()
	minW    int
}

// flatBtn עוטף CustomWidget ודורס CreateLayoutItem — בלי זה walk נותן IdealSize 100×100.
type flatBtn struct {
	*walk.CustomWidget
	width  int // native pixels
	height int // native pixels
}

type flatBtnLayoutItem struct {
	walk.LayoutItemBase
	width  int
	height int
}

func (li *flatBtnLayoutItem) LayoutFlags() walk.LayoutFlags {
	return 0 // גודל קבוע — לא Grow/Greedy
}

func (li *flatBtnLayoutItem) IdealSize() walk.Size {
	return walk.Size{Width: li.width, Height: li.height}
}

func (li *flatBtnLayoutItem) MinSize() walk.Size {
	return walk.Size{Width: li.width, Height: li.height}
}

func (fb *flatBtn) CreateLayoutItem(ctx *walk.LayoutContext) walk.LayoutItem {
	return &flatBtnLayoutItem{width: fb.width, height: fb.height}
}

func (b *DarkBtn) SetText(text string) {
	b.text = text
	if b.fb != nil {
		b.fb.Invalidate()
	}
}

func (b *DarkBtn) SetActive(v bool) {
	if b.active == v {
		return
	}
	b.active = v
	if b.fb != nil {
		b.fb.Invalidate()
	}
}

// Mount יוצר כפתור בגודל מותאם לטקסט (+5px מימין ומשמאל) בתוך parent.
func (b *DarkBtn) Mount(parent walk.Container, tip string) error {
	b.tip = tip
	materialicons.Ensure()

	dpi := 96
	if parent != nil {
		dpi = parent.DPI()
	}
	minW := b.calcWidth(dpi)
	b.minW = minW

	cw, err := walk.NewCustomWidgetPixels(parent, 0, b.paint)
	if err != nil {
		return err
	}
	cw.SetPaintMode(walk.PaintBuffered)
	cw.SetInvalidatesOnResize(true)
	if tip != "" {
		_ = cw.SetToolTipText(tip)
	}

	fb := &flatBtn{CustomWidget: cw, width: minW, height: btnH}
	if err := walk.InitWrapperWindow(fb); err != nil {
		cw.Dispose()
		return err
	}
	b.fb = fb

	_ = fb.SetMinMaxSizePixels(
		walk.Size{Width: minW, Height: btnH},
		walk.Size{Width: minW, Height: btnH},
	)

	cw.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		b.pressed = true
		fb.Invalidate()
	})
	cw.MouseUp().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		was := b.pressed
		b.pressed = false
		fb.Invalidate()
		bounds := fb.ClientBoundsPixels()
		if was && x >= 0 && y >= 0 && x < bounds.Width && y < bounds.Height && b.onClick != nil {
			b.onClick()
		}
	})
	cw.MouseMove().Attach(func(x, y int, button walk.MouseButton) {
		bounds := fb.ClientBoundsPixels()
		h := x >= 0 && y >= 0 && x < bounds.Width && y < bounds.Height
		if h != b.hover {
			b.hover = h
			fb.Invalidate()
		}
	})
	return nil
}

const btnPadX = 5

func (b *DarkBtn) calcWidth(dpi int) int {
	w := btnPadX * 2
	if b.icon != iconNone {
		w += btnIconPt + 4
	}
	if b.text == "" {
		if w < 24 {
			return 24
		}
		return w
	}
	if dpi < 1 {
		dpi = 96
	}
	bmp, err := walk.NewBitmapForDPI(walk.Size{Width: 8, Height: 8}, dpi)
	if err != nil {
		return w + len([]rune(b.text))*7
	}
	defer bmp.Dispose()
	canvas, err := walk.NewCanvasFromImage(bmp)
	if err != nil {
		return w + len([]rune(b.text))*7
	}
	defer canvas.Dispose()
	font, err := walk.NewFont(uiFont, btnFontPt, 0)
	if err != nil {
		return w + len([]rune(b.text))*7
	}
	defer font.Dispose()
	br, _, err := canvas.MeasureTextPixels(b.text, font,
		walk.Rectangle{Width: 4000, Height: 100},
		walk.TextSingleLine|walk.TextRTLReading)
	if err != nil {
		return w + len([]rune(b.text))*7
	}
	return w + br.Width
}

func (b *DarkBtn) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	if b.fb == nil {
		return nil
	}
	bounds := b.fb.ClientBoundsPixels()
	bg := colBtn
	fg := colBtnText
	switch {
	case b.primary && b.pressed:
		bg = colBtnPriPr
		fg = walk.RGB(255, 255, 255)
	case b.primary && b.hover:
		bg = colBtnPriH
		fg = walk.RGB(255, 255, 255)
	case b.primary:
		bg = colBtnPri
		fg = walk.RGB(255, 255, 255)
	case b.active:
		bg = colBtnAct
		fg = walk.RGB(255, 255, 255)
	case b.pressed:
		bg = colBtnPress
	case b.hover:
		bg = colBtnHover
	}

	brush, err := walk.NewSolidColorBrush(bg)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	if err := canvas.FillRectanglePixels(brush, bounds); err != nil {
		return err
	}

	if b.active && !b.primary {
		line, err := walk.NewSolidColorBrush(colBrand)
		if err != nil {
			return err
		}
		defer line.Dispose()
		underline := walk.Rectangle{
			X: bounds.X + 3, Y: bounds.Y + bounds.Height - 2,
			Width: bounds.Width - 6, Height: 2,
		}
		if err := canvas.FillRectanglePixels(line, underline); err != nil {
			return err
		}
	}

	iconW := 0
	if b.icon != iconNone {
		iconW = btnIconPt + 2
		ir := walk.Rectangle{
			X: bounds.X + bounds.Width - btnIconPt - btnPadX, Y: bounds.Y,
			Width: btnIconPt + 2, Height: bounds.Height,
		}
		if err := b.drawMaterialIcon(canvas, ir, fg); err != nil {
			return err
		}
	}

	font, err := walk.NewFont(uiFont, btnFontPt, 0)
	if err != nil {
		font = b.fb.Font()
	} else {
		defer font.Dispose()
	}
	if font == nil {
		return nil
	}
	tr := walk.Rectangle{
		X:      bounds.X + btnPadX,
		Y:      bounds.Y,
		Width:  bounds.Width - iconW - btnPadX*2,
		Height: bounds.Height,
	}
	return canvas.DrawTextPixels(b.text, font, fg, tr,
		walk.TextCenter|walk.TextVCenter|walk.TextSingleLine|walk.TextRTLReading)
}

func (b *DarkBtn) iconRune() rune {
	switch b.icon {
	case iconRun:
		return materialicons.PlayArrow
	case iconVM:
		return materialicons.Memory
	case iconCheck:
		return materialicons.Build
	case iconNew:
		return materialicons.NoteAdd
	case iconOpen:
		return materialicons.FolderOpen
	case iconSave:
		return materialicons.Save
	case iconHighlight:
		return materialicons.Highlight
	case iconFormat:
		return materialicons.FormatIndent
	case iconError:
		return materialicons.ErrorOutline
	case iconOutput:
		return materialicons.Terminal
	case iconPack:
		return materialicons.Archive
	default:
		return 0
	}
}

func (b *DarkBtn) drawMaterialIcon(canvas *walk.Canvas, r walk.Rectangle, fg walk.Color) error {
	glyph := b.iconRune()
	if glyph == 0 {
		return nil
	}
	materialicons.Ensure()
	font, err := walk.NewFont(materialicons.Family, btnIconPt, 0)
	if err != nil {
		return nil
	}
	defer font.Dispose()
	return canvas.DrawTextPixels(materialicons.Glyph(glyph), font, fg, r,
		walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}
