package stdlib

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	appBaseMu  sync.RWMutex
	appBaseDir string
)

// SetAppBaseDir — תיקיית הפרויקט/הקובץ הרץ (לחיפוש יוד.ico וכו').
func SetAppBaseDir(dir string) {
	appBaseMu.Lock()
	defer appBaseMu.Unlock()
	if dir == "" {
		appBaseDir = ""
		return
	}
	if abs, err := filepath.Abs(dir); err == nil {
		appBaseDir = abs
	} else {
		appBaseDir = dir
	}
}

// AppBaseDir מחזיר את תיקיית הבסיס של הריצה הנוכחית.
func AppBaseDir() string {
	appBaseMu.RLock()
	defer appBaseMu.RUnlock()
	return appBaseDir
}

// FindAppIconPath מחפש יוד.ico / app.ico ליד הפרויקט או ה־EXE.
func FindAppIconPath() string {
	names := []string{"יוד.ico", "yod.ico", "app.ico", "icon.ico"}
	seen := map[string]bool{}
	var dirs []string
	add := func(d string) {
		if d == "" {
			return
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		dirs = append(dirs, abs)
	}
	add(AppBaseDir())
	if wd, err := os.Getwd(); err == nil {
		add(wd)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err2 := filepath.EvalSymlinks(exe); err2 == nil {
			exe = resolved
		}
		add(filepath.Dir(exe))
	}
	// עלייה לתיקיות הורה (עד 4) — למקרה שהקובץ הראשי בתיקיית משנה
	base := AppBaseDir()
	for i := 0; i < 4 && base != "" && base != filepath.Dir(base); i++ {
		base = filepath.Dir(base)
		add(base)
	}
	for _, d := range dirs {
		for _, n := range names {
			p := filepath.Join(d, n)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}
