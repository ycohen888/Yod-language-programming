// Package vfs — מערכת קבצים וירטואלית לחבילת EXE (YODBUND1).
package vfs

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FS מחזיקה קבצים ותיקיות יחסיים (מפתחות עם /).
type FS struct {
	Root  string // תיקיית ה־EXE / בסיס הריצה — לנתיבים מוחלטים
	files map[string][]byte
	dirs  map[string]bool
}

// New יוצר FS ריק.
func New(root string) *FS {
	fs := &FS{
		files: make(map[string][]byte),
		dirs:  map[string]bool{"": true},
	}
	if root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			fs.Root = abs
		} else {
			fs.Root = root
		}
	}
	return fs
}

// Add מוסיף קובץ לפי נתיב יחסי (\/ מנורמל ל־/).
func (fs *FS) Add(rel string, data []byte) {
	key := Normalize(rel)
	if key == "" || strings.HasSuffix(key, "/") {
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	fs.files[key] = cp
	dir := parentKey(key)
	for dir != "" {
		fs.dirs[dir] = true
		dir = parentKey(dir)
	}
	fs.dirs[""] = true
}

// Files מחזיר עותק של מפתחות הקבצים (לבדיקות / --רשימה).
func (fs *FS) Files() []string {
	out := make([]string, 0, len(fs.files))
	for k := range fs.files {
		out = append(out, k)
	}
	return out
}

// Normalize מנרמל נתיב יחסי למפתח VFS.
func Normalize(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	for strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	parts := strings.Split(p, "/")
	var stack []string
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		stack = append(stack, part)
	}
	return strings.Join(stack, "/")
}

func parentKey(key string) string {
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return ""
	}
	return key[:i]
}

// keyFor ממפה נתיב מערכת (מוחלט/יחסי) למפתח בחבילה.
func (fs *FS) keyFor(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	path = filepath.Clean(path)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if fs.Root != "" {
		rel, err := filepath.Rel(fs.Root, path)
		if err == nil {
			rel = filepath.ToSlash(rel)
			if rel == "." {
				return "", true
			}
			if !strings.HasPrefix(rel, "../") && rel != ".." {
				return Normalize(rel), true
			}
		}
	}
	// נתיב יחסי כבר (או מחוץ לשורש) — ננסה כמפתח ישיר אחרי ניקוי
	n := Normalize(filepath.ToSlash(path))
	// אם זה נראה כמו נתיב Windows מוחלט (C:/...) — לא מפתח יחסי
	if len(n) >= 2 && n[1] == ':' {
		return "", false
	}
	if strings.HasPrefix(n, "//") {
		return "", false
	}
	return n, true
}

// Get קורא לפי מפתח יחסי מנורמל (בלי Root).
func (fs *FS) Get(key string) ([]byte, bool) {
	key = Normalize(key)
	data, ok := fs.files[key]
	if !ok {
		return nil, false
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true
}

// Read קורא קובץ מה־VFS לפי נתיב מערכת או יחסי.
func (fs *FS) Read(path string) ([]byte, bool) {
	key, ok := fs.keyFor(path)
	if !ok {
		return fs.Get(path)
	}
	if data, ok := fs.Get(key); ok {
		return data, true
	}
	return fs.Get(path)
}

// Exists — קובץ או תיקייה וירטואלית.
func (fs *FS) Exists(path string) bool {
	key, ok := fs.keyFor(path)
	if !ok {
		return false
	}
	if key == "" {
		return true
	}
	if _, ok := fs.files[key]; ok {
		return true
	}
	return fs.dirs[key]
}

// IsDir — תיקייה וירטואלית (או שורש).
func (fs *FS) IsDir(path string) bool {
	key, ok := fs.keyFor(path)
	if !ok {
		return false
	}
	if key == "" {
		return true
	}
	if fs.dirs[key] {
		return true
	}
	// תחילית של קובץ כלשהו
	prefix := key + "/"
	for f := range fs.files {
		if strings.HasPrefix(f, prefix) {
			return true
		}
	}
	return false
}

// List מחזיר שמות ילדים ישירים בתיקייה וירטואלית.
func (fs *FS) List(path string) ([]string, bool) {
	key, ok := fs.keyFor(path)
	if !ok {
		return nil, false
	}
	if !fs.IsDir(path) {
		return nil, false
	}
	seen := map[string]bool{}
	var out []string
	prefix := ""
	if key != "" {
		prefix = key + "/"
	}
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for f := range fs.files {
		if !strings.HasPrefix(f, prefix) {
			continue
		}
		rest := f[len(prefix):]
		if i := strings.Index(rest, "/"); i >= 0 {
			add(rest[:i])
		} else {
			add(rest)
		}
	}
	for d := range fs.dirs {
		if d == key || !strings.HasPrefix(d, prefix) {
			continue
		}
		rest := d[len(prefix):]
		if i := strings.Index(rest, "/"); i >= 0 {
			add(rest[:i])
		} else if rest != "" {
			add(rest)
		}
	}
	return out, true
}

var (
	mu     sync.RWMutex
	active *FS
)

// Mount מגדיר את ה־VFS הגלובלי לריצה הנוכחית.
func Mount(fs *FS) {
	mu.Lock()
	defer mu.Unlock()
	active = fs
}

// Clear מבטל הטענה (לבדיקות).
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	active = nil
}

// Active מחזיר את ה־VFS המורכב, או nil.
func Active() *FS {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

// ReadPrefer — קודם VFS, אחר כך דיסק.
func ReadPrefer(path string) ([]byte, error) {
	if fs := Active(); fs != nil {
		if data, ok := fs.Read(path); ok {
			return data, nil
		}
	}
	return os.ReadFile(path)
}

// ExistsPrefer — VFS או דיסק.
func ExistsPrefer(path string) (bool, error) {
	if fs := Active(); fs != nil && fs.Exists(path) {
		return true, nil
	}
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsDirPrefer — VFS או דיסק.
func IsDirPrefer(path string) (bool, error) {
	if fs := Active(); fs != nil && fs.IsDir(path) {
		return true, nil
	}
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return st.IsDir(), nil
}
