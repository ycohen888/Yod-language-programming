//go:build windows

package editor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lxn/walk"
)

// fileEntry — קובץ או תיקייה ברשימת הסייר (ListBox, לא TreeView — יציב יותר ב־walk).
type fileEntry struct {
	name  string
	path  string
	isDir bool
}

// FileListModel — מודל רשימה לתיקיית פרויקט עם ניווט לתיקיות משנה.
type FileListModel struct {
	walk.ListModelBase
	root    string
	cwd     string
	entries []fileEntry
}

func NewFileListModel() *FileListModel {
	return &FileListModel{}
}

func (m *FileListModel) ItemCount() int { return len(m.entries) }

func (m *FileListModel) Value(index int) interface{} {
	if index < 0 || index >= len(m.entries) {
		return ""
	}
	e := m.entries[index]
	if e.name == ".." {
		return ".."
	}
	if e.isDir {
		return e.name + string(os.PathSeparator)
	}
	return e.name
}

func (m *FileListModel) RootPath() string { return m.root }
func (m *FileListModel) Cwd() string      { return m.cwd }

func (m *FileListModel) EntryAt(index int) *fileEntry {
	if index < 0 || index >= len(m.entries) {
		return nil
	}
	e := m.entries[index]
	return &e
}

// SetRoot מגדיר/מנקה את שורש הפרויקט ומרענן את הרשימה.
func (m *FileListModel) SetRoot(absPath string) {
	absPath = strings.TrimSpace(absPath)
	if absPath == "" {
		m.root = ""
		m.cwd = ""
		m.entries = nil
		m.PublishItemsReset()
		return
	}
	abs, err := filepath.Abs(absPath)
	if err != nil {
		abs = absPath
	}
	m.root = abs
	m.cwd = abs
	m.reload()
}

// Refresh מרענן את התיקייה הנוכחית מהדיסק.
func (m *FileListModel) Refresh() {
	if m.root == "" {
		m.entries = nil
		m.PublishItemsReset()
		return
	}
	m.reload()
}

// Enter נכנס לתיקייה (או עולה עם "..").
func (m *FileListModel) Enter(absPath string) {
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
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return
	}
	m.cwd = abs
	m.reload()
}

func (m *FileListModel) reload() {
	m.entries = nil
	if m.cwd == "" {
		m.PublishItemsReset()
		return
	}
	root := filepath.Clean(m.root)
	cwd := filepath.Clean(m.cwd)
	if cwd != root {
		m.entries = append(m.entries, fileEntry{name: "..", path: filepath.Dir(cwd), isDir: true})
	}
	entries, err := os.ReadDir(cwd)
	if err != nil {
		m.PublishItemsReset()
		return
	}
	var dirs, files []fileEntry
	for _, e := range entries {
		name := e.Name()
		if shouldHideName(name) {
			continue
		}
		childPath := filepath.Join(cwd, name)
		info, err := e.Info()
		if err != nil {
			continue
		}
		ent := fileEntry{name: name, path: childPath, isDir: info.IsDir()}
		if info.IsDir() {
			dirs = append(dirs, ent)
		} else {
			files = append(files, ent)
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
	})
	m.entries = append(m.entries, dirs...)
	m.entries = append(m.entries, files...)
	m.PublishItemsReset()
}

// IndexOfPath מחזיר אינדקס ברשימה או -1.
func (m *FileListModel) IndexOfPath(absPath string) int {
	want, err := filepath.Abs(absPath)
	if err != nil {
		want = absPath
	}
	want = filepath.Clean(want)
	for i, e := range m.entries {
		if filepath.Clean(e.path) == want {
			return i
		}
	}
	return -1
}

func shouldHideName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "dist", "node_modules", "vendor", "__pycache__":
		return true
	}
	if strings.EqualFold(filepath.Ext(name), ".exe") {
		return true
	}
	return false
}

// ParentDirForNew — תיקיית יעד ליצירת קובץ/תיקייה לפי בחירה.
func ParentDirForNew(sel *fileEntry, listCwd, projectRoot string) string {
	if sel != nil && sel.name != ".." {
		if sel.isDir {
			return sel.path
		}
		return filepath.Dir(sel.path)
	}
	if listCwd != "" {
		return listCwd
	}
	return projectRoot
}
