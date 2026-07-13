package highlight

import (
	"strings"
	"testing"
)

func TestSpansColors(t *testing.T) {
	src := `// הערה
פונקציה שלום(שם) {
  משתנה א = 5
  הדפס("שלום")
}
`
	spans := Spans(src)
	if len(spans) < 5 {
		t.Fatalf("expected spans, got %d", len(spans))
	}
	kinds := map[Kind]bool{}
	for _, s := range spans {
		kinds[s.Kind] = true
	}
	if !kinds[KindComment] {
		t.Fatal("missing comment")
	}
	if !kinds[KindKeyword] {
		t.Fatal("missing keyword")
	}
	if !kinds[KindFunction] {
		t.Fatal("missing function name")
	}
	if !kinds[KindString] {
		t.Fatal("missing string")
	}
	if !kinds[KindNumber] {
		t.Fatal("missing number")
	}
}

func TestSpansRichEditCRLF(t *testing.T) {
	// אחרי שורות עם CRLF האינדקסים חייבים להישאר מסונכרנים ל־RichEdit
	src := "משתנה א = 1\r\nמשתנה ב = 2\r\nהדפס(א)\r\n"
	plain := Spans(strings.ReplaceAll(src, "\r\n", "\n"))
	rich := SpansRichEdit(src)
	if len(plain) != len(rich) {
		t.Fatalf("span count plain=%d rich=%d", len(plain), len(rich))
	}
	// ההפרש מצטבר: אחרי N שורות CRLF, RichEdit קטן ב־N
	if rich[0].Start != plain[0].Start {
		t.Fatalf("first span start should match, plain=%d rich=%d", plain[0].Start, rich[0].Start)
	}
	// טוקן בשורה השלישית — הדפס — צריך להיות מוזז ב־2 לעומת LF-only אם ספרנו CRLF כ־2
	var lastPlain, lastRich Span
	for _, s := range plain {
		if s.Kind == KindFunction {
			lastPlain = s
		}
	}
	for _, s := range rich {
		if s.Kind == KindFunction {
			lastRich = s
		}
	}
	if lastPlain.End <= lastPlain.Start || lastRich.End <= lastRich.Start {
		t.Fatal("missing function span")
	}
	// בטקסט עם \n בלבד אין הזזה; ב־RichEdit עם CRLF=1 מול UTF-16 עם CRLF=2:
	// Spans על "\n" vs SpansRichEdit על "\r\n" — אותו מספר שורות → אותם אינדקסים יחסיים
	if lastRich.End-lastRich.Start != lastPlain.End-lastPlain.Start {
		t.Fatalf("function length mismatch")
	}
	// וידוא שספירת CRLF=1 באמת קטנה מ־CRLF=2 על אותו מקור
	asUTF16 := Spans(src) // CRLF=2
	if len(asUTF16) == 0 || len(rich) == 0 {
		t.Fatal("empty")
	}
	var f16, fRich Span
	for _, s := range asUTF16 {
		if s.Kind == KindFunction {
			f16 = s
		}
	}
	for _, s := range rich {
		if s.Kind == KindFunction {
			fRich = s
		}
	}
	// שני CRLF לפני הדפס → הפרש 2
	if f16.Start-fRich.Start != 2 {
		t.Fatalf("expected RichEdit index drift of 2, utf16=%d rich=%d", f16.Start, fRich.Start)
	}
}

func TestRichEditIndex(t *testing.T) {
	src := "אב\r\nגד"
	// rune index 4 = 'ג' (א=0 ב=1 \r=2 \n=3 ג=4)
	got := RichEditIndex(src, 4)
	if got != 3 { // א ב CRLF ג → 0,1,2,3
		t.Fatalf("RichEditIndex= %d want 3", got)
	}
}
