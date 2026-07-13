//go:build windows

package editor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lxn/walk"
)

// fileNode — קובץ או תיקייה בעץ הסייר
type fileNode struct {
	name     string
	path     string
	isDir    bool
	parent   *fileNode
	children []*fileNode
	loaded   bool
}

func newFileNode(name, absPath string, isDir bool, parent *fileNode) *fileNode {
	return &fileNode{name: name, path: absPath, isDir: isDir, parent: parent}
}

func (n *fileNode) Text() string { return n.name }

func (n *fileNode) Parent() walk.TreeItem {
	if n.parent == nil {
		return nil
	}
	return n.parent
}

func (n *fileNode) ChildCount() int {
	if !n.isDir {
		return 0
	}
	n.ensureChildren()
	return len(n.children)
}

func (n *fileNode) ChildAt(i int) walk.TreeItem {
	n.ensureChildren()
	if i < 0 || i >= len(n.children) {
		return nil
	}
	return n.children[i]
}

func (n *fileNode) HasChild() bool {
	if !n.isDir {
		return false
	}
	if !n.loaded {
		return true // מציג חץ עד שנטען
	}
	return len(n.children) > 0
}

func (n *fileNode) Image() interface{} {
	return n.path
}

func (n *fileNode) ensureChildren() {
	if !n.isDir || n.loaded {
		return
	}
	n.loaded = true
	n.children = nil
	entries, err := os.ReadDir(n.path)
	if err != nil {
		return
	}
	var dirs, files []*fileNode
	for _, e := range entries {
		name := e.Name()
		if shouldHideName(name) {
			continue
		}
		childPath := filepath.Join(n.path, name)
		info, err := e.Info()
		if err != nil {
			continue
		}
		node := newFileNode(name, childPath, info.IsDir(), n)
		if info.IsDir() {
			dirs = append(dirs, node)
		} else {
			files = append(files, node)
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
	})
	n.children = append(dirs, files...)
}

func (n *fileNode) resetChildren() {
	n.loaded = false
	n.children = nil
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

// FileTreeModel — מודל Lazy לתיקיית פרויקט
type FileTreeModel struct {
	walk.TreeModelBase
	root *fileNode
}

func NewFileTreeModel() *FileTreeModel {
	return &FileTreeModel{}
}

func (m *FileTreeModel) LazyPopulation() bool { return true }

func (m *FileTreeModel) RootCount() int {
	if m.root == nil {
		return 0
	}
	return 1
}

func (m *FileTreeModel) RootAt(i int) walk.TreeItem {
	if i != 0 || m.root == nil {
		return nil
	}
	return m.root
}

func (m *FileTreeModel) RootPath() string {
	if m.root == nil {
		return ""
	}
	return m.root.path
}

// SetRoot מגדיר/מנקה את שורש הפרויקט ומרענן את העץ.
func (m *FileTreeModel) SetRoot(absPath string) {
	absPath = strings.TrimSpace(absPath)
	if absPath == "" {
		m.root = nil
		m.PublishItemsReset(nil)
		return
	}
	abs, err := filepath.Abs(absPath)
	if err != nil {
		abs = absPath
	}
	m.root = newFileNode(filepath.Base(abs), abs, true, nil)
	m.PublishItemsReset(nil)
}

// Refresh מרענן את כל העץ מהדיסק.
func (m *FileTreeModel) Refresh() {
	if m.root == nil {
		m.PublishItemsReset(nil)
		return
	}
	path := m.root.path
	m.root = newFileNode(filepath.Base(path), path, true, nil)
	m.PublishItemsReset(nil)
}

// RefreshNode מרענן ילדים של תיקייה (או הורה של קובץ).
func (m *FileTreeModel) RefreshNode(n *fileNode) {
	if n == nil {
		m.Refresh()
		return
	}
	target := n
	if !n.isDir {
		if n.parent != nil {
			target = n.parent
		} else {
			m.Refresh()
			return
		}
	}
	target.resetChildren()
	m.PublishItemsReset(target)
}

// FindByPath מחפש צומת לפי נתיב מלא.
func (m *FileTreeModel) FindByPath(absPath string) *fileNode {
	if m.root == nil {
		return nil
	}
	want, err := filepath.Abs(absPath)
	if err != nil {
		want = absPath
	}
	want = filepath.Clean(want)
	return findFileNode(m.root, want)
}

func findFileNode(n *fileNode, want string) *fileNode {
	if filepath.Clean(n.path) == want {
		return n
	}
	if !n.isDir {
		return nil
	}
	n.ensureChildren()
	for _, c := range n.children {
		if found := findFileNode(c, want); found != nil {
			return found
		}
	}
	return nil
}

// ParentDirForNew — תיקיית יעד ליצירת קובץ/תיקייה לפי בחירה.
func ParentDirForNew(sel *fileNode, projectRoot string) string {
	if sel != nil {
		if sel.isDir {
			return sel.path
		}
		if sel.parent != nil {
			return sel.parent.path
		}
		return filepath.Dir(sel.path)
	}
	return projectRoot
}
