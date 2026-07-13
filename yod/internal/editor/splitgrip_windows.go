//go:build windows

package editor

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// splitGrip — ידית גרירה אופקית. מיקום לפי קואורדינטות מסך/לקוח של המארח
// (לא דלתא עם סימן RTL) — הידית עוקבת אחרי העכבר בלי היפוך ובלי קפיצות.
type splitGrip struct {
	*walk.CustomWidget
	dragging bool
	onDragX  func(clientX int) // X בתוך ה־host (פיקסלי לקוח)
	host     walk.Window
}

func (g *splitGrip) CreateLayoutItem(ctx *walk.LayoutContext) walk.LayoutItem {
	return &gripLayoutItem{width: 6}
}

type gripLayoutItem struct {
	walk.LayoutItemBase
	width int
}

func (li *gripLayoutItem) LayoutFlags() walk.LayoutFlags {
	return walk.GrowableVert | walk.GreedyVert
}

func (li *gripLayoutItem) IdealSize() walk.Size {
	return walk.Size{Width: li.width, Height: 100}
}

func (li *gripLayoutItem) MinSize() walk.Size {
	return walk.Size{Width: li.width, Height: 40}
}

func (g *splitGrip) Mount(parent walk.Container, host walk.Window, onDragX func(clientX int)) error {
	g.onDragX = onDragX
	g.host = host
	cw, err := walk.NewCustomWidgetPixels(parent, 0, g.paint)
	if err != nil {
		return err
	}
	cw.SetPaintMode(walk.PaintBuffered)
	cw.SetCursor(walk.CursorSizeWE())

	g.CustomWidget = cw
	if err := walk.InitWrapperWindow(g); err != nil {
		cw.Dispose()
		g.CustomWidget = nil
		return err
	}
	_ = g.SetMinMaxSizePixels(walk.Size{Width: 6, Height: 40}, walk.Size{Width: 6, Height: 8000})

	cw.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		_ = x
		_ = y
		if button != walk.LeftButton {
			return
		}
		g.dragging = true
		win.SetCapture(cw.Handle())
		g.Invalidate()
		g.emitClientX()
	})
	cw.MouseMove().Attach(func(x, y int, button walk.MouseButton) {
		_ = x
		_ = y
		_ = button
		if !g.dragging {
			return
		}
		g.emitClientX()
	})
	cw.MouseUp().Attach(func(x, y int, button walk.MouseButton) {
		_ = x
		_ = y
		if button != walk.LeftButton {
			return
		}
		g.dragging = false
		win.ReleaseCapture()
		g.Invalidate()
	})
	return nil
}

func (g *splitGrip) emitClientX() {
	if g.onDragX == nil || g.host == nil {
		return
	}
	var pt win.POINT
	win.GetCursorPos(&pt)
	win.ScreenToClient(g.host.Handle(), &pt)
	g.onDragX(int(pt.X))
}

func (g *splitGrip) paint(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
	if g.CustomWidget == nil {
		return nil
	}
	bounds := g.ClientBoundsPixels()
	bg := colBorder
	if g.dragging {
		bg = colBrand
	}
	brush, err := walk.NewSolidColorBrush(bg)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	return canvas.FillRectanglePixels(brush, bounds)
}
