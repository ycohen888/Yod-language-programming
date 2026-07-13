package evaluator

import (
	"strings"
	"testing"
)

func TestArrayMapFilter(t *testing.T) {
	_, out := evalSource(t, `
משתנה ר = [1, 2, 3, 4];
הדפס(ר.מפה(פונקציה(x) { החזר x * 10; }).הצטרף(","));
הדפס(ר.סנן(פונקציה(x) { החזר x > 2; }).הצטרף(","));
הדפס(ר.מצא(פונקציה(x) { החזר x == 3; }));
`)
	if !strings.Contains(out, "10,20,30,40") {
		t.Fatalf("מפה: %q", out)
	}
	if !strings.Contains(out, "3,4") {
		t.Fatalf("סנן: %q", out)
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("מצא: %q", out)
	}
}

func TestRangeAndSortNumbers(t *testing.T) {
	_, out := evalSource(t, `
משתנה ר = [10, 2, 30];
ר.מיין;
הדפס(ר.הצטרף(","));
הדפס(טווח(3).הצטרף(","));
הדפס(טווח(2, 5).הצטרף(","));
`)
	if !strings.Contains(out, "2,10,30") {
		t.Fatalf("מיין מספרים: %q", out)
	}
	if !strings.Contains(out, "0,1,2") {
		t.Fatalf("טווח(3): %q", out)
	}
	if !strings.Contains(out, "2,3,4") {
		t.Fatalf("טווח(2,5): %q", out)
	}
}

func TestStringFormatAndRegex(t *testing.T) {
	_, out := evalSource(t, `
הדפס("שלום {0}".פורמט("עולם"));
הדפס("a1b2".התאם("\\d"));
הדפס("קובץ.יוד".סיומת());
`)
	if !strings.Contains(out, "שלום עולם") {
		t.Fatalf("פורמט: %q", out)
	}
	if !strings.Contains(out, "אמת") {
		t.Fatalf("התאם: %q", out)
	}
	if !strings.Contains(out, ".יוד") {
		t.Fatalf("סיומת: %q", out)
	}
}

func TestJSONAndUnique(t *testing.T) {
	_, out := evalSource(t, `
כלול "JSON";
משתנה מ = JSON.פרסר("{\"א\":1}");
הדפס(מ.א);
הדפס(JSON.מחרוזת([1, 2]));
הדפס([1, 1, 2].ייחודי().הצטרף(","));
`)
	if !strings.Contains(out, "1") {
		t.Fatalf("JSON/ייחודי: %q", out)
	}
	if !strings.Contains(out, "[1,2]") && !strings.Contains(out, "[1, 2]") {
		// JSON.Marshal typically produces [1,2]
		if !strings.Contains(out, "1,2") {
			t.Fatalf("JSON מחרוזת: %q", out)
		}
	}
}

func TestForInString(t *testing.T) {
	_, out := evalSource(t, `
עבור ת בתוך "אב"
  הדפס(ת);
סוף
`)
	if !strings.Contains(out, "א") || !strings.Contains(out, "ב") {
		t.Fatalf("עבור על מחרוזת: %q", out)
	}
}
