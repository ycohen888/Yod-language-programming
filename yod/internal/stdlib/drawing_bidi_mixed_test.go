package stdlib

import (
	"math"
	"strings"
	"testing"

	"golang.org/x/image/font/basicfont"
	"golang.org/x/text/unicode/bidi"

	"yod/internal/object"
)

func TestVisualMapMixedWordLike(t *testing.T) {
	cases := []struct {
		in   string
		want string // visual LTR
	}{
		{"שלום", "םולש"},
		{"Hi", "Hi"},
		{"שלום Hi", "Hi םולש"},   // רווח נשאר עם ריצת העברית → "Hi"+" םולש"
		{"שלום 12", "12 םולש"},
		{"Hi שלום", "םולש Hi"}, // אחרי היפוך ריצות
	}
	for _, c := range cases {
		disp, m, fromRTL := buildVisualMap(c.in, true)
		if len([]rune(disp)) != len(m) || len(m) != len(fromRTL) {
			t.Fatalf("%q: len mismatch", c.in)
		}
		t.Logf("%q → %q (want ~%q)", c.in, disp, c.want)
		if disp != c.want {
			// רווחים בריצה יכולים לזוז קלות — משווים בלי רווחים כפולים
			if compact(disp) != compact(c.want) {
				t.Fatalf("%q: visual %q want %q", c.in, disp, c.want)
			}
		}
	}
}

func compact(s string) string {
	return strings.ReplaceAll(s, "  ", " ")
}

func TestCaretMixedInRange(t *testing.T) {
	st := &drawBoard{align: 1, fontSz: 13, face: basicfont.Face7x13}
	text := "שלום Hi 12"
	rs := []rune(text)
	full, _, _ := measureBoardText(st, text)
	for i := 0; i <= len(rs); i++ {
		out := boardTextCaretInset(st, &object.String{Value: text}, &object.Number{Value: float64(i)})
		n, ok := out.(*object.Number)
		if !ok {
			t.Fatalf("caret %d: %v", i, out)
		}
		if n.Value < -0.01 || n.Value > full+0.01 {
			t.Fatalf("caret %d inset %v out of [0,%v]", i, n.Value, full)
		}
		t.Logf("i=%d inset=%.2f", i, n.Value)
	}
}

func TestCaretHebrewStillWordLike(t *testing.T) {
	st := &drawBoard{align: 1, fontSz: 13, face: basicfont.Face7x13}
	text := "שלום"
	full, _, _ := measureBoardText(st, text)
	n0 := boardTextCaretInset(st, &object.String{Value: text}, &object.Number{Value: 0}).(*object.Number)
	if math.Abs(n0.Value) > 0.5 {
		t.Fatalf("heb start inset want ~0 got %v", n0.Value)
	}
	n4 := boardTextCaretInset(st, &object.String{Value: text}, &object.Number{Value: 4}).(*object.Number)
	if math.Abs(n4.Value-full) > 0.5 {
		t.Fatalf("heb end inset want ~full=%v got %v", full, n4.Value)
	}
}

func TestCaretEnglishEndAtRight(t *testing.T) {
	st := &drawBoard{align: 1, fontSz: 13, face: basicfont.Face7x13}
	text := "Hi"
	n2 := boardTextCaretInset(st, &object.String{Value: text}, &object.Number{Value: 2}).(*object.Number)
	if math.Abs(n2.Value) > 0.01 {
		t.Fatalf("English end inset want 0 (right edge) got %v", n2.Value)
	}
}

func TestBidiDump(t *testing.T) {
	s := "שלום Hi 12"
	var p bidi.Paragraph
	_, _ = p.SetString(s, bidi.DefaultDirection(bidi.RightToLeft))
	ord, err := p.Order()
	if err != nil {
		t.Fatal(err)
	}
	disp, _, _ := buildVisualMap(s, true)
	t.Logf("visual=%q", disp)
	for i := 0; i < ord.NumRuns(); i++ {
		r := ord.Run(i)
		a, b := r.Pos()
		t.Logf("run %d %d..%d dir=%v %q", i, a, b, r.Direction(), r.String())
	}
}

func indexOfLog(mapping []int, log int) int {
	for i, v := range mapping {
		if v == log {
			return i
		}
	}
	return -1
}
