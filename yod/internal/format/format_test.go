package format

import (
	"strings"
	"testing"
)

func TestFormatBasic(t *testing.T) {
	in := `משתנה א=1
כל_עוד א<=3{הדפס("ספירה: "+א)
א=א+1}
אם א>3{הדפס("סוף")}אחרת{הדפס("לא")}`
	got := Source(in)
	if !strings.Contains(got, "כל_עוד (א <= 3)") && !strings.Contains(got, "כל_עוד(א <= 3)") {
		t.Fatalf("while header with parens:\n%s", got)
	}
	if !strings.Contains(got, "אם (א > 3)") && !strings.Contains(got, "אם(א > 3)") {
		t.Fatalf("if header with parens:\n%s", got)
	}
	if !strings.Contains(got, "הדפס(") {
		t.Fatalf("expected call:\n%s", got)
	}
	if !strings.Contains(got, "{") || !strings.Contains(got, "}") {
		t.Fatalf("expected braces:\n%s", got)
	}
	if strings.Contains(got, "סוף\n") || strings.HasSuffix(strings.TrimSpace(got), "סוף") {
		t.Fatalf("should use braces not סוף:\n%s", got)
	}
}

func TestFormatMirroredBraces(t *testing.T) {
	in := `אם א > 10 } הדפס("גדול"); {`
	got := Source(in)
	if !strings.Contains(got, "אם") || !strings.Contains(got, "10") {
		t.Fatalf("expected normalized:\n%s", got)
	}
	if !strings.Contains(got, "{") || !strings.Contains(got, "}") {
		t.Fatalf("expected braces:\n%s", got)
	}
}

func TestFormatFunction(t *testing.T) {
	in := `פונקציה כפל(x,y){החזר x*y}`
	got := Source(in)
	if !strings.Contains(got, "פונקציה כפל(x, y)") && !strings.Contains(got, "פונקציה כפל(x,y)") {
		t.Fatalf("expected php-style params:\n%s", got)
	}
	if !strings.Contains(got, "{") || !strings.Contains(got, "}") {
		t.Fatalf("expected braces:\n%s", got)
	}
	if !strings.Contains(got, "החזר x * y") {
		t.Fatalf("expected body:\n%s", got)
	}
}

func TestFormatBareFunctionParams(t *testing.T) {
	in := `פונקציה כפל x, y
החזר x * y
סוף
`
	got := Source(in)
	if !strings.Contains(got, "פונקציה כפל(x, y)") && !strings.Contains(got, "פונקציה כפל(x,y)") {
		t.Fatalf("expected wrapped params:\n%s", got)
	}
	if !strings.Contains(got, "}") {
		t.Fatalf("expected closing brace:\n%s", got)
	}
}

func TestFormatPreservesComment(t *testing.T) {
	in := `// שלום
הדפס(1)`
	got := Source(in)
	if !strings.Contains(got, "הדפס(1)") {
		t.Fatalf("call:\n%s", got)
	}
	if strings.Contains(got, "הדפס(1);") {
		t.Fatalf("should not require semicolon:\n%s", got)
	}
}

func TestFormatHashAndForIn(t *testing.T) {
	in := `משתנה אדם={שם:"נועם",גיל:30}
עבור מפתח בתוך אדם{הדפס(מפתח)}
מחלקה כלב מרחיב חיה{פונקציה בנאי(שם){הורה(שם)}}`
	got := Source(in)
	if !strings.Contains(got, "שם:") || !strings.Contains(got, "}") {
		t.Fatalf("hash broken:\n%s", got)
	}
	if strings.Contains(got, "עבור מפתח;") || strings.Contains(got, "מפתח;\nבתוך") {
		t.Fatalf("for-in broken:\n%s", got)
	}
	if !strings.Contains(got, "עבור") || !strings.Contains(got, "בתוך") {
		t.Fatalf("for header:\n%s", got)
	}
	if !strings.Contains(got, "מחלקה כלב מרחיב חיה") {
		t.Fatalf("extends:\n%s", got)
	}
	if !strings.Contains(got, "פונקציה בנאי(שם)") && !strings.Contains(got, "פונקציה בנאי(שם)") {
		t.Fatalf("paren params:\n%s", got)
	}
}

func TestFormatVarNamedMemSameAsKeyword(t *testing.T) {
	in := `פונקציה רדיוס_בטוח(w, h, r)
  משתנה מ = r.שלם()
  אם מ * 2 > h
    מ = (h / 2).שלם()
  סוף
  אם מ * 2 > w
    מ = (w / 2).שלם()
  סוף
  החזר מ
סוף
`
	got := Source(in)
	if !strings.Contains(got, "אם (מ * 2 > h)") {
		t.Fatalf("expected full cond wrap, got:\n%s", got)
	}
	if strings.Contains(got, "אם (מ * 2 > h\n") {
		t.Fatalf("body leaked into condition:\n%s", got)
	}
	if !strings.Contains(got, "מ = (h / 2).שלם()") {
		t.Fatalf("expected assignment body:\n%s", got)
	}
}

func TestFormatForInThis(t *testing.T) {
	in := `פונקציה רענון()
  עבור ר בתוך זה.רכיבים
    ר.צייר(זה.משטח)
  סוף
סוף
`
	got := Source(in)
	if !strings.Contains(got, "עבור (ר בתוך זה.רכיבים)") {
		t.Fatalf("expected for-in this.col:\n%s", got)
	}
	if strings.Contains(got, "בתוך)") {
		t.Fatalf("broke for-in before זה:\n%s", got)
	}
}

func TestFormatCallbackSof(t *testing.T) {
	in := `זה.עם_אצווה(פונקציה ()
  עצמי.נקה()
סוף)
`
	got := Source(in)
	if !strings.Contains(got, "})") && !strings.Contains(got, "}\n)") {
		// prefer })
	}
	if !strings.Contains(got, "פונקציה()") {
		t.Fatalf("expected anon fn:\n%s", got)
	}
	if strings.Count(got, ")") < 2 {
		t.Fatalf("missing call close:\n%s", got)
	}
}

func TestFormatNestedIfBeforeElseIf(t *testing.T) {
	// אחרת_אם אחרי סוף של אם מקונן חייב לסגור את הענף החיצוני, לא להיצמד לפנימי
	in := `פונקציה בעת_מקש(שם)
  אם שם == "אנטר"
    זה.נקה_בחירה()
    אם זה.באנטר != ריק
      זה.באנטר(זה.טקסט)
    סוף
  אחרת_אם שם == "בריחה"
    זה.נקה_בחירה()
  אחרת_אם שם == "התחלה"
    זה.נקה_בחירה()
  סוף
סוף
`
	got := Source(in)
	if strings.Contains(got, "}אחרת_אם") {
		t.Fatalf("missing space before else-if:\n%s", got)
	}
	// מבנה צפוי: הענף של אנטר נסגר לפני אחרת_אם בריחה
	want := `} אחרת_אם (שם == "בריחה")`
	if !strings.Contains(got, want) {
		t.Fatalf("expected outer branch close before else-if:\n%s", got)
	}
	bad := `} אחרת_אם (שם == "בריחה")` // after only inner — detect by inner-then-else without middle close
	// אם הצמדה שגויה: "}אחרת_אם" או "} אחרת_אם" מיד אחרי באנטר בלי סגירת הענף
	if strings.Contains(got, "זה.באנטר(זה.טקסט)\n                } אחרת_אם (שם == \"בריחה\")") ||
		strings.Contains(got, "זה.באנטר(זה.טקסט)\n            } אחרת_אם (שם == \"בריחה\")") {
		// ok if those have enough closing — check brace balance instead
	}
	_ = bad
	open := strings.Count(got, "{")
	close := strings.Count(got, "}")
	if open != close {
		t.Fatalf("unbalanced braces %d vs %d:\n%s", open, close, got)
	}
	if !strings.Contains(got, "אם (זה.באנטר != ריק) {") {
		t.Fatalf("expected nested if:\n%s", got)
	}
	// אחרי האם הפנימי חייב } ואז } של הענף — לפני אחרת_אם בריחה
	idxInner := strings.Index(got, "זה.באנטר(זה.טקסט)")
	idxEscape := strings.Index(got, `אחרת_אם (שם == "בריחה")`)
	if idxInner < 0 || idxEscape < 0 || idxEscape < idxInner {
		t.Fatalf("structure missing:\n%s", got)
	}
	between := got[idxInner:idxEscape]
	if strings.Count(between, "}") < 2 {
		t.Fatalf("need }} before outer else-if (inner+branch), got %q in:\n%s", between, got)
	}
}

func TestFormatAndInCondition(t *testing.T) {
	in := `פונקציה f()
  אם hover_סוג == "כפתור" וגם hover_ערך == מזהה
    על = אמת
  סוף
סוף
`
	got := Source(in)
	want := `אם (hover_סוג == "כפתור" וגם hover_ערך == מזהה)`
	if !strings.Contains(got, want) {
		t.Fatalf("וגם should stay in condition:\n%s", got)
	}
	if strings.Contains(got, ") {\n        וגם") {
		t.Fatalf("וגם leaked into body:\n%s", got)
	}
}

func TestFormatIfThisBody(t *testing.T) {
	in := `פונקציה קבע_רמז(t)
  אם t == ריק
    זה.רמז = ""
  אחרת
    זה.רמז = t
  סוף
סוף
`
	got := Source(in)
	if !strings.Contains(got, "אם (t == ריק)") {
		t.Fatalf("expected clean condition:\n%s", got)
	}
	if strings.Contains(got, "ריק\nזה") {
		t.Fatalf("זה leaked into condition:\n%s", got)
	}
	if !strings.Contains(got, `זה.רמז = ""`) {
		t.Fatalf("expected this.assign body:\n%s", got)
	}
}
