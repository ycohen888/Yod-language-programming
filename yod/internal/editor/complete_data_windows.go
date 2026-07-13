//go:build windows

package editor

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf16"

	"yod/internal/object"
	"yod/internal/stdlib"
	"yod/internal/token"
)

type completeKind int

const (
	kindKeyword completeKind = iota
	kindBuiltin
	kindMethod
	kindLibrary
)

func (k completeKind) label() string {
	switch k {
	case kindKeyword:
		return "מילת מפתח"
	case kindBuiltin:
		return "פונקציה"
	case kindMethod:
		return "מתודה"
	case kindLibrary:
		return "ספרייה"
	default:
		return ""
	}
}

type completeItem struct {
	Text   string
	Kind   completeKind
	Detail string // למשל "רשימה" / "מחרוזת"
}

type completeMode int

const (
	modeNone completeMode = iota
	modeIdent
	modeMember
	modeInclude // כלול "…
)

type completeQuery struct {
	Mode        completeMode
	Prefix      string
	ReplaceFrom int // אינדקס UTF-16 להתחלת ההחלפה
	Receiver    string
	TypeHint    string // array|string|hash|number|bool|null|module|any
}

var (
	// מילות שמורות — רשימה מפורשת (מקור אמת להשלמה)
	completeKeywords = []string{
		"משתנה", "אם", "אחרת", "אחרת_אם", "כל_עוד", "עבור", "עצור", "המשך",
		"פונקציה", "החזר", "אמת", "שקר", "ריק", "בתוך", "מחלקה", "חדש", "זה",
		"מרחיב", "הורה", "פרטי", "ציבורי", "כלול", "נסה", "תפוס", "זרוק", "סוף",
		"וגם", "או", "לא", "בחר", "מקרה", "ברירת_מחדל",
	}
	completeBuiltins = []string{
		"הדפס", "קלט", "אורך", "הוסף", "למספר", "למחרוזת",
		"סוג", "טווח", "אקראי", "אקראי_בין",
	}
	cachedKeywords []string
	methodCache    map[string][]completeItem
	allMethods     []completeItem
	libraryItems   []completeItem
)

func ensureCompleteData() {
	if cachedKeywords != nil {
		return
	}
	seen := map[string]bool{}
	for _, k := range completeKeywords {
		seen[k] = true
		cachedKeywords = append(cachedKeywords, k)
	}
	// מיזוג עם מילות המפתח מהטוקן (אם נוספו)
	for _, k := range token.Keywords() {
		if !seen[k] {
			seen[k] = true
			cachedKeywords = append(cachedKeywords, k)
		}
	}
	sort.Strings(cachedKeywords)

	methodCache = map[string][]completeItem{
		"string": toMethodItems(object.StringMethodNames(), "מחרוזת"),
		"number": toMethodItems(object.NumberMethodNames(), "מספר"),
		"bool":   toMethodItems(object.BooleanMethodNames(), "בוליאני"),
		"null":   toMethodItems(object.NullMethodNames(), "ריק"),
		"array":  toMethodItems(object.ArrayMethodNames(), "רשימה"),
		"hash":   toMethodItems(object.HashMethodNames(), "מילון"),
	}
	seenM := map[string]bool{}
	for _, list := range methodCache {
		for _, it := range list {
			if seenM[it.Text] {
				continue
			}
			seenM[it.Text] = true
			allMethods = append(allMethods, completeItem{Text: it.Text, Kind: kindMethod, Detail: "מתודה"})
		}
	}
	sort.Slice(allMethods, func(i, j int) bool { return allMethods[i].Text < allMethods[j].Text })

	for _, n := range stdlib.BuiltinNames() {
		libraryItems = append(libraryItems, completeItem{Text: n, Kind: kindLibrary, Detail: "ספרייה"})
	}
}

func toMethodItems(names []string, detail string) []completeItem {
	out := make([]completeItem, len(names))
	for i, n := range names {
		out[i] = completeItem{Text: n, Kind: kindMethod, Detail: detail}
	}
	return out
}

func isBidiMark(r rune) bool {
	switch r {
	case '\u200E', '\u200F', '\u2066', '\u2067', '\u2068', '\u2069', '\u202A', '\u202B', '\u202C', '\u202D', '\u202E':
		return true
	}
	return false
}

func isIdentRune(r rune) bool {
	if isBidiMark(r) {
		return false
	}
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// normalizeHebrew — אותיות סופיות → רגילות (ן→נ וכו') להתאמת קידומת
func normalizeHebrew(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'ך':
			b.WriteRune('כ')
		case 'ם':
			b.WriteRune('מ')
		case 'ן':
			b.WriteRune('נ')
		case 'ף':
			b.WriteRune('פ')
		case 'ץ':
			b.WriteRune('צ')
		default:
			if isBidiMark(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func hebrewHasPrefix(text, prefix string) bool {
	if prefix == "" {
		return true
	}
	nt, np := normalizeHebrew(text), normalizeHebrew(prefix)
	if strings.HasPrefix(nt, np) {
		return true
	}
	// JSON / SQL — התאמה לא רגישה לאותיות לטיניות
	return strings.HasPrefix(strings.ToLower(nt), strings.ToLower(np))
}

func filterComplete(prefix string, items []completeItem) []completeItem {
	if prefix == "" {
		out := make([]completeItem, len(items))
		copy(out, items)
		return out
	}
	var out []completeItem
	for _, it := range items {
		if hebrewHasPrefix(it.Text, prefix) {
			out = append(out, it)
		}
	}
	return out
}

func identSuggestions(prefix string) []completeItem {
	ensureCompleteData()
	var out []completeItem
	for _, k := range cachedKeywords {
		if hebrewHasPrefix(k, prefix) {
			out = append(out, completeItem{Text: k, Kind: kindKeyword, Detail: kindKeyword.label()})
		}
	}
	for _, b := range completeBuiltins {
		if hebrewHasPrefix(b, prefix) {
			out = append(out, completeItem{Text: b, Kind: kindBuiltin, Detail: kindBuiltin.label()})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		// קודם התאמות קצרות יותר (משתנה לפני מילים ארוכות)
		ni, nj := len([]rune(out[i].Text)), len([]rune(out[j].Text))
		if ni != nj {
			return ni < nj
		}
		return out[i].Text < out[j].Text
	})
	return out
}

func librarySuggestions(prefix string) []completeItem {
	ensureCompleteData()
	return filterComplete(prefix, libraryItems)
}

func memberSuggestions(typeHint, receiver, prefix string) []completeItem {
	ensureCompleteData()
	if typeHint == "module" {
		names := stdlib.BuiltinAttrNames(receiver)
		if len(names) == 0 {
			return nil
		}
		return filterComplete(prefix, toMethodItems(names, "מודול"))
	}
	var base []completeItem
	if list, ok := methodCache[typeHint]; ok {
		base = list
	} else {
		// טיפוס לא ידוע — איחוד מתודות (רק כשאין השמה מזוהה)
		base = allMethods
	}
	return filterComplete(prefix, base)
}

// utf16Len — אורך מחרוזת באינדקסי RichEdit (UTF-16)
func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func bytePosFromUTF16(s string, utf16Idx int) int {
	if utf16Idx <= 0 {
		return 0
	}
	n := 0
	for i, r := range s {
		if n >= utf16Idx {
			return i
		}
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return len(s)
}

func utf16PosFromByte(s string, bytePos int) int {
	if bytePos <= 0 {
		return 0
	}
	if bytePos > len(s) {
		bytePos = len(s)
	}
	return utf16Len(s[:bytePos])
}

func skipBidiBack(runes []rune, i int) int {
	for i >= 0 && isBidiMark(runes[i]) {
		i--
	}
	return i
}

func skipSpaceBidiBack(runes []rune, i int) int {
	for i >= 0 && (unicode.IsSpace(runes[i]) || isBidiMark(runes[i])) {
		i--
	}
	return i
}

// tryAnalyzeInclude — כלול "קידומת
func tryAnalyzeInclude(text string, runes []rune, end int, prefix string, prefixStartByte int) (completeQuery, bool) {
	j := skipBidiBack(runes, end-1)
	// אם יש קידומת — j מצביע לתו האחרון שלה; לדלג אחורה עליה
	if prefix != "" {
		for j >= 0 {
			r := runes[j]
			if isBidiMark(r) {
				j--
				continue
			}
			if !isIdentRune(r) {
				break
			}
			j--
		}
		j = skipBidiBack(runes, j)
	}
	if j < 0 || (runes[j] != '"' && runes[j] != '`') {
		return completeQuery{}, false
	}
	k := skipSpaceBidiBack(runes, j-1)
	kw := []rune("כלול")
	if k+1 < len(kw) {
		return completeQuery{}, false
	}
	start := k - len(kw) + 1
	if start < 0 {
		return completeQuery{}, false
	}
	got := normalizeHebrew(string(runes[start : k+1]))
	if got != "כלול" {
		return completeQuery{}, false
	}
	if start > 0 && isIdentRune(runes[start-1]) {
		return completeQuery{}, false
	}
	return completeQuery{
		Mode:        modeInclude,
		Prefix:      prefix,
		ReplaceFrom: utf16PosFromByte(text, prefixStartByte),
	}, true
}

func resolveBuiltinModule(name string) string {
	n := normalizeHebrew(name)
	for _, b := range stdlib.BuiltinNames() {
		if normalizeHebrew(b) == n || strings.EqualFold(b, name) {
			return b
		}
	}
	switch strings.ToLower(name) {
	case "json":
		return "JSON"
	case "sql":
		return "SQL"
	}
	return ""
}

// analyzeCompletion מנתח את הטקסט סביב הסמן
func analyzeCompletion(text string, caretUTF16 int) completeQuery {
	bytePos := bytePosFromUTF16(text, caretUTF16)
	if bytePos > len(text) {
		bytePos = len(text)
	}

	runes := []rune(text[:bytePos])
	// דלג אחורה על סימני בידי בסוף (לפני הסמן)
	end := len(runes)
	for end > 0 && isBidiMark(runes[end-1]) {
		end--
	}
	i := end - 1
	for i >= 0 {
		r := runes[i]
		if isBidiMark(r) {
			i-- // דלג — לא שובר מזהה
			continue
		}
		if !isIdentRune(r) {
			break
		}
		i--
	}
	// i מצביע על התו שלפני תחילת המזהה (או -1)
	rawPrefix := string(runes[i+1 : end])
	prefix := normalizeHebrew(rawPrefix)

	// תחילת ההחלפה ב־UTF-16: אחרי התו שאינו מזהה
	prefixStartByte := len(string(runes[:i+1]))

	// כלול "…
	if q, ok := tryAnalyzeInclude(text, runes, end, prefix, prefixStartByte); ok {
		return q
	}

	// דלג אחורה מעל בידי כדי לבדוק נקודה למצב מתודה
	j := skipBidiBack(runes, i)
	if j >= 0 && runes[j] == '.' {
		recvEnd := j
		k := j - 1
		for k >= 0 {
			r := runes[k]
			if isBidiMark(r) {
				k--
				continue
			}
			if !isIdentRune(r) {
				break
			}
			k--
		}
		recv := normalizeHebrew(string(runes[k+1 : recvEnd]))
		hint := inferTypeHint(text, recv)
		if hint == "any" {
			if mod := resolveBuiltinModule(recv); mod != "" {
				hint = "module"
				recv = mod
			}
		}
		return completeQuery{
			Mode:        modeMember,
			Prefix:      prefix,
			ReplaceFrom: utf16PosFromByte(text, prefixStartByte),
			Receiver:    recv,
			TypeHint:    hint,
		}
	}

	if prefix == "" {
		return completeQuery{Mode: modeNone}
	}
	return completeQuery{
		Mode:        modeIdent,
		Prefix:      prefix,
		ReplaceFrom: utf16PosFromByte(text, prefixStartByte),
	}
}

func splitSourceLines(src string) []string {
	return strings.FieldsFunc(src, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

func trimLineNoise(line string) string {
	line = strings.TrimSpace(line)
	for len(line) > 0 {
		r := []rune(line)[0]
		if !isBidiMark(r) {
			break
		}
		line = string([]rune(line)[1:])
		line = strings.TrimSpace(line)
	}
	return line
}

// parseAssignment — מזהה «משתנה שם = …» או «שם = …»
func parseAssignment(line string) (name, rhs string, ok bool) {
	line = trimLineNoise(stripBidiMarks(line))
	if line == "" {
		return "", "", false
	}
	runes := []rune(line)
	i := 0
	kw := []rune("משתנה")
	if len(runes) >= len(kw) && normalizeHebrew(string(runes[:len(kw)])) == "משתנה" {
		if len(runes) == len(kw) || !isIdentRune(runes[len(kw)]) {
			i = len(kw)
			for i < len(runes) && (unicode.IsSpace(runes[i]) || isBidiMark(runes[i])) {
				i++
			}
		}
	}
	start := i
	for i < len(runes) {
		r := runes[i]
		if isBidiMark(r) {
			i++
			continue
		}
		if !isIdentRune(r) {
			break
		}
		i++
	}
	if i == start {
		return "", "", false
	}
	name = normalizeHebrew(string(runes[start:i]))
	if name == "" {
		return "", "", false
	}
	for i < len(runes) && (unicode.IsSpace(runes[i]) || isBidiMark(runes[i])) {
		i++
	}
	if i >= len(runes) || runes[i] != '=' {
		return "", "", false
	}
	if i+1 < len(runes) && runes[i+1] == '=' {
		return "", "", false // ==
	}
	i++
	for i < len(runes) && (unicode.IsSpace(runes[i]) || isBidiMark(runes[i])) {
		i++
	}
	rhs = strings.TrimSpace(string(runes[i:]))
	if idx := strings.Index(rhs, "//"); idx >= 0 {
		rhs = strings.TrimSpace(rhs[:idx])
	}
	return name, rhs, true
}

func bareIdent(rhs string) (string, bool) {
	rhs = strings.TrimSpace(rhs)
	if rhs == "" {
		return "", false
	}
	runes := []rune(rhs)
	i := 0
	for i < len(runes) && isIdentRune(runes[i]) {
		i++
	}
	if i == 0 || i != len(runes) {
		return "", false
	}
	return normalizeHebrew(rhs), true
}

// inferTypeHint — ניחוש טיפוס לפי השמות האחרונות למשתנה
func inferTypeHint(src, name string) string {
	name = normalizeHebrew(name)
	if name == "" {
		return "any"
	}
	// RichEdit מכניס סימני בידי בין אותיות — בלי ניקוי השמות נחתכים (מ→מספר)
	src = stripBidiMarks(src)
	types := map[string]string{}
	for _, line := range splitSourceLines(src) {
		n, rhs, ok := parseAssignment(line)
		if !ok {
			continue
		}
		t := guessValueType(rhs)
		if t == "any" {
			if id, ok := bareIdent(rhs); ok {
				if tt, ok2 := types[id]; ok2 {
					t = tt
				}
			}
		}
		types[n] = t
	}
	if t, ok := types[name]; ok {
		return t
	}
	return "any"
}

func guessValueType(rhs string) string {
	rhs = strings.TrimSpace(rhs)
	if rhs == "" {
		return "any"
	}
	for len(rhs) > 0 {
		r := []rune(rhs)[0]
		if !isBidiMark(r) {
			break
		}
		rhs = string([]rune(rhs)[1:])
	}
	if rhs == "" {
		return "any"
	}
	switch {
	case strings.HasPrefix(rhs, "["):
		return "array"
	case strings.HasPrefix(rhs, "{"):
		return "hash"
	case strings.HasPrefix(rhs, "\"") || strings.HasPrefix(rhs, "`"):
		return "string"
	case rhs == "אמת" || rhs == "שקר" || strings.HasPrefix(rhs, "לא "):
		return "bool"
	case rhs == "ריק":
		return "null"
	case (rhs[0] >= '0' && rhs[0] <= '9') || rhs[0] == '-' || rhs[0] == '+':
		return "number"
	case strings.HasPrefix(rhs, "טווח(") || strings.HasPrefix(rhs, "טווח:"):
		return "array"
	case strings.HasPrefix(rhs, "קלט(") || strings.HasPrefix(rhs, "קלט:"):
		return "string"
	case strings.HasPrefix(rhs, "למחרוזת(") || strings.HasPrefix(rhs, "למחרוזת:"):
		return "string"
	case strings.HasPrefix(rhs, "למספר(") || strings.HasPrefix(rhs, "למספר:"):
		return "number"
	default:
		return "any"
	}
}

func buildSuggestions(q completeQuery) []completeItem {
	switch q.Mode {
	case modeIdent:
		items := identSuggestions(q.Prefix)
		if len(items) == 0 {
			return nil
		}
		if len(items) == 1 && normalizeHebrew(items[0].Text) == normalizeHebrew(q.Prefix) {
			return nil
		}
		return items
	case modeMember:
		items := memberSuggestions(q.TypeHint, q.Receiver, q.Prefix)
		if len(items) == 0 {
			return nil
		}
		if len(items) == 1 && normalizeHebrew(items[0].Text) == normalizeHebrew(q.Prefix) {
			return nil
		}
		return items
	case modeInclude:
		items := librarySuggestions(q.Prefix)
		if len(items) == 0 {
			return nil
		}
		if len(items) == 1 && normalizeHebrew(items[0].Text) == normalizeHebrew(q.Prefix) {
			return nil
		}
		return items
	default:
		return nil
	}
}
