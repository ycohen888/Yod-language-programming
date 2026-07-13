package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MainFileName — שם קובץ הכניסה הקבוע של כל פרויקט יוד.
const MainFileName = "התחל.יוד"

// MainTemplate — תוכן ברירת מחדל לקובץ הראשי החדש.
const MainTemplate = `// התחל.יוד — נקודת הכניסה של הפרויקט
// כללו כאן קבצים אחרים מהפרויקט, למשל:
// כלול "עזר.יוד"

הדפס: "שלום מפרויקט יוד"
`

// MainPath מחזיר נתיב מלא לקובץ הראשי בתיקיית הפרויקט.
func MainPath(projectRoot string) string {
	return filepath.Join(projectRoot, MainFileName)
}

// IsMain מדווח אם הנתיב הוא קובץ הראשי של פרויקט.
func IsMain(path string) bool {
	return strings.EqualFold(filepath.Base(path), MainFileName)
}

// EnsureMain יוצר את התחל.יוד אם חסר. מחזיר את הנתיב המלא.
func EnsureMain(projectRoot string) (string, error) {
	if projectRoot == "" {
		return "", fmt.Errorf("אין תיקיית פרויקט")
	}
	main := MainPath(projectRoot)
	if _, err := os.Stat(main); err == nil {
		return main, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.WriteFile(main, []byte(MainTemplate), 0644); err != nil {
		return "", fmt.Errorf("לא הצלחתי ליצור %s: %w", MainFileName, err)
	}
	return main, nil
}

// FindProjectRoot מחפש כלפי מעלה תיקייה שמכילה התחל.יוד.
// start יכול להיות קובץ או תיקייה. מחזיר "" אם לא נמצא.
func FindProjectRoot(start string) string {
	if start == "" {
		return ""
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		abs = start
	}
	dir := abs
	if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
		dir = filepath.Dir(abs)
	}
	for {
		main := filepath.Join(dir, MainFileName)
		if fi, err := os.Stat(main); err == nil && !fi.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// IsYodSource — האם הנתיב הוא קובץ מקור יוד (.יוד / .yod)
func IsYodSource(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".יוד" || strings.EqualFold(ext, ".yod")
}

// ResolveEntry — אם path תיקייה, מחזיר את התחל.יוד שבתוכה; אחרת את path עצמו.
func ResolveEntry(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return path, nil
	}
	main := MainPath(path)
	if _, err := os.Stat(main); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("בתיקייה חסר קובץ ראשי %s (זהו קובץ ההתחלה של הפרויקט)", MainFileName)
		}
		return "", err
	}
	return main, nil
}

// ListYodFiles מחזיר שמות קבצי .יוד יחסיים לשורש (בלי התחל.יוד), לעומק מוגבל.
func ListYodFiles(projectRoot string) ([]string, error) {
	if projectRoot == "" {
		return nil, nil
	}
	var out []string
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "." || name == ".." {
				return nil
			}
			if strings.HasPrefix(name, ".") || name == "dist" || name == "node_modules" || name == "vendor" {
				if path != projectRoot {
					return filepath.SkipDir
				}
			}
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext != ".יוד" && !strings.EqualFold(ext, ".yod") {
			return nil
		}
		if strings.EqualFold(info.Name(), MainFileName) {
			return nil
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}
