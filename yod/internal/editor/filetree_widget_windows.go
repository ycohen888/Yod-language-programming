//go:build windows

package editor

import (
	"time"

	materialicons "yod/MaterialIcons"

	"github.com/lxn/walk"
)

// =============================================================================
// Custom File Tree — שלב 3: ציור + בחירה + קלט (בלי walk.TreeView)
// =============================================================================

const (
	treeRowH     = 22
	treeIndentPx = 16 // Depth * treeIndentPx
	treePadX     = 4
	treeGlyphW   = 12
	treeIconW    = 16
	treeFontPt   = 9
	treeIconPt   = 12
	treeDblMS    = 400
)

var (
	colTreeSel   = walk.RGB(14, 70, 110) // VS Code–like selection
	colTreeHover = walk.RGB(42, 45, 50)
	colTreeGlyph = walk.RGB(160, 165, 170)
	colTreeFile  = walk.RGB(200, 205, 210)
	colTreeDir   = walk.RGB(220, 220, 220)
)

// FileTreeView — רשימת עץ שטוחה מצוירת ב־CustomWidget.
type FileTreeView struct {
	*walk.CustomWidget
	model      *FileTreeModel
	selected   int
	hover      int
	scroll     int // אינדקס השורה הראשונה הגלויה
	onActivate func(n treeNode)
	lastClick  time.Time
	lastIdx    int
}

func (t *FileTreeView) CreateLayoutItem(ctx *walk.LayoutContext) walk.LayoutItem {
	return walk.NewGreedyLayoutItem()
}

// NewFileTreeView יוצר סייר עץ ריק (יש לקרוא Mount אחרי שיש parent).
func NewFileTreeView(model *FileTreeModel) *FileTreeView {
	if model == nil {
		model = NewFileTreeModel()
	}
	return &FileTreeView{
		model:    model,
		selected: -1,
		hover:    -1,
	}
}

func (t *FileTreeView) Model() *FileTreeModel { return t.model }

func (t *FileTreeView) SelectedIndex() int { return t.selected }

func (t *FileTreeView) SelectedNode() *treeNode {
	if t.model == nil {
		return nil
	}
	return t.model.NodeAt(t.selected)
}

// OnActivate נקרא בלחיצה כפולה / Enter על קובץ או תיקייה.
func (t *FileTreeView) OnActivate(fn func(n treeNode)) { t.onActivate = fn }

// Mount יוצר את ה־CustomWidget בתוך parent.
func (t *FileTreeView) Mount(parent walk.Container) error {
	materialicons.Ensure()
	cw, err := walk.NewCustomWidgetPixels(parent, 0, t.paint)
	if err != nil {
		return err
	}
	cw.SetPaintMode(walk.PaintBuffered)
	cw.SetInvalidatesOnResize(true)

	t.CustomWidget = cw
	if err := walk.InitWrapperWindow(t); err != nil {
		cw.Dispose()
		t.CustomWidget = nil
		return err
	}

	_ = t.SetMinMaxSizePixels(
		walk.Size{Width: 80, Height: 80},
		walk.Size{Width: 4000, Height: 8000},
	)

	cw.MouseDown().Attach(t.onMouseDown)
	cw.MouseMove().Attach(t.onMouseMove)
	cw.MouseWheel().Attach(t.onMouseWheel)
	cw.KeyDown().Attach(t.onKeyDown)
	return nil
}

// InvalidateTree מרענן ציור אחרי שינוי במודל.
func (t *FileTreeView) InvalidateTree() {
	t.clampScroll()
	if t.selected >= 0 && t.model != nil && t.selected >= t.model.VisibleCount() {
		t.selected = t.model.VisibleCount() - 1
	}
	if t.CustomWidget != nil {
		t.Invalidate()
	}
}

// SetSelectedIndex קובע שורה נבחרת ומבטיח שהיא גלויה.
func (t *FileTreeView) SetSelectedIndex(idx int) {
	if t.model == nil {
		t.selected = -1
		t.InvalidateTree()
		return
	}
	n := t.model.VisibleCount()
	if idx < 0 || idx >= n {
		t.selected = -1
	} else {
		t.selected = idx
		t.ensureVisible(idx)
	}
	t.InvalidateTree()
}

// SelectPath פותח אבות, מרענן, ומסמן את הנתיב.
func (t *FileTreeView) SelectPath(absPath string) {
	if t.model == nil || absPath == "" {
		return
	}
	t.model.ExpandToPath(absPath)
	idx := t.model.IndexOfPath(absPath)
	t.SetSelectedIndex(idx)
}

// RefreshFromDisk מרענן את המודל ואת הציור.
func (t *FileTreeView) RefreshFromDisk() {
	if t.model != nil {
		t.model.Refresh()
	}
	t.InvalidateTree()
}

func (t *FileTreeView) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	if t.CustomWidget == nil {
		return nil
	}
	bounds := t.ClientBoundsPixels()
	bg, err := walk.NewSolidColorBrush(colPanel)
	if err != nil {
		return err
	}
	defer bg.Dispose()
	if err := canvas.FillRectanglePixels(bg, bounds); err != nil {
		return err
	}
	if t.model == nil {
		return nil
	}

	font, err := walk.NewFont(uiFont, treeFontPt, 0)
	if err != nil {
		font = t.Font()
	} else {
		defer font.Dispose()
	}
	if font == nil {
		return nil
	}

	page := bounds.Height / treeRowH
	if page < 1 {
		page = 1
	}
	vis := t.model.Visible()
	end := t.scroll + page + 1
	if end > len(vis) {
		end = len(vis)
	}

	selBrush, err := walk.NewSolidColorBrush(colTreeSel)
	if err != nil {
		return err
	}
	defer selBrush.Dispose()
	hovBrush, err := walk.NewSolidColorBrush(colTreeHover)
	if err != nil {
		return err
	}
	defer hovBrush.Dispose()

	for i := t.scroll; i < end; i++ {
		n := vis[i]
		y := (i - t.scroll) * treeRowH
		row := walk.Rectangle{X: 0, Y: y, Width: bounds.Width, Height: treeRowH}
		switch {
		case i == t.selected:
			if err := canvas.FillRectanglePixels(selBrush, row); err != nil {
				return err
			}
		case i == t.hover:
			if err := canvas.FillRectanglePixels(hovBrush, row); err != nil {
				return err
			}
		}

		indent := treePadX + n.Depth*treeIndentPx
		kind := classifyFileKind(n.Name, n.IsDir)
		iconRune, iconColor := fileKindIcon(kind, n.Expanded)

		chev := " "
		if n.IsDir {
			if n.Expanded {
				chev = "▾"
			} else {
				chev = "▸"
			}
		}
		gr := walk.Rectangle{
			X: indent, Y: y, Width: treeGlyphW, Height: treeRowH,
		}
		if err := canvas.DrawTextPixels(chev, font, colTreeGlyph, gr,
			walk.TextCenter|walk.TextVCenter|walk.TextSingleLine); err != nil {
			return err
		}

		ir := walk.Rectangle{
			X: indent + treeGlyphW, Y: y, Width: treeIconW, Height: treeRowH,
		}
		if err := drawTreeMaterialIcon(canvas, ir, iconRune, iconColor); err != nil {
			return err
		}

		fgName := colTreeFile
		if n.IsDir {
			fgName = colTreeDir
		} else if kind == fileKindYod {
			fgName = colBrand
		}
		nr := walk.Rectangle{
			X:      indent + treeGlyphW + treeIconW + 2,
			Y:      y,
			Width:  bounds.Width - (indent + treeGlyphW + treeIconW + 8),
			Height: treeRowH,
		}
		if nr.Width < 8 {
			continue
		}
		if err := canvas.DrawTextPixels(n.Name, font, fgName, nr,
			walk.TextLeft|walk.TextVCenter|walk.TextSingleLine|walk.TextEndEllipsis|walk.TextRTLReading); err != nil {
			return err
		}
	}
	return nil
}

func drawTreeMaterialIcon(canvas *walk.Canvas, r walk.Rectangle, glyph rune, fg walk.Color) error {
	if glyph == 0 {
		return nil
	}
	font, err := walk.NewFont(materialicons.Family, treeIconPt, 0)
	if err != nil {
		return nil
	}
	defer font.Dispose()
	return canvas.DrawTextPixels(materialicons.Glyph(glyph), font, fg, r,
		walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}

func (t *FileTreeView) rowAt(y int) int {
	if t.model == nil || treeRowH <= 0 {
		return -1
	}
	idx := t.scroll + y/treeRowH
	if idx < 0 || idx >= t.model.VisibleCount() {
		return -1
	}
	return idx
}

func (t *FileTreeView) onMouseDown(x, y int, button walk.MouseButton) {
	_ = t.SetFocus()
	idx := t.rowAt(y)
	if idx < 0 {
		return
	}
	if button == walk.RightButton {
		t.SetSelectedIndex(idx)
		return
	}
	if button != walk.LeftButton {
		return
	}

	dbl := time.Since(t.lastClick) < treeDblMS*time.Millisecond && t.lastIdx == idx
	t.lastClick = time.Now()
	t.lastIdx = idx

	t.SetSelectedIndex(idx)
	n := t.model.NodeAt(idx)
	if n == nil {
		return
	}

	// לחיצה על אזור החץ/איקון תיקייה — פתיחה/סגירה
	indent := treePadX + n.Depth*treeIndentPx
	onGlyph := x >= indent && x < indent+treeGlyphW+treeIconW+2
	if n.IsDir && (onGlyph || dbl) {
		t.model.ToggleExpanded(n.Path)
		// אחרי Rebuild האינדקס של אותה תיקייה עשוי להישאר
		if ni := t.model.IndexOfPath(n.Path); ni >= 0 {
			t.selected = ni
		}
		t.InvalidateTree()
		return
	}
	if dbl && t.onActivate != nil {
		t.onActivate(*n)
	}
}

func (t *FileTreeView) onMouseMove(x, y int, button walk.MouseButton) {
	_ = x
	_ = button
	idx := t.rowAt(y)
	if idx != t.hover {
		t.hover = idx
		if t.CustomWidget != nil {
			t.Invalidate()
		}
	}
}

func (t *FileTreeView) onMouseWheel(x, y int, button walk.MouseButton) {
	_ = x
	_ = y
	delta := walk.MouseWheelEventDelta(button)
	steps := delta / 120
	if steps == 0 {
		if delta > 0 {
			steps = 1
		} else if delta < 0 {
			steps = -1
		}
	}
	// Windows: positive delta = scroll up → smaller first index
	t.scroll -= steps
	t.clampScroll()
	t.Invalidate()
}

func (t *FileTreeView) onKeyDown(key walk.Key) {
	if t.model == nil {
		return
	}
	n := t.model.VisibleCount()
	if n == 0 {
		return
	}
	switch key {
	case walk.KeyUp:
		if t.selected <= 0 {
			t.SetSelectedIndex(0)
		} else {
			t.SetSelectedIndex(t.selected - 1)
		}
	case walk.KeyDown:
		if t.selected < 0 {
			t.SetSelectedIndex(0)
		} else if t.selected < n-1 {
			t.SetSelectedIndex(t.selected + 1)
		}
	case walk.KeyPrior:
		page := t.pageRows()
		t.SetSelectedIndex(max0(t.selected - page))
	case walk.KeyNext:
		page := t.pageRows()
		idx := t.selected + page
		if idx >= n {
			idx = n - 1
		}
		t.SetSelectedIndex(idx)
	case walk.KeyHome:
		t.SetSelectedIndex(0)
	case walk.KeyEnd:
		t.SetSelectedIndex(n - 1)
	case walk.KeyLeft:
		node := t.model.NodeAt(t.selected)
		if node != nil && node.IsDir && node.Expanded {
			t.model.ToggleExpanded(node.Path)
			if ni := t.model.IndexOfPath(node.Path); ni >= 0 {
				t.selected = ni
			}
			t.InvalidateTree()
		}
	case walk.KeyRight:
		node := t.model.NodeAt(t.selected)
		if node != nil && node.IsDir && !node.Expanded {
			t.model.ToggleExpanded(node.Path)
			if ni := t.model.IndexOfPath(node.Path); ni >= 0 {
				t.selected = ni
			}
			t.InvalidateTree()
		}
	case walk.KeyReturn:
		node := t.model.NodeAt(t.selected)
		if node == nil {
			return
		}
		if node.IsDir {
			t.model.ToggleExpanded(node.Path)
			if ni := t.model.IndexOfPath(node.Path); ni >= 0 {
				t.selected = ni
			}
			t.InvalidateTree()
			return
		}
		if t.onActivate != nil {
			t.onActivate(*node)
		}
	}
}

func (t *FileTreeView) pageRows() int {
	if t.CustomWidget == nil {
		return 10
	}
	h := t.ClientBoundsPixels().Height
	p := h / treeRowH
	if p < 1 {
		return 1
	}
	return p
}

func (t *FileTreeView) clampScroll() {
	if t.model == nil {
		t.scroll = 0
		return
	}
	maxScroll := t.model.VisibleCount() - t.pageRows()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
	if t.scroll > maxScroll {
		t.scroll = maxScroll
	}
}

func (t *FileTreeView) ensureVisible(idx int) {
	if idx < 0 {
		return
	}
	page := t.pageRows()
	if idx < t.scroll {
		t.scroll = idx
	} else if idx >= t.scroll+page {
		t.scroll = idx - page + 1
	}
	t.clampScroll()
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func addTreeMenuAction(menu *walk.Menu, text string, fn func()) {
	if menu == nil || fn == nil {
		return
	}
	a := walk.NewAction()
	_ = a.SetText(text)
	a.Triggered().Attach(fn)
	menu.Actions().Add(a)
}
