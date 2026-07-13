package evaluator

import (
	"strings"
	"testing"
)

func TestLogicalAndOr(t *testing.T) {
	_, out := evalSource(t, `
הדפס(אמת וגם שקר);
הדפס(שקר או אמת);
הדפס(לא שקר);
הדפס(אמת && אמת);
הדפס(שקר || אמת);
`)
	if !strings.Contains(out, "שקר") || !strings.Contains(out, "אמת") {
		t.Fatalf("logic: %q", out)
	}
}

func TestShortCircuit(t *testing.T) {
	_, out := evalSource(t, `
משתנה נקרא = שקר;
פונקציה סמן
  נקרא = אמת;
  החזר אמת;
סוף
הדפס(שקר וגם סמן());
הדפס(נקרא);
הדפס(אמת או סמן());
הדפס(נקרא);
`)
	// first: וגם short-circuits → נקרא still שקר
	// second: או short-circuits → נקרא still שקר
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected 4 lines: %q", out)
	}
	if !strings.Contains(lines[0], "שקר") {
		t.Fatalf("וגם result: %q", lines[0])
	}
	if !strings.Contains(lines[1], "שקר") {
		t.Fatalf("should not call סמן: %q", lines[1])
	}
	if !strings.Contains(lines[2], "אמת") {
		t.Fatalf("או result: %q", lines[2])
	}
	if !strings.Contains(lines[3], "שקר") {
		t.Fatalf("או should not call סמן: %q", lines[3])
	}
}

func TestCompoundAssign(t *testing.T) {
	_, out := evalSource(t, `
משתנה א = 10;
א += 5;
א *= 2;
הדפס(א);
`)
	if !strings.Contains(out, "30") {
		t.Fatalf("compound: %q", out)
	}
}

func TestReduceEverySome(t *testing.T) {
	_, out := evalSource(t, `
משתנה ר = [1, 2, 3];
הדפס(ר.צמצם(0, פונקציה(ס, x) { החזר ס + x; }));
הדפס(ר.כל(פונקציה(x) { החזר x > 0; }));
הדפס(ר.יש_כל(פונקציה(x) { החזר x == 2; }));
הדפס(ר.מצא_אינדקס(פונקציה(x) { החזר x == 3; }));
`)
	if !strings.Contains(out, "6") {
		t.Fatalf("צמצם: %q", out)
	}
	if !strings.Contains(out, "אמת") {
		t.Fatalf("כל/יש_כל: %q", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("מצא_אינדקס: %q", out)
	}
}

func TestHashMapFilter(t *testing.T) {
	_, out := evalSource(t, `
משתנה מ = { "א": 1, "ב": 10 };
הדפס(מ.סנן(פונקציה(v) { החזר v > 5; }).גודל());
הדפס(מ.מפה(פונקציה(v) { החזר v * 2; }).א);
`)
	if !strings.Contains(out, "1") {
		t.Fatalf("סנן גודל: %q", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("מפה: %q", out)
	}
}

func TestMathModule(t *testing.T) {
	_, out := evalSource(t, `
כלול "מתמטיקה";
הדפס(מתמטיקה.מקסימום(1, 9, 3));
הדפס(מתמטיקה.עיגול(מתמטיקה.פי));
`)
	if !strings.Contains(out, "9") {
		t.Fatalf("מקסימום: %q", out)
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("פי עיגול: %q", out)
	}
}
