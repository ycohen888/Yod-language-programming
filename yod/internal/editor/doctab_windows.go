//go:build windows

package editor

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// צבעי טאבי קבצים — בסגנון VS Code Dark+
var (
	colDocTabBar     = walk.RGB(37, 37, 38)  // #252526
	colDocTabIdle    = walk.RGB(45, 45, 45)  // #2d2d2d
	colDocTabHover   = walk.RGB(50, 50, 52)
	colDocTabActive  = walk.RGB(30, 30, 30)  // #1e1e1e כמו העורך
	colDocTabAccent  = walk.RGB(0, 122, 204) // #007acc כחול VS
	colDocTabText    = walk.RGB(255, 255, 255)
	colDocTabMuted   = walk.RGB(150, 150, 150)
	colDocTabCloseHv = walk.RGB(232, 68, 68)
)

const (
	docTabH      = 35
	docTabFontPt = 9
	docTabPadX   = 12
	docTabCloseW = 22
	docTabGap    = 0
)

// DocTabBtn — טאב קובץ בסגנון Visual Studio Code (כותרת + dirty + ×).
type DocTabBtn struct {
	fb         *flatBtn
	title      string
	tip        string
	dirty      bool
	active     bool
	pressed    bool
	hover      bool
	closeHover bool
	onActivate func()
	onClose    func()
	minW       int
}

func (b *DocTabBtn) SetTitle(title string) {
	b.title = title
	if b.fb != nil {
		b.fb.Invalidate()
	}
}

func (b *DocTabBtn) SetDirty(v bool) {
	if b.dirty == v {
		return
	}
	b.dirty = v
	if b.fb != nil {
		b.fb.Invalidate()
	}
}

func (b *DocTabBtn) SetActive(v bool) {
	if b.active == v {
		return
	}
	b.active = v
	if b.fb != nil {
		b.fb.Invalidate()
	}
}

func (b *DocTabBtn) Mount(parent walk.Container, tip string) error {
	b.tip = tip
	hwnd := win.HWND(0)
	if parent != nil {
		hwnd = parent.Handle()
	}
	minW := b.calcWidth(hwnd)
	if minW < 90 {
		minW = 90
	}
	if minW > 220 {
		minW = 220
	}
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

	fb := &flatBtn{CustomWidget: cw, width: minW, height: docTabH}
	if err := walk.InitWrapperWindow(fb); err != nil {
		cw.Dispose()
		return err
	}
	b.fb = fb
	_ = fb.SetMinMaxSizePixels(
		walk.Size{Width: minW, Height: docTabH},
		walk.Size{Width: minW, Height: docTabH},
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
		if !was || x < 0 || y < 0 || x >= bounds.Width || y >= bounds.Height {
			return
		}
		if b.inCloseZone(x, bounds.Width) {
			if b.onClose != nil {
				b.onClose()
			}
			return
		}
		if b.onActivate != nil {
			b.onActivate()
		}
	})
	cw.MouseMove().Attach(func(x, y int, button walk.MouseButton) {
		bounds := fb.ClientBoundsPixels()
		h := x >= 0 && y >= 0 && x < bounds.Width && y < bounds.Height
		ch := h && b.inCloseZone(x, bounds.Width)
		if h != b.hover || ch != b.closeHover {
			b.hover = h
			b.closeHover = ch
			fb.Invalidate()
		}
	})
	return nil
}

func (b *DocTabBtn) inCloseZone(x, width int) bool {
	// LTR בתוך הטאב: × בימין
	return x >= width-docTabCloseW-4
}

func (b *DocTabBtn) calcWidth(hwnd win.HWND) int {
	tw := measureTextPx(hwnd, b.title, docTabFontPt)
	if tw <= 0 {
		tw = len([]rune(b.title)) * 7
	}
	return docTabPadX*2 + tw + docTabCloseW + 4
}

func (b *DocTabBtn) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	if b.fb == nil {
		return nil
	}
	bounds := b.fb.ClientBoundsPixels()

	bg := colDocTabIdle
	fg := colDocTabMuted
	switch {
	case b.active:
		bg = colDocTabActive
		fg = colDocTabText
	case b.hover:
		bg = colDocTabHover
		fg = colDocTabText
	}

	brush, err := walk.NewSolidColorBrush(bg)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	if err := canvas.FillRectanglePixels(brush, bounds); err != nil {
		return err
	}

	// קו עליון כחול לטאב פעיל — כמו VS Code
	if b.active {
		acc, err := walk.NewSolidColorBrush(colDocTabAccent)
		if err != nil {
			return err
		}
		defer acc.Dispose()
		top := walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: 2}
		if err := canvas.FillRectanglePixels(acc, top); err != nil {
			return err
		}
	}

	// מפריד דק משמאל (בין טאבים)
	sep, err := walk.NewSolidColorBrush(colBorder)
	if err != nil {
		return err
	}
	defer sep.Dispose()
	_ = canvas.FillRectanglePixels(sep, walk.Rectangle{
		X: bounds.X, Y: bounds.Y + 6, Width: 1, Height: bounds.Height - 12,
	})

	font, err := walk.NewFont(uiFont, docTabFontPt, 0)
	if err != nil {
		font = b.fb.Font()
	} else {
		defer font.Dispose()
	}
	if font == nil {
		return nil
	}

	titleR := walk.Rectangle{
		X:      bounds.X + docTabPadX,
		Y:      bounds.Y,
		Width:  bounds.Width - docTabPadX - docTabCloseW - 6,
		Height: bounds.Height,
	}
	if err := canvas.DrawTextPixels(b.title, font, fg, titleR,
		walk.TextLeft|walk.TextVCenter|walk.TextSingleLine|walk.TextEndEllipsis); err != nil {
		return err
	}

	return b.paintClose(canvas, bounds, fg)
}

func (b *DocTabBtn) paintClose(canvas *walk.Canvas, bounds walk.Rectangle, fg walk.Color) error {
	showChrome := b.active || b.hover || b.dirty
	if !showChrome {
		return nil
	}
	cx := bounds.X + bounds.Width - docTabCloseW/2 - 6
	cy := bounds.Y + bounds.Height/2

	// VS Code: dirty בלי hover על × → נקודה; אחרת ×
	if b.dirty && !b.closeHover {
		dot, err := walk.NewSolidColorBrush(colDocTabText)
		if err != nil {
			return err
		}
		defer dot.Dispose()
		return canvas.FillEllipsePixels(dot, walk.Rectangle{X: cx - 3, Y: cy - 3, Width: 6, Height: 6})
	}

	closeFg := fg
	if b.closeHover {
		closeFg = walk.RGB(255, 255, 255)
		bg, err := walk.NewSolidColorBrush(walk.RGB(60, 60, 62))
		if err == nil {
			defer bg.Dispose()
			_ = canvas.FillEllipsePixels(bg, walk.Rectangle{X: cx - 9, Y: cy - 9, Width: 18, Height: 18})
		}
	}

	font, err := walk.NewFont(uiFont, 11, 0)
	if err != nil {
		return nil
	}
	defer font.Dispose()
	tr := walk.Rectangle{X: cx - 8, Y: cy - 9, Width: 16, Height: 18}
	return canvas.DrawTextPixels("×", font, closeFg, tr,
		walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}

// forceTabBarLTR — טאבי VS Code זורמים משמאל לימין גם בחלון RTL.
func forceTabBarLTR(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	ex := uint32(win.GetWindowLong(hwnd, win.GWL_EXSTYLE))
	ex |= win.WS_EX_NOINHERITLAYOUT
	ex &^= win.WS_EX_LAYOUTRTL
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(ex))
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
}