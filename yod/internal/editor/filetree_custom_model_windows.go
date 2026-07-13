//go:build windows

package editor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// =============================================================================
// Custom File Tree — מודל נתונים (רשימה שטוחה + expanded)
//
// לא משתמשים ב־walk.TreeView. מצב מפורש (אילו תיקיות פתוחות) + slice שטוח
// של הצמתים הגלויים. ציור וקלט: FileTreeView ב־filetree_widget_windows.go.
// =============================================================================

// treeNode — צומת בעץ הקבצים (שורה אחת ברשימה השטוחה).
type treeNode struct {
	Path     string // נתיב מוחלט, מנורמל
	Name     string // שם תצוגה (בלי נתיב מלא)
	IsDir    bool
	Depth    int  // 0 = שורש הפרויקט / ילדי השורש
	Expanded bool // רלוונטי רק לתיקיות; קבצים תמיד false
}

// FileTreeModel — מצב הסייר המתקפל (עץ שטוח).
// לא תלוי ב־walk; הממשק הוויזואלי יבוא בהמשך.
type FileTreeModel struct {
	root     string          // שורש הפרויקט (מוחלט)
	expanded map[string]bool // תיקיות פתוחות לפי Path מנורמל
	visible  []treeNode      // הצמתים שמוצגים כרגע
}

// NewFileTreeModel יוצר מודל ריק.
func NewFileTreeModel() *FileTreeModel {
	return &FileTreeModel{
		expanded: map[string]bool{},
	}
}

func (m *FileTreeModel) RootPath() string { return m.root }

// Visible מחזיר את הרשימה השטוחה לתצוגה (עותק לא נדרש — אל תשנו מבחוץ).
func (m *FileTreeModel) Visible() []treeNode { return m.visible }

// VisibleCount מספר שורות גלויות.
func (m *FileTreeModel) VisibleCount() int { return len(m.visible) }

// NodeAt מחזיר צומת גלוי לפי אינדקס, או nil.
func (m *FileTreeModel) NodeAt(index int) *treeNode {
	if index < 0 || index >= len(m.visible) {
		return nil
	}
	n := m.visible[index]
	return &n
}

// SetRoot מגדיר שורש פרויקט. השורש עצמו נחשב פתוח כברירת מחדל.
func (m *FileTreeModel) SetRoot(absPath string) {
	absPath = strings.TrimSpace(absPath)
	if absPath == "" {
		m.root = ""
		m.expanded = map[string]bool{}
		m.visible = nil
		return
	}
	abs, err := filepath.Abs(absPath)
	if err != nil {
		abs = absPath
	}
	abs = filepath.Clean(abs)
	m.root = abs
	m.expanded = map[string]bool{abs: true}
	m.RebuildVisible()
}

// Clear מנקה את המודל (סגירת תיקיית פרויקט).
func (m *FileTreeModel) Clear() {
	m.SetRoot("")
}

// IsExpanded האם תיקייה פתוחה במודל.
func (m *FileTreeModel) IsExpanded(absPath string) bool {
	if m.expanded == nil {
		return false
	}
	return m.expanded[filepath.Clean(absPath)]
}

// SetExpanded קובע מצב פתיחה לתיקייה (לא מרענן דיסק לבד — קראו RebuildVisible).
func (m *FileTreeModel) SetExpanded(absPath string, open bool) {
	absPath = filepath.Clean(absPath)
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}
	if open {
		m.expanded[absPath] = true
	} else {
		delete(m.expanded, absPath)
		// סוגר גם צאצאים שנפתחו — מצב עקבי
		prefix := absPath + string(os.PathSeparator)
		for p := range m.expanded {
			if strings.HasPrefix(p, prefix) {
				delete(m.expanded, p)
			}
		}
	}
}

// ToggleExpanded הופך פתיחה/סגירה ובונה מחדש את הרשימה הגלויה.
func (m *FileTreeModel) ToggleExpanded(absPath string) {
	absPath = filepath.Clean(absPath)
	m.SetExpanded(absPath, !m.IsExpanded(absPath))
	m.RebuildVisible()
}

// Refresh מרענן מהדיסק תוך שמירת אילו תיקיות פתוחות (אם עדיין קיימות).
func (m *FileTreeModel) Refresh() {
	if m.root == "" {
		m.visible = nil
		return
	}
	m.RebuildVisible()
}

// ExpandToPath פותח את כל האבות של נתיב (כדי להציג קובץ פתוח בעורך).
func (m *FileTreeModel) ExpandToPath(absPath string) {
	if m.root == "" || absPath == "" {
		return
	}
	abs, err := filepath.Abs(absPath)
	if err != nil {
		abs = absPath
	}
	abs = filepath.Clean(abs)
	root := filepath.Clean(m.root)
	if abs != root && !strings.HasPrefix(abs+string(os.PathSeparator), root+string(os.PathSeparator)) {
		return
	}
	dir := abs
	fi, err := os.Stat(abs)
	if err == nil && !fi.IsDir() {
		dir = filepath.Dir(abs)
	}
	for {
		dir = filepath.Clean(dir)
		m.SetExpanded(dir, true)
		if dir == root || len(dir) <= len(root) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	m.RebuildVisible()
}

// IndexOfPath מחזיר אינדקס ברשימה הגלויה, או -1.
func (m *FileTreeModel) IndexOfPath(absPath string) int {
	want, err := filepath.Abs(absPath)
	if err != nil {
		want = absPath
	}
	want = filepath.Clean(want)
	for i := range m.visible {
		if filepath.Clean(m.visible[i].Path) == want {
			return i
		}
	}
	return -1
}

// RebuildVisible סורק את הדיסק ובונה slice שטוח של צמתים גלויים בלבד.
// כלל: ילדי תיקייה מופיעים רק אם התיקייה ב־expanded.
func (m *FileTreeModel) RebuildVisible() {
	m.visible = nil
	if m.root == "" {
		return
	}
	root := filepath.Clean(m.root)
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		return
	}
	// שורש לא מוצג כשורה נפרדת — רק תוכנו בעומק 0 (כמו VS Code sidebar).
	if !m.IsExpanded(root) {
		m.SetExpanded(root, true)
	}
	m.appendChildren(root, 0)
}

// appendChildren מוסיף ל־visible את תוכן dir (ממוין: תיקיות ואז קבצים),
// ואם תיקיית־בן פתוחה — רקורסיבית את ילדיה.
func (m *FileTreeModel) appendChildren(dir string, depth int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var dirs, files []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if shouldHideName(name) {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name()) < strings.ToLower(dirs[j].Name())
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name())
	})

	for _, e := range dirs {
		childPath := filepath.Clean(filepath.Join(dir, e.Name()))
		open := m.IsExpanded(childPath)
		m.visible = append(m.visible, treeNode{
			Path:     childPath,
			Name:     e.Name(),
			IsDir:    true,
			Depth:    depth,
			Expanded: open,
		})
		if open {
			m.appendChildren(childPath, depth+1)
		}
	}
	for _, e := range files {
		childPath := filepath.Clean(filepath.Join(dir, e.Name()))
		m.visible = append(m.visible, treeNode{
			Path:     childPath,
			Name:     e.Name(),
			IsDir:    false,
			Depth:    depth,
			Expanded: false,
		})
	}
}

// ParentDirForTreeNew — תיקיית יעד ליצירה לפי צומת נבחר בעץ.
func ParentDirForTreeNew(sel *treeNode, projectRoot string) string {
	if sel != nil {
		if sel.IsDir {
			return sel.Path
		}
		return filepath.Dir(sel.Path)
	}
	return projectRoot
}
