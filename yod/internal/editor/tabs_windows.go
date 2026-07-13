//go:build windows

package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
)

// OpenFileTab — מסמך פתוח בטאב (עורך נפרד לכל קובץ).
type OpenFileTab struct {
	Key      string // נתיב מנורמל, או untitled:N
	FilePath string // ריק = קובץ חדש שלא נשמר
	Title    string
	Editor   *CodeEdit
	IsDirty  bool
	TabBtn   *DarkBtn
	CloseBtn *DarkBtn
}

// DocTabs — ניהול טאבי מסמכים כמו ב־Visual Studio.
type DocTabs struct {
	Bar         *walk.Composite
	EditorsHost *walk.Composite

	ByKey map[string]*OpenFileTab
	Order []*OpenFileTab
	Active *OpenFileTab

	untitledSeq int

	// WireEditor מחבר TextChanged / zoom / סטטוס לעורך חדש
	WireEditor func(ce *CodeEdit, tab *OpenFileTab)
	// OnActivate אחרי מעבר טאב (סנכרון gutter, כותרת, סייר)
	OnActivate func(tab *OpenFileTab)
	// ConfirmClose — false = ביטול סגירה (dirty)
	ConfirmClose func(tab *OpenFileTab) bool
	// OnClosed אחרי סגירה מוצלחת
	OnClosed func(tab *OpenFileTab)
}

func NewDocTabs(bar, editorsHost *walk.Composite) *DocTabs {
	return &DocTabs{
		Bar:         bar,
		EditorsHost: editorsHost,
		ByKey:       map[string]*OpenFileTab{},
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

func (d *DocTabs) ActiveEditor() *CodeEdit {
	if d == nil || d.Active == nil {
		return nil
	}
	return d.Active.Editor
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
	if d == nil || d.Active == nil {
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
	for _, t := range d.Order {
		if t.IsDirty {
			return true
		}
	}
	return false
}

func (d *DocTabs) DirtyTabs() []*OpenFileTab {
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

// Activate מציג את העורך של הטאב ומסתיר את השאר.
func (d *DocTabs) Activate(tab *OpenFileTab) {
	if d == nil || tab == nil {
		return
	}
	d.Active = tab
	d.layoutEditors()
	if tab.Editor != nil {
		tab.Editor.SetFocus()
	}
	d.refreshBar()
	if d.OnActivate != nil {
		d.OnActivate(tab)
	}
}

// layoutEditors ממלא את המארח בעורך הפעיל בלבד (מיקום ידני — בלי RequestLayout שדורס).
func (d *DocTabs) layoutEditors() {
	if d == nil || d.EditorsHost == nil {
		return
	}
	bounds := d.EditorsHost.ClientBoundsPixels()
	for _, t := range d.Order {
		if t.Editor == nil {
			continue
		}
		if t == d.Active {
			t.Editor.SetVisible(true)
			if bounds.Width > 0 && bounds.Height > 0 {
				_ = t.Editor.SetBoundsPixels(walk.Rectangle{
					X: 0, Y: 0, Width: bounds.Width, Height: bounds.Height,
				})
			}
		} else {
			t.Editor.SetVisible(false)
			_ = t.Editor.SetBoundsPixels(walk.Rectangle{})
		}
	}
}

func (d *DocTabs) refreshBar() {
	if d.Bar == nil {
		return
	}
	// מנקים כפתורים ישנים
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

func (d *DocTabs) createEditor() (*CodeEdit, error) {
	if d.EditorsHost == nil {
		return nil, fmt.Errorf("אין מארח לעורכים")
	}
	ce, err := NewCodeEdit(d.EditorsHost)
	if err != nil {
		return nil, err
	}
	styleEditorPane(ce)
	return ce, nil
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
	ce, err := d.createEditor()
	if err != nil {
		return nil, err
	}
	tab := &OpenFileTab{
		Key:      key,
		FilePath: path,
		Title:    d.tabTitle(path, ""),
		Editor:   ce,
		IsDirty:  false,
	}
	d.ByKey[key] = tab
	d.Order = append(d.Order, tab)
	if d.WireEditor != nil {
		d.WireEditor(ce, tab)
	}
	if err := ce.SetText(string(data)); err != nil {
		return nil, err
	}
	tab.IsDirty = false
	d.Activate(tab)
	return tab, nil
}

// OpenUntitled יוצר טאב קובץ חדש עם תוכן התחלתי.
func (d *DocTabs) OpenUntitled(content, title string) (*OpenFileTab, error) {
	ce, err := d.createEditor()
	if err != nil {
		return nil, err
	}
	key := d.newUntitledKey()
	if title == "" {
		title = "קובץ-חדש.יוד"
	}
	tab := &OpenFileTab{
		Key:      key,
		FilePath: "",
		Title:    title,
		Editor:   ce,
		IsDirty:  false,
	}
	d.ByKey[key] = tab
	d.Order = append(d.Order, tab)
	if d.WireEditor != nil {
		d.WireEditor(ce, tab)
	}
	_ = ce.SetText(content)
	tab.IsDirty = false
	d.Activate(tab)
	return tab, nil
}

// Close סוגר טאב (אחרי ConfirmClose אם הוגדר).
func (d *DocTabs) Close(tab *OpenFileTab) bool {
	if d == nil || tab == nil {
		return true
	}
	if d.ConfirmClose != nil && !d.ConfirmClose(tab) {
		return false
	}
	// הסרה מהמבנים
	delete(d.ByKey, tab.Key)
	for i, t := range d.Order {
		if t == tab {
			d.Order = append(d.Order[:i], d.Order[i+1:]...)
			break
		}
	}
	if tab.Editor != nil {
		tab.Editor.Dispose()
		tab.Editor = nil
	}
	closed := tab
	wasActive := d.Active == tab
	if d.OnClosed != nil {
		d.OnClosed(closed)
	}
	if wasActive {
		d.Active = nil
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
