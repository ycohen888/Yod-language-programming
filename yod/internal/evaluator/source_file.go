package evaluator

import "strings"

// ערימת קבצי מקור — כדי ששגיאות מ־כלול יציינו את הקובץ הנכון.
var sourceFileStack []string

// PushSourceFile מסמן את קובץ המקור הנוכחי להערכת שגיאות.
func PushSourceFile(path string) {
	if path == "" {
		return
	}
	sourceFileStack = append(sourceFileStack, path)
}

// PopSourceFile מסיר את קובץ המקור האחרון מהערימה.
func PopSourceFile() {
	if len(sourceFileStack) == 0 {
		return
	}
	sourceFileStack = sourceFileStack[:len(sourceFileStack)-1]
}

func currentSourceFile() string {
	n := len(sourceFileStack)
	if n == 0 {
		return ""
	}
	return sourceFileStack[n-1]
}

func extractLineNum(msg string) int {
	lower := strings.ToLower(msg)
	for _, prefix := range []string{"שורה ", "line "} {
		idx := strings.Index(lower, strings.ToLower(prefix))
		if idx < 0 {
			continue
		}
		// חיפוש במחרוזת המקורית לפי אותה עמדה (עברית — אורך זהה בבתים? עדיף Index על msg)
		idx = strings.Index(msg, prefix)
		if idx < 0 {
			idx = strings.Index(lower, "line ")
			if idx < 0 {
				continue
			}
			prefix = msg[idx : idx+5] // "line "
		}
		rest := msg[idx+len(prefix):]
		n := 0
		for _, r := range rest {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		if n > 0 {
			return n
		}
	}
	return 0
}
