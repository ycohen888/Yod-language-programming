//go:build windows

package editor

import "testing"

func TestToggleCommentLines(t *testing.T) {
	in := []string{"  הדפס: 1", "  הדפס: 2"}
	out := toggleCommentLines(in)
	if out[0] != "  // הדפס: 1" || out[1] != "  // הדפס: 2" {
		t.Fatalf("comment: %#v", out)
	}
	out2 := toggleCommentLines(out)
	if out2[0] != "  הדפס: 1" || out2[1] != "  הדפס: 2" {
		t.Fatalf("uncomment: %#v", out2)
	}
}

func TestLineSuggestsBlockIndent(t *testing.T) {
	if !lineSuggestsBlockIndent("אם אמת") {
		t.Fatal("אם")
	}
	if !lineSuggestsBlockIndent("פונקציה שלום()") {
		t.Fatal("פונקציה")
	}
	if lineSuggestsBlockIndent("הדפס: 1") {
		t.Fatal("הדפס should not indent")
	}
	if lineSuggestsBlockIndent("סוף") {
		t.Fatal("סוף should not indent")
	}
}

func TestLeadingSpacesOfLine(t *testing.T) {
	if g := leadingSpacesOfLine("    הדפס"); g != "    " {
		t.Fatalf("%q", g)
	}
	if g := leadingSpacesOfLine("\u200F  א"); g != "  " {
		t.Fatalf("bidi: %q", g)
	}
}
