//go:build windows

package editor

import (
	"yod/MaterialIcons"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
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

const btnH = 20
const btnFontPt = 8
const btnIconPx = 12

// DarkBtn כפתור שטוח מקצועי עם Material Icon + tooltip
type DarkBtn struct {
	cw      *walk.CustomWidget
	text    string
	tip     string
	icon    int
	primary bool
	active  bool
	pressed bool
	hover   bool
	onClick func()
}

func (b *DarkBtn) SetText(text string) {
	b.text = text
	if b.cw != nil {
		b.cw.Invalidate()
	}
}

func (b *DarkBtn) SetActive(v bool) {
	if b.active == v {
		return
	}
	b.active = v
	if b.cw != nil {
		b.cw.Invalidate()
	}
}

func (b *DarkBtn) Decl(minW int, tip string) CustomWidget {
	b.tip = tip
	materialicons.Ensure()
	return CustomWidget{
		AssignTo:            &b.cw,
		MinSize:             Size{Width: minW, Height: btnH},
		MaxSize:             Size{Height: btnH},
		ToolTipText:         tip,
		PaintMode:           PaintBuffered,
		InvalidatesOnResize: true,
		Paint:               b.paint,
		OnMouseDown: func(x, y int, button walk.MouseButton) {
			if button != walk.LeftButton || b.cw == nil {
				return
			}
			b.pressed = true
			b.cw.Invalidate()
		},
		OnMouseUp: func(x, y int, button walk.MouseButton) {
			if button != walk.LeftButton || b.cw == nil {
				return
			}
			was := b.pressed
			b.pressed = false
			b.cw.Invalidate()
			bounds := b.cw.ClientBounds()
			if was && x >= 0 && y >= 0 && x < bounds.Width && y < bounds.Height && b.onClick != nil {
				b.onClick()
			}
		},
		OnMouseMove: func(x, y int, button walk.MouseButton) {
			if b.cw == nil {
				return
			}
			bounds := b.cw.ClientBounds()
			h := x >= 0 && y >= 0 && x < bounds.Width && y < bounds.Height
			if h != b.hover {
				b.hover = h
				b.cw.Invalidate()
			}
		},
	}
}

func (b *DarkBtn) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	if b.cw == nil {
		return nil
	}
	bounds := b.cw.ClientBounds()
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
	if err := canvas.FillRectangle(brush, bounds); err != nil {
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
		if err := canvas.FillRectangle(line, underline); err != nil {
			return err
		}
	}

	iconW := 0
	if b.icon != iconNone {
		iconW = btnIconPx
		ir := walk.Rectangle{
			X: bounds.X + bounds.Width - btnIconPx - 3, Y: bounds.Y,
			Width: btnIconPx, Height: bounds.Height,
		}
		if err := b.drawMaterialIcon(canvas, ir, fg); err != nil {
			return err
		}
	}

	font, err := walk.NewFont(uiFont, btnFontPt, 0)
	if err != nil {
		font = b.cw.Font()
	} else {
		defer font.Dispose()
	}
	if font == nil {
		return nil
	}
	tr := walk.Rectangle{
		X:      bounds.X + 4,
		Y:      bounds.Y,
		Width:  bounds.Width - iconW - 8,
		Height: bounds.Height,
	}
	return canvas.DrawText(b.text, font, fg, tr,
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
	font, err := walk.NewFont(materialicons.Family, btnIconPx, 0)
	if err != nil {
		return nil // בלי פונט — מדלגים על איקון
	}
	defer font.Dispose()
	return canvas.DrawText(materialicons.Glyph(glyph), font, fg, r,
		walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}
