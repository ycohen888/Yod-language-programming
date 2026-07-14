//go:build windows

package editor

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// splitGrip — ידית גרירה אופקית לפי קואורדינטות מסך (עמיד ל־RTL של החלון).
// מדווח את הזזת המסך המצטברת מתחילת הגרירה (לא דלתא מצטברת) — מונע קפיצות.
type splitGrip struct {
	*walk.CustomWidget
	dragging bool
	startSX  int
	onDrag   func(totalDX int) // totalDX>0 = העכבר זז ימינה על המסך מאז MouseDown
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

func (g *splitGrip) Mount(parent walk.Container, onDrag func(totalDX int)) error {
	g.onDrag = onDrag
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
		var pt win.POINT
		win.GetCursorPos(&pt)
		g.startSX = int(pt.X)
		win.SetCapture(cw.Handle())
		g.Invalidate()
	})
	cw.MouseMove().Attach(func(x, y int, button walk.MouseButton) {
		_ = x
		_ = y
		_ = button
		if !g.dragging || g.onDrag == nil {
			return
		}
		var pt win.POINT
		win.GetCursorPos(&pt)
		g.onDrag(int(pt.X) - g.startSX)
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

// treeLeftOfGrip — האם הסייר משמאל לידית במסך (לא תלוי ב־RTL של Walk).
func treeLeftOfGrip(tree, grip walk.Window) bool {
	if tree == nil || grip == nil {
		return true
	}
	var tr, gr win.RECT
	if !win.GetWindowRect(tree.Handle(), &tr) || !win.GetWindowRect(grip.Handle(), &gr) {
		return true
	}
	return tr.Left < gr.Left
}
