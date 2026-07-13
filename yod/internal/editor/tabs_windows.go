//go:build windows

package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// OpenFileTab — מסמך פתוח בטאב (buffer בזיכרון; עורך CodeEdit משותף אחד).
type OpenFileTab struct {
	Key          string // נתיב מנורמל, או untitled:N
	FilePath     string // ריק = קובץ חדש שלא נשמר
	Title        string
	Text         string
	SelStart     int
	SelEnd       int
	FirstVisible int
	IsDirty      bool
	TabBtn       *DarkBtn
	CloseBtn     *DarkBtn
}

// DocTabs — ניהול טאבי מסמכים כמו ב־Visual Studio, עם עורך יחיד.
type DocTabs struct {
	Bar    *walk.Composite
	Editor *CodeEdit

	ByKey  map[string]*OpenFileTab
	Order  []*OpenFileTab
	Active *OpenFileTab

	untitledSeq int
	swapping    bool // true בזמן החלפת buffer — לא לסמן dirty

	// OnActivate אחרי מעבר טאב (סנכרון gutter, כותרת, סייר)
	OnActivate func(tab *OpenFileTab)
	// ConfirmClose — false = ביטול סגירה (dirty)
	ConfirmClose func(tab *OpenFileTab) bool
	// OnClosed אחרי סגירה מוצלחת
	OnClosed func(tab *OpenFileTab)
}

func NewDocTabs(bar *walk.Composite, editor *CodeEdit) *DocTabs {
	return &DocTabs{
		Bar:    bar,
		Editor: editor,
		ByKey:  map[string]*OpenFileTab{},
	}
}

func normalizeTabPath(p string) string {
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(abs)
}

func (d *DocTabs) tabTitle(path, fallback string) string {
	if path != "" {
		return filepath.Base(path)
	}
	if fallback != "" {
		return fallback
	}
	return "קובץ-חדש.יוד"
}

func (d *DocTabs) IsSwapping() bool {
	return d != nil && d.swapping
}

func (d *DocTabs) ActiveEditor() *CodeEdit {
	if d == nil {
		return nil
	}
	return d.Editor
}

func (d *DocTabs) ActivePath() string {
	if d == nil || d.Active == nil {
		return ""
	}
	return d.Active.FilePath
}

func (d *DocTabs) ActiveDirty() bool {
	return d != nil && d.Active != nil && d.Active.IsDirty
}

func (d *DocTabs) SetActiveDirty(v bool) {
	if d == nil || d.Active == nil || d.swapping {
		return
	}
	if d.Active.IsDirty == v {
		return
	}
	d.Active.IsDirty = v
	d.refreshBar()
}

func (d *DocTabs) MarkPath(path string) {
	if d == nil || d.Active == nil {
		return
	}
	d.persistActive()
	key := normalizeTabPath(path)
	old := d.Active.Key
	if old != key {
		delete(d.ByKey, old)
		d.Active.Key = key
		d.ByKey[key] = d.Active
	}
	d.Active.FilePath = path
	d.Active.Title = d.tabTitle(path, d.Active.Title)
	d.Active.IsDirty = false
	d.refreshBar()
}

func (d *DocTabs) AnyDirty() bool {
	d.persistActive()
	for _, t := range d.Order {
		if t.IsDirty {
			return true
		}
	}
	return false
}

func (d *DocTabs) DirtyTabs() []*OpenFileTab {
	d.persistActive()
	var out []*OpenFileTab
	for _, t := range d.Order {
		if t.IsDirty {
			out = append(out, t)
		}
	}
	return out
}

func (d *DocTabs) FindByPath(path string) *OpenFileTab {
	if path == "" {
		return nil
	}
	return d.ByKey[normalizeTabPath(path)]
}

// TabText מחזיר את תוכן הטאב (מהעורך אם פעיל, אחרת מה־buffer).
func (d *DocTabs) TabText(tab *OpenFileTab) string {
	if tab == nil {
		return ""
	}
	if d != nil && tab == d.Active && d.Editor != nil && !d.swapping {
		return d.Editor.Text()
	}
	return tab.Text
}

func (d *DocTabs) persistActive() {
	if d == nil || d.Active == nil || d.Editor == nil || d.swapping {
		return
	}
	d.Active.Text = d.Editor.Text()
	d.Active.SelStart, d.Active.SelEnd = d.Editor.TextSelection()
	d.Active.FirstVisible = d.Editor.FirstVisibleLine()
}

func (d *DocTabs) loadIntoEditor(tab *OpenFileTab) {
	if d == nil || d.Editor == nil || tab == nil {
		return
	}
	d.swapping = true
	_ = d.Editor.SetText(tab.Text)
	start, end := tab.SelStart, tab.SelEnd
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	d.Editor.SetTextSelection(start, end)
	cur := d.Editor.FirstVisibleLine()
	delta := tab.FirstVisible - cur
	if delta != 0 {
		d.Editor.SendMessage(win.EM_LINESCROLL, 0, uintptr(delta))
	}
	d.Editor.ScrollCaret()
	d.swapping = false
}

// Activate מציג את תוכן הטאב בעורך המשותף.
func (d *DocTabs) Activate(tab *OpenFileTab) {
	if d == nil || tab == nil {
		return
	}
	if d.Active == tab {
		d.refreshBar()
		if d.Editor != nil {
			d.Editor.SetFocus()
		}
		if d.OnActivate != nil {
			d.OnActivate(tab)
		}
		return
	}
	d.persistActive()
	d.Active = tab
	d.loadIntoEditor(tab)
	if d.Editor != nil {
		d.Editor.SetFocus()
	}
	d.refreshBar()
	if d.OnActivate != nil {
		d.OnActivate(tab)
	}
}

func (d *DocTabs) refreshBar() {
	if d.Bar == nil {
		return
	}
	for d.Bar.Children().Len() > 0 {
		d.Bar.Children().At(0).Dispose()
	}
	for _, t := range d.Order {
		tab := t
		title := tab.Title
		if tab.IsDirty {
			title = "● " + title
		}
		btn := &DarkBtn{
			text:   title,
			active: tab == d.Active,
			onClick: func() {
				d.Activate(tab)
			},
		}
		tip := tab.FilePath
		if tip == "" {
			tip = tab.Title
		}
		_ = btn.Mount(d.Bar, tip)
		tab.TabBtn = btn

		xbtn := &DarkBtn{
			text: "×",
			onClick: func() {
				_ = d.Close(tab)
			},
		}
		_ = xbtn.Mount(d.Bar, "סגור טאב")
		tab.CloseBtn = xbtn
	}
	if _, err := walk.NewHSpacer(d.Bar); err == nil {
		// ok
	}
	d.Bar.RequestLayout()
}

func (d *DocTabs) newUntitledKey() string {
	d.untitledSeq++
	return fmt.Sprintf("untitled:%d", d.untitledSeq)
}

// OpenPath פותח קובץ בטאב חדש או ממקד טאב קיים.
func (d *DocTabs) OpenPath(path string) (*OpenFileTab, error) {
	key := normalizeTabPath(path)
	if t := d.ByKey[key]; t != nil {
		d.Activate(t)
		return t, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tab := &OpenFileTab{
		Key:      key,
		FilePath: path,
		Title:    d.tabTitle(path, ""),
		Text:     string(data),
		IsDirty:  false,
	}
	d.ByKey[key] = tab
	d.Order = append(d.Order, tab)
	d.Activate(tab)
	return tab, nil
}

// OpenUntitled יוצר טאב קובץ חדש עם תוכן התחלתי.
func (d *DocTabs) OpenUntitled(content, title string) (*OpenFileTab, error) {
	key := d.newUntitledKey()
	if title == "" {
		title = "קובץ-חדש.יוד"
	}
	tab := &OpenFileTab{
		Key:      key,
		FilePath: "",
		Title:    title,
		Text:     content,
		IsDirty:  false,
	}
	d.ByKey[key] = tab
	d.Order = append(d.Order, tab)
	d.Activate(tab)
	return tab, nil
}

// Close סוגר טאב (אחרי ConfirmClose אם הוגדר).
func (d *DocTabs) Close(tab *OpenFileTab) bool {
	if d == nil || tab == nil {
		return true
	}
	if tab == d.Active {
		d.persistActive()
	}
	if d.ConfirmClose != nil && !d.ConfirmClose(tab) {
		return false
	}
	delete(d.ByKey, tab.Key)
	for i, t := range d.Order {
		if t == tab {
			d.Order = append(d.Order[:i], d.Order[i+1:]...)
			break
		}
	}
	closed := tab
	wasActive := d.Active == tab
	if wasActive {
		d.Active = nil
	}
	if d.OnClosed != nil {
		d.OnClosed(closed)
	}
	if wasActive {
		if len(d.Order) > 0 {
			d.Activate(d.Order[len(d.Order)-1])
		} else {
			d.refreshBar()
			if d.OnActivate != nil {
				d.OnActivate(nil)
			}
		}
	} else {
		d.refreshBar()
	}
	return true
}

// CloseActive סוגר את הטאב הפעיל.
func (d *DocTabs) CloseActive() bool {
	if d == nil || d.Active == nil {
		return true
	}
	return d.Close(d.Active)
}

// EnsureWelcome פותח טאב קבלת פנים אם אין טאבים.
func (d *DocTabs) EnsureWelcome(content string) error {
	if len(d.Order) > 0 {
		return nil
	}
	_, err := d.OpenUntitled(content, "קובץ-חדש.יוד")
	return err
}

func pathEqual(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	return strings.EqualFold(normalizeTabPath(a), normalizeTabPath(b))
}
