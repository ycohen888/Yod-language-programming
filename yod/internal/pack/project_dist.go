package pack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/project"
)

// שמות תיקיות נתונים/בדיקה שלא נכנסים להפצה נקייה.
var runtimeDataDirNames = map[string]bool{
	"נתונים":         true,
	"data":           true,
	"Data":           true,
	"DATA":           true,
	"_בדיקת_נתיבים":  true,
	"__pycache__":    true,
	".git":           true,
	"node_modules":   true,
	project.DistDirName: true,
	"הפצה":           true,
}

func isRuntimeDataDir(name string) bool {
	if runtimeDataDirNames[name] {
		return true
	}
	lower := strings.ToLower(name)
	if lower == "data" {
		return true
	}
	if strings.HasPrefix(name, "_בדיק") || strings.HasPrefix(lower, "_test") {
		return true
	}
	return false
}

func isPackSkipSource(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(name, "בדיקה_") || strings.HasPrefix(lower, "test_") {
		return true
	}
	return false
}

// rewriteSiblingIncludes — במקור פיתוח: ..\עיצוב; בהפצה העיצוב בתוך dist_exe.
func rewriteSiblingIncludes(script []byte) []byte {
	s := string(script)
	repls := []struct{ old, new string }{
		{`..\עיצוב\`, `עיצוב\`},
		{`../עיצוב/`, `עיצוב/`},
		{`"..\עיצוב`, `"עיצוב`},
		{`"../עיצוב`, `"עיצוב`},
		{`'\\..\\עיצוב'`, `'\\עיצוב'`},
		{`"\\..\\עיצוב"`, `"\\עיצוב"`},
		{` + "\\..\\עיצוב"`, ` + "\\עיצוב"`},
		{`+"\\..\\עיצוב"`, `+"\\עיצוב"`},
	}
	for _, r := range repls {
		s = strings.ReplaceAll(s, r.old, r.new)
	}
	return []byte(s)
}

// RemoveRuntimeData מוחק מתיקיית היעד נתונים/בדיקות — הפצה נקיה.
func RemoveRuntimeData(dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() && isRuntimeDataDir(name) {
			if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
				return fmt.Errorf("ניקוי %s: %w", name, err)
			}
		}
	}
	return nil
}

// PrepareBinaryDist מנקה מקורות ישנים מ־dist_exe ומשאיר רק בינאריים/איקון/מניפסט.
func PrepareBinaryDist(srcPath, distDir string) error {
	srcPath = filepath.Clean(srcPath)
	distDir = filepath.Clean(distDir)
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
	}
	if err := RemoveRuntimeData(distDir); err != nil {
		return err
	}
	if err := removePackagedSources(distDir); err != nil {
		return err
	}
	_ = copyProjectIcon(srcPath, distDir)
	// איקון גם משורש הפרויקט אם המקור בתיקיית משנה
	if root := project.FindProjectRoot(srcPath); root != "" && root != filepath.Dir(srcPath) {
		_ = copyProjectIcon(root, distDir)
	}
	return nil
}

// removePackagedSources מוחק *.יוד / עיצוב / מדריך גלויים מהפצה (שרידים מאריזה ישנה).
func removePackagedSources(distDir string) error {
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		full := filepath.Join(distDir, name)
		if e.IsDir() {
			// כל תיקייה גלויה (עיצוב/מדריך/נתונים כבר ב־RemoveRuntimeData) — לא שייכת להפצה בינארית
			if err := os.RemoveAll(full); err != nil {
				return err
			}
			continue
		}
		lower := strings.ToLower(name)
		ext := strings.ToLower(filepath.Ext(name))
		// משאירים רק EXE, מניפסט ואיקונים
		switch ext {
		case ".exe", ".manifest", ".ico":
			continue
		}
		if strings.HasPrefix(lower, ".") {
			continue // לא מוחקים קבצי מערכת נסתרים אם יש
		}
		_ = os.Remove(full)
	}
	return nil
}

// PrepareCleanProjectDist מעתיק מקורות נקיים (+ עיצוב אחות) ל־dist_exe ומנקה נתונים.
// משמש בעיקר ל־`יוד ארוז --תיקייה` (הפצה עם קבצים על הדיסק).
func PrepareCleanProjectDist(srcPath, distDir string) error {
	srcPath = filepath.Clean(srcPath)
	distDir = filepath.Clean(distDir)
	root := project.FindProjectRoot(srcPath)
	if root == "" {
		root = filepath.Dir(srcPath)
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
	}
	if err := RemoveRuntimeData(distDir); err != nil {
		return err
	}

	// מקורות .יוד מהפרויקט (בלי בדיקות / בלי תיקיות נתונים)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if path == root {
				return nil
			}
			if isRuntimeDataDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if isPackSkipSource(info.Name()) {
			return nil
		}
		if !project.IsYodSource(path) && !isProjectAsset(path, rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if project.IsYodSource(path) {
			data = rewriteSiblingIncludes(data)
		}
		dest := filepath.Join(distDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
	if err != nil {
		return err
	}

	// ספריית עיצוב אחות (פרוייקט דוגמה/עיצוב) → dist_exe/עיצוב
	siblingDesign := filepath.Join(filepath.Dir(root), "עיצוב")
	if st, err := os.Stat(siblingDesign); err == nil && st.IsDir() {
		destDesign := filepath.Join(distDir, "עיצוב")
		_ = os.RemoveAll(destDesign)
		if err := copyDirFiltered(siblingDesign, destDesign); err != nil {
			return fmt.Errorf("העתקת עיצוב: %w", err)
		}
	}

	// וידוא סופי: בלי נתונים בהפצה
	return RemoveRuntimeData(distDir)
}

// isProjectAsset — מדריך HTML, לוגו/איקון, נכסים סטטיים להפצה.
func isProjectAsset(path, rel string) bool {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".b64", ".json", ".txt", ".md", ".po", ".mo", ".pot":
		// לא מעתיקים מניפסטים/מצב מתוך נתונים — Walk מדלג על תיקיית נתונים
	default:
		return false
	}
	// איקונים/לוגו בשורש הפרויקט
	switch base {
	case "app.ico", "yod.ico", "icon.ico", "לוגו.png", "logo.png":
		return true
	}
	if strings.HasSuffix(base, ".ico") && (base == "יוד.ico" || strings.Contains(base, "לוגו")) {
		return true
	}
	// תיקיות נכסים / מדריך / assets
	relSlash := filepath.ToSlash(rel)
	parts := strings.Split(relSlash, "/")
	if len(parts) == 0 {
		return false
	}
	top := parts[0]
	switch top {
	case "מדריך", "נכסים", "assets", "resources", "img", "images", "icons", "תרגומים", "languages", "locale", "locales":
		return true
	}
	return false
}

func copyDirFiltered(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if path != src && isRuntimeDataDir(name) {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0755)
		}
		in, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer in.Close()
		outPath := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		out, err := os.Create(outPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
