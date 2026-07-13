package evaluator

import (
	"strings"
	"testing"
)

func TestStringMethods(t *testing.T) {
	_, out := evalSource(t, `
משתנה ט = "  שלום עולם  ";
הדפס(ט.גזום().אורך());
הדפס("abc".אותיות_גדולות());
הדפס("Hello".מתחיל_ב("He"));
הדפס("א-ב-ג".פצל("-").הצטרף("|"));
הדפס("42".למספר() + 1);
`)
	if !strings.Contains(out, "9") {
		t.Fatalf("אורך אחרי גזום: %q", out)
	}
	if !strings.Contains(out, "ABC") {
		t.Fatalf("אותיות_גדולות: %q", out)
	}
	if !strings.Contains(out, "אמת") {
		t.Fatalf("מתחיל_ב: %q", out)
	}
	if !strings.Contains(out, "א|ב|ג") {
		t.Fatalf("פצל/הצטרף: %q", out)
	}
	if !strings.Contains(out, "43") {
		t.Fatalf("למספר: %q", out)
	}
}

func TestNumberMethods(t *testing.T) {
	_, out := evalSource(t, `
הדפס((-3.7).מוחלט());
הדפס(4.2.עיגול());
הדפס(8.זוגי());
הדפס(2.חזקה(3));
הדפס(5.בין(1, 10));
`)
	if !strings.Contains(out, "3.7") {
		t.Fatalf("מוחלט: %q", out)
	}
	if !strings.Contains(out, "4") {
		t.Fatalf("עיגול: %q", out)
	}
	if !strings.Contains(out, "אמת") {
		t.Fatalf("זוגי/בין: %q", out)
	}
	if !strings.Contains(out, "8") {
		t.Fatalf("חזקה: %q", out)
	}
}

func TestBooleanAndNullMethods(t *testing.T) {
	_, out := evalSource(t, `
הדפס(אמת.לא());
הדפס(שקר.למספר());
הדפס(ריק.ריק());
הדפס(ריק.למחרוזת());
`)
	if !strings.Contains(out, "שקר") {
		t.Fatalf("לא: %q", out)
	}
	if !strings.Contains(out, "0") {
		t.Fatalf("למספר שקר: %q", out)
	}
	if !strings.Contains(out, "אמת") {
		t.Fatalf("ריק.ריק: %q", out)
	}
	if !strings.Contains(out, "ריק") {
		t.Fatalf("למחרוזת: %q", out)
	}
}
