package evaluator

import (
	"strings"
	"testing"
)

func TestHashCollectionMethods(t *testing.T) {
	_, out := evalSource(t, `
משתנה אדם = { "שם": "נועם", "גיל": 30 };
אדם.הדפס_שמות;
אדם.הדפס_הכל;
הדפס(אדם.יש("שם"));
הדפס(אדם.גודל());
`)
	if !strings.Contains(out, "שם") || !strings.Contains(out, "גיל") {
		t.Fatalf("keys missing: %q", out)
	}
	if !strings.Contains(out, "שם: נועם") {
		t.Fatalf("הדפס_הכל: %q", out)
	}
	if !strings.Contains(out, "אמת") || !strings.Contains(out, "2") {
		t.Fatalf("יש/גודל: %q", out)
	}
}

func TestArrayCollectionMethods(t *testing.T) {
	_, out := evalSource(t, `
משתנה ר = ["ב", "א"];
ר.הוסף("ג");
הדפס(ר.הצטרף("-"));
הדפס(ר.מכיל("א"));
הדפס(ר.גודל());
`)
	if !strings.Contains(out, "ב-א-ג") {
		t.Fatalf("הצטרף: %q", out)
	}
	if !strings.Contains(out, "אמת") || !strings.Contains(out, "3") {
		t.Fatalf("מכיל/גודל: %q", out)
	}
}
