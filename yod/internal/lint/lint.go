package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/lexer"
	"yod/internal/parser"
)

// Issue — אזהרת סגנון / בדיקה סטטית.
type Issue struct {
	File    string
	Line    int
	Message string
}

func (i Issue) String() string {
	if i.File != "" {
		return fmt.Sprintf("%s:%d: %s", i.File, i.Line, i.Message)
	}
	return fmt.Sprintf("שורה %d: %s", i.Line, i.Message)
}

// CheckSource בודק מקור יוד יחיד.
func CheckSource(filename, source string) []Issue {
	var issues []Issue

	l := lexer.New(source)
	p := parser.New(l)
	_ = p.ParseProgram()
	for _, e := range p.Errors() {
		issues = append(issues, Issue{File: filename, Line: lineFromMsg(e), Message: "שגיאת תחביר: " + e})
	}

	issues = append(issues, checkRiskyJuxtaMinus(filename, source)...)
	issues = append(issues, checkPreferImport(filename, source)...)
	return issues
}

// CheckPath בודק קובץ או תיקייה (רקורסיבי ל־.יוד/.yod).
func CheckPath(path string) ([]Issue, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var files []string
	if fi.IsDir() {
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			ext := filepath.Ext(p)
			if ext == ".יוד" || ext == ".yod" {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		files = []string{path}
	}
	var all []Issue
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return all, err
		}
		all = append(all, CheckSource(f, string(data))...)
	}
	return all, nil
}

func lineFromMsg(msg string) int {
	var line int
	if _, err := fmt.Sscanf(msg, "שורה %d:", &line); err == nil {
		return line
	}
	return 1
}

// checkRiskyJuxtaMinus — מזהה ואז מינוס צמוד בלי רווח: שם-1 עלול לבלבל.
func checkRiskyJuxtaMinus(filename, source string) []Issue {
	var issues []Issue
	lines := strings.Split(source, "\n")
	for li, line := range lines {
		runes := []rune(line)
		for i := 0; i+1 < len(runes); i++ {
			if runes[i] == '-' {
				continue
			}
			// מזהה ואז מיד '-' ואז ספרה/מזהה בלי רווח
			if isIdentPartRune(runes[i]) && runes[i+1] == '-' {
				if i+2 < len(runes) && (isIdentPartRune(runes[i+2]) || isDigitRune(runes[i+2])) {
					// דלג על `--` או `-=`
					if runes[i+1] == '-' && i+2 < len(runes) && (runes[i+2] == '-' || runes[i+2] == '=') {
						continue
					}
					issues = append(issues, Issue{
						File: filename,
						Line: li + 1,
						Message: "סגנון: מינוס צמוד אחרי מזהה עלול להתפרש כחיסור — העדיפו רווח (א - ב) או קידומת בארגומנט (: -1)",
					})
					break
				}
			}
		}
	}
	return issues
}

func isIdentStartRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= 0x0590 && r <= 0x05FF) || (r >= 0xFB1D && r <= 0xFB4F)
}

func isIdentPartRune(r rune) bool {
	return isIdentStartRune(r) || isDigitRune(r)
}

func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

// checkPreferImport — כלול "קובץ.יוד" בפרויקטים חדשים: עדיף יבא.
func checkPreferImport(filename, source string) []Issue {
	var issues []Issue
	lines := strings.Split(source, "\n")
	for li, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "//") {
			continue
		}
		if !strings.Contains(trim, "כלול") {
			continue
		}
		if strings.Contains(trim, ".יוד") || strings.Contains(trim, ".yod") {
			issues = append(issues, Issue{
				File:    filename,
				Line:    li + 1,
				Message: "סגנון: לקבצי משתמש העדיפו «יבא שם מתוך \"קובץ.יוד\"» (כלול נשאר לתאימות ולספריות מובנות)",
			})
		}
	}
	return issues
}
