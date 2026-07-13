//go:build windows

package editor

import "testing"
import "strings"

func TestAnalyzeIdentPrefix(t *testing.T) {
	for _, src := range []string{"משת", "משתנ", "משתן", "פונ", "פון", "פונק"} {
		q := analyzeCompletion(src, utf16Len(src))
		if q.Mode != modeIdent {
			t.Fatalf("%q: mode %d", src, q.Mode)
		}
		items := buildSuggestions(q)
		want := "משתנה"
		if strings.HasPrefix(normalizeHebrew(src), "פונ") {
			want = "פונקציה"
		}
		found := false
		for _, it := range items {
			if it.Text == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("%q: expected %s in %#v (prefix=%q)", src, want, items, q.Prefix)
		}
	}
}

func TestCRLFIndexMatchesCaret(t *testing.T) {
	// מדמה טקסט RichEdit אחרי נרמול \r\n → \r
	src := "שורה ראשונה\rפונ"
	q := analyzeCompletion(src, utf16Len(src))
	if q.Prefix != "פונ" && normalizeHebrew(q.Prefix) != "פונ" {
		t.Fatalf("prefix=%q", q.Prefix)
	}
	items := buildSuggestions(q)
	found := false
	for _, it := range items {
		if it.Text == "פונקציה" {
			found = true
		}
	}
	if !found {
		t.Fatalf("items=%#v", items)
	}
}

func TestAnalyzeWithBidiMarks(t *testing.T) {
	// סימני RTL בין אותיות — כמו ש־RichEdit לפעמים מכניס
	src := "פ\u200Fו\u200Fנ"
	q := analyzeCompletion(src, utf16Len(src))
	items := buildSuggestions(q)
	found := false
	for _, it := range items {
		if it.Text == "פונקציה" {
			found = true
		}
	}
	if !found {
		t.Fatalf("bidi prefix %q mode=%d items=%#v", q.Prefix, q.Mode, items)
	}
}

func TestAllKeywordsCompletable(t *testing.T) {
	ensureCompleteData()
	for _, kw := range cachedKeywords {
		if kw == "" {
			continue
		}
		// קידומת של תו אחד לפחות
		r := []rune(kw)
		prefix := string(r[:1])
		items := identSuggestions(prefix)
		found := false
		for _, it := range items {
			if it.Text == kw {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("keyword %q not found for prefix %q", kw, prefix)
		}
		// קידומת כמעט מלאה
		if len(r) >= 2 {
			prefix2 := string(r[:len(r)-1])
			items2 := identSuggestions(prefix2)
			found2 := false
			for _, it := range items2 {
				if it.Text == kw {
					found2 = true
					break
				}
			}
			if !found2 {
				t.Fatalf("keyword %q not found for prefix %q", kw, prefix2)
			}
		}
	}
}

func TestAnalyzeMember(t *testing.T) {
	src := "משתנה שמות = [\"א\"]\nשמות."
	q := analyzeCompletion(src, utf16Len(src))
	if q.Mode != modeMember {
		t.Fatalf("mode %d", q.Mode)
	}
	if q.Receiver != "שמות" {
		t.Fatalf("receiver %q", q.Receiver)
	}
	if q.TypeHint != "array" {
		t.Fatalf("hint %q", q.TypeHint)
	}
	items := buildSuggestions(q)
	if len(items) < 5 {
		t.Fatalf("expected array methods, got %d", len(items))
	}
	found := false
	for _, it := range items {
		if it.Text == "גודל" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing גודל")
	}
}

func TestAnalyzeMemberPrefix(t *testing.T) {
	src := "שמות.גו"
	q := analyzeCompletion(src, utf16Len(src))
	items := buildSuggestions(q)
	if len(items) == 0 {
		t.Fatal("expected filtered methods")
	}
	for _, it := range items {
		if it.Text != "גודל" && it.Text != "גזום" {
			// גזום is string-only; with any hint both may appear
			if len(it.Text) < 2 || it.Text[:2] != "גו" {
				t.Fatalf("unexpected %q", it.Text)
			}
		}
	}
}

func TestInferTypeHint(t *testing.T) {
	src := "משתנה שם = \"נועם\"\nמשתנה ר = [1, 2]\nמשתנה מ = {א: 1}\n"
	if g := inferTypeHint(src, "שם"); g != "string" {
		t.Fatalf("string: %q", g)
	}
	if g := inferTypeHint(src, "ר"); g != "array" {
		t.Fatalf("array: %q", g)
	}
	if g := inferTypeHint(src, "מ"); g != "hash" {
		t.Fatalf("hash: %q", g)
	}
}

func TestInferTypeHintCRAndNumber(t *testing.T) {
	src := "משתנה טקסט = \"אאא\"\rמשתנה מספר = 1\r"
	if g := inferTypeHint(src, "טקסט"); g != "string" {
		t.Fatalf("string: %q", g)
	}
	if g := inferTypeHint(src, "מספר"); g != "number" {
		t.Fatalf("number: %q", g)
	}
}

func TestInferTypeHintExactName(t *testing.T) {
	src := "משתנה טקסט = \"א\"\nמשתנה טקסט2 = [1]\n"
	if g := inferTypeHint(src, "טקסט"); g != "string" {
		t.Fatalf("טקסט: %q", g)
	}
	if g := inferTypeHint(src, "טקסט2"); g != "array" {
		t.Fatalf("טקסט2: %q", g)
	}
}

func TestInferTypeHintCopy(t *testing.T) {
	src := "משתנה א = \"שלום\"\nמשתנה ב = א\n"
	if g := inferTypeHint(src, "ב"); g != "string" {
		t.Fatalf("copy: %q", g)
	}
}

func TestAnalyzeInclude(t *testing.T) {
	src := `כלול "`
	q := analyzeCompletion(src, utf16Len(src))
	if q.Mode != modeInclude {
		t.Fatalf("mode %d", q.Mode)
	}
	items := buildSuggestions(q)
	found := map[string]bool{}
	for _, it := range items {
		found[it.Text] = true
		if it.Kind != kindLibrary {
			t.Fatalf("kind of %q: %v", it.Text, it.Kind)
		}
	}
	for _, want := range []string{"בסיס", "קבצים", "JSON", "זמן", "מתמטיקה", "מערכת", "רשת", "SQL", "חלונות"} {
		if !found[want] {
			t.Fatalf("missing library %q in %#v", want, items)
		}
	}
}

func TestAnalyzeIncludePrefix(t *testing.T) {
	src := `כלול "קב`
	q := analyzeCompletion(src, utf16Len(src))
	if q.Mode != modeInclude {
		t.Fatalf("mode %d", q.Mode)
	}
	items := buildSuggestions(q)
	if len(items) != 1 || items[0].Text != "קבצים" {
		t.Fatalf("items=%#v", items)
	}
}

func TestAnalyzeStringMember(t *testing.T) {
	src := "משתנה טקסט = \"אאא\"\nטקסט."
	q := analyzeCompletion(src, utf16Len(src))
	if q.Mode != modeMember || q.TypeHint != "string" {
		t.Fatalf("mode=%d hint=%q", q.Mode, q.TypeHint)
	}
	items := buildSuggestions(q)
	found := false
	for _, it := range items {
		if it.Text == "אורך" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing אורך in %#v", items)
	}
}

func TestAnalyzeNumberMember(t *testing.T) {
	src := "משתנה מספר = 1\nמספר."
	q := analyzeCompletion(src, utf16Len(src))
	if q.TypeHint != "number" {
		t.Fatalf("hint %q", q.TypeHint)
	}
	items := buildSuggestions(q)
	found := false
	for _, it := range items {
		if it.Text == "למחרוזת" {
			found = true
		}
		if it.Text == "אורך" {
			t.Fatalf("אורך must not appear for number, got %#v", items)
		}
	}
	if !found {
		t.Fatalf("missing למחרוזת")
	}
}

func TestAnalyzeNumberMemberWithBidi(t *testing.T) {
	// מדמה RichEdit שמכניס RLM בין אותיות עבריות
	rlm := "\u200F"
	src := "משתנה מ" + rlm + "ס" + rlm + "פ" + rlm + "ר = 5\rמ" + rlm + "ס" + rlm + "פ" + rlm + "ר."
	q := analyzeCompletion(src, utf16Len(src))
	if q.Mode != modeMember {
		t.Fatalf("mode %d", q.Mode)
	}
	if q.TypeHint != "number" {
		t.Fatalf("hint %q receiver %q (bidi should not break inference)", q.TypeHint, q.Receiver)
	}
	items := buildSuggestions(q)
	for _, it := range items {
		if it.Text == "אורך" {
			t.Fatalf("אורך leaked for number under bidi: %#v", items)
		}
	}
}

func TestUserExampleStringVsNumber(t *testing.T) {
	src := "משתנה טקסט = \"טקסט\"\rמשתנה מספר = 5\r\rהדפס: טקסט.אורך\rהדפס: מספר."
	q := analyzeCompletion(src, utf16Len(src))
	if q.TypeHint != "number" {
		t.Fatalf("מספר. hint=%q recv=%q", q.TypeHint, q.Receiver)
	}
	items := buildSuggestions(q)
	for _, it := range items {
		if it.Text == "אורך" {
			t.Fatalf("אורך on מספר: %#v", items)
		}
	}
}

func TestAnalyzeModuleMember(t *testing.T) {
	src := "JSON."
	q := analyzeCompletion(src, utf16Len(src))
	if q.Mode != modeMember || q.TypeHint != "module" {
		t.Fatalf("mode=%d hint=%q recv=%q", q.Mode, q.TypeHint, q.Receiver)
	}
	items := buildSuggestions(q)
	found := false
	for _, it := range items {
		if it.Text == "פרסר" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing פרסר in %#v", items)
	}
}

func TestLeadingIndentSpan(t *testing.T) {
	cases := []struct {
		line    string
		wantOff int
		wantN   int
	}{
		{"הדפס", 0, 0},
		{"  הדפס", 0, 2},
		{" הדפס", 0, 1},
		{"\tהדפס", 0, 1},
		{"    הדפס", 0, 2},
		{"\u200F  הדפס", 1, 2},
	}
	for _, c := range cases {
		off, n := leadingIndentSpan(c.line, 0)
		if off != c.wantOff || n != c.wantN {
			t.Fatalf("%q: got off=%d n=%d want off=%d n=%d", c.line, off, n, c.wantOff, c.wantN)
		}
	}
}

func TestIndentCharsBefore(t *testing.T) {
	src := "  הדפס"
	// סמן אחרי שני רווחים
	n := indentCharsBefore(src, 2, 0)
	if n != 2 {
		t.Fatalf("after spaces: n=%d", n)
	}
	// סמן באמצע המילה — אין מה למחוק לפניו
	n = indentCharsBefore(src, utf16Len(src), 0)
	if n != 0 {
		t.Fatalf("at end of word: n=%d", n)
	}
	// רווחים שהוזנו ליד סמן באמצע (טאב ישן)
	mid := "הד  פס"
	// caret after two spaces: "הד  |פס"
	caret := utf16Len("הד  ")
	n = indentCharsBefore(mid, caret, 0)
	if n != 2 {
		t.Fatalf("mid-line spaces: n=%d caret=%d", n, caret)
	}
}
