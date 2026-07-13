//go:build windows

package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
)

// OpenFileTab — מסמך פתוח בטאב עם עורך RichEdit משלו (מעבר טאב = הצגה/הסתרה).
type OpenFileTab struct {
	Key          string // נתיב מנורמל, או untitled:N
	FilePath     string // ריק = קובץ חדש שלא נשמר
	Title        string
	Text         string // גיבוי; התוכן החי ב־edit
	SelStart     int
	SelEnd       int
	FirstVisible int
	ScrollX      int
	ScrollY      int
	IsDirty      bool
	TabBtn       *DocTabBtn
	edit         *CodeEdit // עורך ייעודי לטאב — לא נטען מחדש במעבר
}

// DocTabs — ניהול טאבי מסמכים; כל טאב מחזיק CodeEdit נפרד.
type DocTabs struct {
	Bar  *walk.Composite
	Host *walk.Composite // codeHost — אב לכל העורכים

	Editor *CodeEdit // העורך הגלוי כרגע (== Active.edit)

	ByKey  map[string]*OpenFileTab
	Order  []*OpenFileTab
	Active *OpenFileTab

	spare *CodeEdit             // עורך ראשון שנוצר ב־main לפני טאב
	wire  func(ed *CodeEdit)    // עיצוב + אירועים לכל עורך חדש

	untitledSeq int
	swapping    bool

	OnActivate   func(tab *OpenFileTab)
	ConfirmClose func(tab *OpenFileTab) bool
	OnClosed     func(tab *OpenFileTab)
	OnEditor     func(ed *CodeEdit) // אחרי החלפת עורך גלוי (סנכרון codeEdit ב־main)
}

func NewDocTabs(bar, host *walk.Composite, spare *CodeEdit, wire func(*CodeEdit)) *DocTabs {
	return &DocTabs{
		Bar:    bar,
		Host:   host,
		Editor: spare,
		spare:  spare,
		wire:   wire,
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
	if d.Active != nil && d.Active.edit != nil {
		return d.Active.edit
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
	if d.Active.TabBtn != nil {
		d.Active.TabBtn.SetDirty(v)
	} else {
		d.rebuildBar()
	}
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
	if d.Active.TabBtn != nil {
		d.Active.TabBtn.SetTitle(d.Active.Title)
		d.Active.TabBtn.SetDirty(false)
	} else {
		d.rebuildBar()
	}
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

// TabText מחזיר את תוכן הטאב (מהעורך אם קיים, אחרת מה־buffer).
func (d *DocTabs) TabText(tab *OpenFileTab) string {
	if tab == nil {
		return ""
	}
	if tab.edit != nil && !d.swapping {
		return tab.edit.Text()
	}
	return tab.Text
}

func (d *DocTabs) persistActive() {
	if d == nil || d.Active == nil || d.swapping {
		return
	}
	ed := d.Active.edit
	if ed == nil {
		ed = d.Editor
	}
	if ed == nil {
		return
	}
	d.Active.SelStart, d.Active.SelEnd = ed.TextSelection()
	d.Active.FirstVisible = ed.FirstVisibleLine()
	d.Active.ScrollX, d.Active.ScrollY = ed.ScrollPos()
	if d.Active.IsDirty {
		d.Active.Text = ed.Text()
	}
}

// ensureEditor יוצר/מקצה עורך לטאב וטוען טקסט פעם אחת בלבד.
func (d *DocTabs) ensureEditor(tab *OpenFileTab) error {
	if tab == nil {
		return nil
	}
	if tab.edit != nil {
		return nil
	}
	var ed *CodeEdit
	if d.spare != nil {
		ed = d.spare
		d.spare = nil
	} else {
		if d.Host == nil {
			return fmt.Errorf("אין אזור עורך")
		}
		var err error
		ed, err = NewCodeEdit(d.Host)
		if err != nil {
			return err
		}
		if d.wire != nil {
			d.wire(ed)
		}
	}
	d.swapping = true
	ed.SetVisible(false)
	if tab.Text != "" {
		_ = ed.SetText(tab.Text)
	} else {
		_ = ed.SetText("")
	}
	d.swapping = false
	tab.edit = ed
	return nil
}

func (d *DocTabs) showEditor(tab *OpenFileTab) {
	if tab == nil {
		return
	}
	d.swapping = true
	defer func() { d.swapping = false }()

	if prev := d.Active; prev != nil && prev != tab && prev.edit != nil {
		prev.edit.SetVisible(false)
	}
	if err := d.ensureEditor(tab); err != nil {
		return
	}
	tab.edit.SetVisible(true)
	d.Editor = tab.edit
	if d.Host != nil {
		d.Host.RequestLayout()
	}
	_ = tab.edit.SetFocus()
	// אחרי Focus/Layout — משחזרים סקרול (SetTextSelection לבד גולל לסמן)
	tab.edit.RestoreView(tab.SelStart, tab.SelEnd, tab.FirstVisible, tab.ScrollX, tab.ScrollY)
	if d.OnEditor != nil {
		d.OnEditor(tab.edit)
	}
}

// Activate מציג את הטאב (הצגה/הסתרה בלבד אם העורך כבר קיים).
func (d *DocTabs) Activate(tab *OpenFileTab) {
	if d == nil || tab == nil {
		return
	}
	if d.Active == tab {
		d.paintActiveChrome()
		if tab.edit != nil {
			_ = tab.edit.SetFocus()
		}
		if d.OnActivate != nil {
			d.OnActivate(tab)
		}
		return
	}
	prev := d.Active
	d.persistActive()
	d.showEditor(tab)
	d.Active = tab

	if prev != nil && prev.TabBtn != nil {
		prev.TabBtn.SetActive(false)
	}
	if tab.TabBtn != nil {
		tab.TabBtn.SetActive(true)
		tab.TabBtn.SetDirty(tab.IsDirty)
	} else {
		d.rebuildBar()
	}
	if d.OnActivate != nil {
		d.OnActivate(tab)
	}
}

func (d *DocTabs) refreshBar() {
	d.rebuildBar()
}

func (d *DocTabs) paintActiveChrome() {
	for _, t := range d.Order {
		if t.TabBtn == nil {
			d.rebuildBar()
			return
		}
		t.TabBtn.SetActive(t == d.Active)
		t.TabBtn.SetDirty(t.IsDirty)
	}
}

func (d *DocTabs) rebuildBar() {
	if d.Bar == nil {
		return
	}
	for d.Bar.Children().Len() > 0 {
		d.Bar.Children().At(0).Dispose()
	}
	forceTabBarLTR(d.Bar.Handle())
	for _, t := range d.Order {
		tab := t
		tip := tab.FilePath
		if tip == "" {
			tip = tab.Title
		}
		btn := &DocTabBtn{
			title:  tab.Title,
			dirty:  tab.IsDirty,
			active: tab == d.Active,
			onActivate: func() {
				d.Activate(tab)
			},
			onClose: func() {
				_ = d.Close(tab)
			},
		}
		_ = btn.Mount(d.Bar, tip)
		tab.TabBtn = btn
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

func (d *DocTabs) openNewTab(tab *OpenFileTab) (*OpenFileTab, error) {
	d.persistActive()
	d.ByKey[tab.Key] = tab
	d.Order = append(d.Order, tab)
	if err := d.ensureEditor(tab); err != nil {
		delete(d.ByKey, tab.Key)
		d.Order = d.Order[:len(d.Order)-1]
		return nil, err
	}
	prev := d.Active
	if prev != nil && prev.edit != nil {
		prev.edit.SetVisible(false)
	}
	d.Active = tab
	tab.edit.SetVisible(true)
	d.Editor = tab.edit
	_ = tab.edit.SetFocus()
	if d.Host != nil {
		d.Host.RequestLayout()
	}
	if d.OnEditor != nil {
		d.OnEditor(tab.edit)
	}
	d.rebuildBar()
	if d.OnActivate != nil {
		d.OnActivate(tab)
	}
	return tab, nil
}

// OpenPath פותח קובץ בטאב חדש או ממקד טאב קיים.
func (d *DocTabs) OpenPath(path string) (*OpenFileTab, error) {
	key := normalizeTabPath(path)
	if t := d.ByKey[key]; t != nil {
		d.Activate(t)
		return t, nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("לא ניתן לפתוח תיקייה כקובץ")
	}
	if fi.Size() > maxEditorOpenBytes {
		return nil, fmt.Errorf("הקובץ גדול מדי לעורך")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if looksBinary(data) {
		return nil, fmt.Errorf("קובץ זה לא נפתח בעורך יוד (בינארי):\n%s", filepath.Base(path))
	}
	tab := &OpenFileTab{
		Key:      key,
		FilePath: path,
		Title:    d.tabTitle(path, ""),
		Text:     string(data),
		IsDirty:  false,
	}
	return d.openNewTab(tab)
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
	return d.openNewTab(tab)
}

func (d *DocTabs) disposeEdit(tab *OpenFileTab) {
	if tab == nil || tab.edit == nil {
		return
	}
	ed := tab.edit
	tab.edit = nil
	if d.Editor == ed {
		d.Editor = nil
	}
	ed.Dispose()
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
	d.disposeEdit(closed)
	if d.OnClosed != nil {
		d.OnClosed(closed)
	}
	if wasActive {
		if len(d.Order) > 0 {
			next := d.Order[len(d.Order)-1]
			d.showEditor(next)
			d.Active = next
			d.rebuildBar()
			if d.OnActivate != nil {
				d.OnActivate(next)
			}
		} else {
			d.rebuildBar()
			if d.OnActivate != nil {
				d.OnActivate(nil)
			}
		}
	} else {
		d.rebuildBar()
	}
	return true
}

func (d *DocTabs) CloseActive() bool {
	if d == nil || d.Active == nil {
		return true
	}
	return d.Close(d.Active)
}

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
