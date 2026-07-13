//go:build windows

package editor

import (
	"strings"
	"unicode"
	"unicode/utf16"
)

// toggleCommentLines — אם כל השורות (עם תוכן) מוערות: הסר //, אחרת הוסף
func toggleCommentLines(lines []string) []string {
	allCommented := true
	any := false
	for _, line := range lines {
		t := strings.TrimSpace(stripBidiMarks(line))
		if t == "" {
			continue
		}
		any = true
		if !strings.HasPrefix(t, "//") {
			allCommented = false
			break
		}
	}
	if !any {
		allCommented = false
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if allCommented {
			out[i] = uncommentOneLine(line)
		} else {
			out[i] = commentOneLine(line)
		}
	}
	return out
}

func commentOneLine(line string) string {
	runes := []rune(line)
	i := 0
	for i < len(runes) && (isBidiMark(runes[i]) || runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	prefix := string(runes[:i])
	rest := string(runes[i:])
	t := strings.TrimSpace(stripBidiMarks(rest))
	if strings.HasPrefix(t, "//") {
		return line
	}
	return prefix + "// " + rest
}

func uncommentOneLine(line string) string {
	runes := []rune(line)
	i := 0
	for i < len(runes) && (isBidiMark(runes[i]) || runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	prefix := string(runes[:i])
	rest := string(runes[i:])
	rest = stripBidiMarks(rest)
	switch {
	case strings.HasPrefix(rest, "// "):
		rest = rest[3:]
	case strings.HasPrefix(rest, "//"):
		rest = rest[2:]
	}
	return prefix + rest
}

// leadingSpacesOfLine — רווחי הזחה בתחילת שורה (אחרי בידי)
func leadingSpacesOfLine(line string) string {
	runes := []rune(line)
	i := 0
	for i < len(runes) && isBidiMark(runes[i]) {
		i++
	}
	start := i
	for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	var b strings.Builder
	for _, r := range runes[start:i] {
		if r == '\t' {
			b.WriteString("  ")
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// lineSuggestsBlockIndent — אחרי Enter כדאי להוסיף רמת הזחה
func lineSuggestsBlockIndent(line string) bool {
	line = strings.TrimSpace(stripBidiMarks(line))
	if line == "" || strings.HasPrefix(line, "//") {
		return false
	}
	if idx := strings.Index(line, "//"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	if strings.HasSuffix(line, "סוף") {
		return false
	}
	openers := []string{
		"אם", "אחרת", "אחרת_אם", "כל_עוד", "עבור",
		"פונקציה", "מחלקה", "נסה", "תפוס", "בחר", "מקרה",
	}
	for _, o := range openers {
		if line == o {
			return true
		}
		if strings.HasPrefix(line, o) {
			rest := line[len(o):]
			if rest == "" {
				return true
			}
			r := []rune(rest)[0]
			if unicode.IsSpace(r) || r == ':' || r == '(' {
				return true
			}
		}
	}
	return false
}

func substringUTF16(text string, start, end int) string {
	u := utf16.Encode([]rune(text))
	if start < 0 {
		start = 0
	}
	if end > len(u) {
		end = len(u)
	}
	if start >= end {
		return ""
	}
	return string(utf16.Decode(u[start:end]))
}

func replaceUTF16Range(text string, start, end int, repl string) string {
	u := utf16.Encode([]rune(text))
	if start < 0 {
		start = 0
	}
	if end > len(u) {
		end = len(u)
	}
	if start > end {
		start, end = end, start
	}
	ru := utf16.Encode([]rune(repl))
	out := append([]uint16{}, u[:start]...)
	out = append(out, ru...)
	out = append(out, u[end:]...)
	return string(utf16.Decode(out))
}
