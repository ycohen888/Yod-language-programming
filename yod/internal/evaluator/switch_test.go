package evaluator

import (
	"strings"
	"testing"
)

func TestSwitch(t *testing.T) {
	_, out := evalSource(t, `
בחר 2
  מקרה 1
    הדפס("א");
  מקרה 2, 3
    הדפס("ב");
  ברירת_מחדל
    הדפס("ג");
סוף
`)
	if !strings.Contains(out, "ב") {
		t.Fatalf("switch: %q", out)
	}
}

func TestNullCoalesceAndPower(t *testing.T) {
	_, out := evalSource(t, `
הדפס(ריק ?? "כן");
הדפס("יש" ?? "לא");
הדפס(2 ** 8);
`)
	if !strings.Contains(out, "כן") || !strings.Contains(out, "יש") {
		t.Fatalf("??: %q", out)
	}
	if !strings.Contains(out, "256") {
		t.Fatalf("**: %q", out)
	}
}

func TestArrayTakeChunkShift(t *testing.T) {
	_, out := evalSource(t, `
משתנה ר = [1, 2, 3, 4];
הדפס(ר.לקח(2).הצטרף(","));
הדפס(ר.חלק(2).גודל());
הדפס(ר.הסר_ראשון());
הדפס(ר.הצטרף(","));
`)
	if !strings.Contains(out, "1,2") {
		t.Fatalf("לקח: %q", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("חלק/הסר: %q", out)
	}
}

func TestSystemModule(t *testing.T) {
	_, out := evalSource(t, `
כלול "מערכת";
הדפס(מערכת.ארגומנטים().גודל());
הדפס(מערכת.תיקייה().אורך() > 0);
`)
	if !strings.Contains(out, "0") {
		t.Fatalf("ארגומנטים: %q", out)
	}
	if !strings.Contains(out, "אמת") {
		t.Fatalf("תיקייה: %q", out)
	}
}
