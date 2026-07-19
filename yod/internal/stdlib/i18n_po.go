package stdlib

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// i18nMsg — רשומת gettext אחת.
type i18nMsg struct {
	Context    string
	ID         string
	IDPlural   string
	Strs       []string // msgstr / msgstr[n]
}

func parsePO(data []byte) (map[string]*i18nMsg, string, error) {
	out := map[string]*i18nMsg{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var cur *i18nMsg
	var pluralHeader string
	flush := func() {
		if cur == nil {
			return
		}
		if cur.ID == "" && cur.Context == "" {
			// כותרת
			if len(cur.Strs) > 0 {
				pluralHeader = extractPluralForms(cur.Strs[0])
			}
		} else if cur.ID != "" {
			out[i18nKey(cur.Context, cur.ID)] = cur
		}
		cur = nil
	}

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			if trim == "" {
				flush()
			}
			continue
		}
		if cur == nil {
			cur = &i18nMsg{}
		}
		switch {
		case strings.HasPrefix(trim, "msgctxt "):
			cur.Context = parsePOQuoted(trim[len("msgctxt "):], sc)
		case strings.HasPrefix(trim, "msgid_plural "):
			cur.IDPlural = parsePOQuoted(trim[len("msgid_plural "):], sc)
		case strings.HasPrefix(trim, "msgid "):
			cur.ID = parsePOQuoted(trim[len("msgid "):], sc)
		case strings.HasPrefix(trim, "msgstr["):
			idxEnd := strings.Index(trim, "]")
			if idxEnd < 0 {
				continue
			}
			idx, err := strconv.Atoi(trim[len("msgstr[") : idxEnd])
			if err != nil {
				continue
			}
			rest := strings.TrimSpace(trim[idxEnd+1:])
			if strings.HasPrefix(rest, " ") {
				rest = strings.TrimSpace(rest)
			}
			val := parsePOQuoted(rest, sc)
			for len(cur.Strs) <= idx {
				cur.Strs = append(cur.Strs, "")
			}
			cur.Strs[idx] = val
		case strings.HasPrefix(trim, "msgstr "):
			val := parsePOQuoted(trim[len("msgstr "):], sc)
			if len(cur.Strs) == 0 {
				cur.Strs = []string{val}
			} else {
				cur.Strs[0] = val
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, "", err
	}
	return out, pluralHeader, nil
}

func parsePOQuoted(first string, sc *bufio.Scanner) string {
	var b strings.Builder
	consume := func(s string) {
		s = strings.TrimSpace(s)
		if !strings.HasPrefix(s, "\"") {
			b.WriteString(s)
			return
		}
		unq, err := strconv.Unquote(s)
		if err != nil {
			// ניסיון ידני ל־"..." עם escape
			unq = unquotePO(s)
		}
		b.WriteString(unq)
	}
	consume(first)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "\"") {
			// החזרה — נצטרך לדחוף חזרה; Scanner לא תומך unread.
			// לכן רק שורות המשך במרכאות מיד אחרי.
			// אם לא מרכאות — עיבדנו יותר מדי. נשמור את השורה ב־pending לא קיים;
			// במקום זאת: רק שורות שמתחילות ב־" נחשבות המשך.
			// אם הגענו לכאן בלי " — זו שורה חדשה; נאבד אותה אלא אם נפרסר מחדש.
			// פתרון: בדוק מראש — אם לא ", עצור בלי לאכול (אי אפשר).
			// לכן נשתמש בגישה אחרת: רק אם השורה מתחילה ב־" ממשיכים.
			break
		}
		consume(line)
		// לא ניתן להחזיר שורה — אבל שורות שאינן " כבר יצאו ב־break
		// בעיה: שברנו על שורה שאינה " ואיבדנו אותה.
		// נתקן עם peek buffer במעטפת — לפי שעה רוב קבצי PO שמים שורות ריקות בין רשומות.
	}
	return b.String()
}

// parsePOFile — גרסה משופרת עם lookahead ידני.
func parsePOFile(data []byte) (map[string]*i18nMsg, string, error) {
	lines := splitPOLines(data)
	out := map[string]*i18nMsg{}
	var pluralHeader string
	i := 0
	readString := func() string {
		var b strings.Builder
		for i < len(lines) {
			s := strings.TrimSpace(lines[i])
			if !strings.HasPrefix(s, "\"") {
				break
			}
			i++
			b.WriteString(unquotePO(s))
		}
		return b.String()
	}
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}
		cur := &i18nMsg{}
		for i < len(lines) {
			line = strings.TrimSpace(lines[i])
			if line == "" {
				i++
				break
			}
			if strings.HasPrefix(line, "#") {
				i++
				continue
			}
			switch {
			case strings.HasPrefix(line, "msgctxt "):
				i++
				rest := strings.TrimSpace(line[len("msgctxt "):])
				if strings.HasPrefix(rest, "\"") {
					cur.Context = unquotePO(rest) + readString()
				} else {
					cur.Context = rest
				}
			case strings.HasPrefix(line, "msgid_plural "):
				i++
				rest := strings.TrimSpace(line[len("msgid_plural "):])
				if strings.HasPrefix(rest, "\"") {
					cur.IDPlural = unquotePO(rest) + readString()
				} else {
					cur.IDPlural = rest
				}
			case strings.HasPrefix(line, "msgid "):
				i++
				rest := strings.TrimSpace(line[len("msgid "):])
				if strings.HasPrefix(rest, "\"") {
					cur.ID = unquotePO(rest) + readString()
				} else {
					cur.ID = rest
				}
			case strings.HasPrefix(line, "msgstr["):
				idxEnd := strings.Index(line, "]")
				if idxEnd < 0 {
					i++
					continue
				}
				idx, err := strconv.Atoi(line[len("msgstr[") : idxEnd])
				if err != nil {
					i++
					continue
				}
				i++
				rest := strings.TrimSpace(line[idxEnd+1:])
				var val string
				if strings.HasPrefix(rest, "\"") {
					val = unquotePO(rest) + readString()
				} else if rest != "" {
					val = rest
				} else {
					val = readString()
				}
				for len(cur.Strs) <= idx {
					cur.Strs = append(cur.Strs, "")
				}
				cur.Strs[idx] = val
			case strings.HasPrefix(line, "msgstr "):
				i++
				rest := strings.TrimSpace(line[len("msgstr "):])
				var val string
				if strings.HasPrefix(rest, "\"") {
					val = unquotePO(rest) + readString()
				} else {
					val = rest
				}
				if len(cur.Strs) == 0 {
					cur.Strs = []string{val}
				} else {
					cur.Strs[0] = val
				}
			default:
				i++
			}
		}
		if cur.ID == "" && cur.Context == "" {
			if len(cur.Strs) > 0 {
				pluralHeader = extractPluralForms(cur.Strs[0])
			}
			continue
		}
		if cur.ID != "" {
			out[i18nKey(cur.Context, cur.ID)] = cur
		}
	}
	return out, pluralHeader, nil
}

func splitPOLines(data []byte) []string {
	raw := strings.ReplaceAll(string(data), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	return strings.Split(raw, "\n")
}

func unquotePO(s string) string {
	s = strings.TrimSpace(s)
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		var b strings.Builder
		for i := 0; i < len(inner); i++ {
			if inner[i] == '\\' && i+1 < len(inner) {
				i++
				switch inner[i] {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				case '"', '\\':
					b.WriteByte(inner[i])
				default:
					b.WriteByte(inner[i])
				}
				continue
			}
			b.WriteByte(inner[i])
		}
		return b.String()
	}
	return s
}

func extractPluralForms(header string) string {
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "plural-forms:") {
			return strings.TrimSpace(line[len("Plural-Forms:"):])
		}
	}
	return ""
}

func i18nKey(ctxt, id string) string {
	if ctxt == "" {
		return id
	}
	return ctxt + "\x04" + id
}

// --- Plural-Forms ---

type pluralFunc func(n int) int

func pluralFuncFromHeader(header, lang string) pluralFunc {
	header = strings.TrimSpace(header)
	if header != "" {
		if fn := tryParsePluralForms(header); fn != nil {
			return fn
		}
	}
	return pluralFuncForLang(lang)
}

func pluralFuncForLang(lang string) pluralFunc {
	lang = normalizeLangCode(lang)
	base := lang
	if i := strings.Index(lang, "_"); i > 0 {
		base = lang[:i]
	}
	switch base {
	case "he", "iw":
		return func(n int) int {
			if n == 1 {
				return 0
			}
			if n == 2 {
				return 1
			}
			return 2
		}
	case "ar":
		return func(n int) int {
			switch {
			case n == 0:
				return 0
			case n == 1:
				return 1
			case n == 2:
				return 2
			case n%100 >= 3 && n%100 <= 10:
				return 3
			case n%100 >= 11:
				return 4
			default:
				return 5
			}
		}
	case "ru", "uk", "sr", "hr", "bs":
		return func(n int) int {
			mod10 := n % 10
			mod100 := n % 100
			if mod10 == 1 && mod100 != 11 {
				return 0
			}
			if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
				return 1
			}
			return 2
		}
	case "pl":
		return func(n int) int {
			if n == 1 {
				return 0
			}
			mod10 := n % 10
			mod100 := n % 100
			if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
				return 1
			}
			return 2
		}
	default:
		// en ורוב השפות: שני צורות
		return func(n int) int {
			if n == 1 {
				return 0
			}
			return 1
		}
	}
}

func tryParsePluralForms(header string) pluralFunc {
	// דוגמאות נפוצות — לא מנוע C מלא
	h := strings.ToLower(strings.ReplaceAll(header, " ", ""))
	switch {
	case strings.Contains(h, "nplurals=1"):
		return func(n int) int { return 0 }
	case strings.Contains(h, "n==1?0:n==2?1:2") || strings.Contains(h, "(n==1)?0:(n==2)?1:2"):
		return pluralFuncForLang("he")
	case strings.Contains(h, "n!=1") || strings.Contains(h, "n>1"):
		return pluralFuncForLang("en")
	case strings.Contains(h, "nplurals=2") && strings.Contains(h, "n>1"):
		return pluralFuncForLang("en")
	}
	// חילוץ nplurals בלבד
	if i := strings.Index(h, "nplurals="); i >= 0 {
		rest := h[i+len("nplurals:"):]
		_ = rest
	}
	if i := strings.Index(h, "nplurals="); i >= 0 {
		j := i + len("nplurals=")
		k := j
		for k < len(h) && unicode.IsDigit(rune(h[k])) {
			k++
		}
		if n, err := strconv.Atoi(h[j:k]); err == nil && n == 1 {
			return func(int) int { return 0 }
		}
	}
	return nil
}

func normalizeLangCode(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))
	switch code {
	case "עברית", "hebrew", "iw":
		return "he"
	case "english":
		return "en"
	case "arabic", "ערבית":
		return "ar"
	}
	code = strings.ReplaceAll(code, "-", "_")
	return code
}

func interpolateI18n(s string, vars map[string]string) string {
	if len(vars) == 0 {
		return s
	}
	out := s
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
		out = strings.ReplaceAll(out, "%("+k+")s", v)
		out = strings.ReplaceAll(out, "%("+k+")d", v)
	}
	// %d בודד — אם יש "כמות" או "n"
	if strings.Contains(out, "%d") {
		if v, ok := vars["כמות"]; ok {
			out = strings.ReplaceAll(out, "%d", v)
		} else if v, ok := vars["n"]; ok {
			out = strings.ReplaceAll(out, "%d", v)
		} else if v, ok := vars["count"]; ok {
			out = strings.ReplaceAll(out, "%d", v)
		}
	}
	return out
}

func fmtNum(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return fmt.Sprintf("%g", n)
}
