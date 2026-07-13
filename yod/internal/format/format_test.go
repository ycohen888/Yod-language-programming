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
	if !strings.Contains(got, "כל_עוד א <= 3") {
		t.Fatalf("while header:\n%s", got)
	}
	if !strings.Contains(got, "אם א > 3") {
		t.Fatalf("if header:\n%s", got)
	}
	if strings.Contains(got, "כל_עוד (") || strings.Contains(got, "אם (") {
		t.Fatalf("should not wrap conditions:\n%s", got)
	}
	if !strings.Contains(got, "הדפס(") {
		t.Fatalf("expected call:\n%s", got)
	}
	if !strings.Contains(got, "סוף") {
		t.Fatalf("expected סוף:\n%s", got)
	}
}

func TestFormatMirroredBraces(t *testing.T) {
	in := `אם א > 10 } הדפס("גדול"); {`
	got := Source(in)
	if !strings.Contains(got, "אם א > 10") {
		t.Fatalf("expected normalized:\n%s", got)
	}
	if !strings.Contains(got, "סוף") {
		t.Fatalf("expected סוף:\n%s", got)
	}
}

func TestFormatFunction(t *testing.T) {
	in := `פונקציה כפל(x,y){החזר x*y}`
	got := Source(in)
	want := "פונקציה כפל x, y\n    החזר x * y\nסוף\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
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
	if !strings.Contains(got, "עבור מפתח בתוך אדם") {
		t.Fatalf("for header:\n%s", got)
	}
	if !strings.Contains(got, "מחלקה כלב מרחיב חיה") {
		t.Fatalf("extends:\n%s", got)
	}
	if strings.Contains(got, "כלב;") || strings.Contains(got, "בנאי;") {
		t.Fatalf("bad semicolon:\n%s", got)
	}
	if !strings.Contains(got, "פונקציה בנאי שם") {
		t.Fatalf("bare params:\n%s", got)
	}
}

func TestFormatReturnHash(t *testing.T) {
	in := `פונקציה f(){החזר {"קוד":200}}`
	got := Source(in)
	if !strings.Contains(got, "החזר") || !strings.Contains(got, `"קוד"`) || !strings.Contains(got, "}") {
		t.Fatalf("return hash:\n%s", got)
	}
	if strings.Count(got, "סוף") < 1 {
		t.Fatalf("expected סוף:\n%s", got)
	}
}
