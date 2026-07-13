package console

import (
	"os"
	"runtime"
	"strings"
	"unicode"
)

// Rewrite מסדר טקסט עברי לקונסול LTR רק אם ביקשו במפורש.
// אצל רוב משתמשי Windows (כולל CMD עם UTF-8) ההיפוך מקלקל — לכן כבוי כברירת מחדל.
func Rewrite(s string) string {
	if s == "" || !containsRTL(s) || !needConsoleReorder() {
		return s
	}

	var out strings.Builder
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out.WriteString(reverseString(s[start:i]))
			out.WriteByte('\n')
			start = i + 1
		}
	}
	out.WriteString(reverseString(s[start:]))
	return out.String()
}

func needConsoleReorder() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	switch strings.ToLower(os.Getenv("YOD_CONSOLE_BIDI")) {
	case "1", "on", "true":
		return true
	default:
		return false
	}
}

func containsRTL(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hebrew, r) || unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
