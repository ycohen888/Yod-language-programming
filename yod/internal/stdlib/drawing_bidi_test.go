package stdlib

import (
	"math"
	"testing"

	"golang.org/x/image/font/basicfont"

	"yod/internal/object"
)

func TestCaretOffsetEnglishInRTL(t *testing.T) {
	face := basicfont.Face7x13
	text := "Hi"
	full := caretOffsetFromTextLeft(face, text, 2, true)
	mid := caretOffsetFromTextLeft(face, text, 1, true)
	start := caretOffsetFromTextLeft(face, text, 0, true)
	wH := measureFaceAdvance(face, "H")
	wHi := measureFaceAdvance(face, "Hi")

	if math.Abs(start) > 0.01 {
		t.Fatalf("start wants 0 (left of H), got %v", start)
	}
	if math.Abs(mid-wH) > 0.01 {
		t.Fatalf("after H wants %v, got %v", wH, mid)
	}
	if math.Abs(full-wHi) > 0.01 {
		t.Fatalf("end wants %v, got %v", wHi, full)
	}
}

func TestCaretOffsetDigitsInRTL(t *testing.T) {
	face := basicfont.Face7x13
	text := "12"
	start := caretOffsetFromTextLeft(face, text, 0, true)
	mid := caretOffsetFromTextLeft(face, text, 1, true)
	end := caretOffsetFromTextLeft(face, text, 2, true)
	w1 := measureFaceAdvance(face, "1")
	w12 := measureFaceAdvance(face, "12")
	if math.Abs(start) > 0.01 || math.Abs(mid-w1) > 0.01 || math.Abs(end-w12) > 0.01 {
		t.Fatalf("digits RTL: start=%v mid=%v end=%v (want 0, %v, %v)", start, mid, end, w1, w12)
	}
}

func TestCaretOffsetHebrewRTL(t *testing.T) {
	face := basicfont.Face7x13
	text := "אב"
	fullW := measureFaceAdvance(face, visualOrderForAlign(text, 1))
	start := caretOffsetFromTextLeft(face, text, 0, true)
	end := caretOffsetFromTextLeft(face, text, 2, true)

	if math.Abs(start-fullW) > 0.5 {
		t.Fatalf("RTL start wants fullW=%v (right), got %v", fullW, start)
	}
	if end > 0.5 {
		t.Fatalf("RTL end wants ~0 (left), got %v", end)
	}
}

func TestCaretInsetEnglishRTLAlign(t *testing.T) {
	st := &drawBoard{
		align:  1,
		fontSz: 13,
		face:   basicfont.Face7x13,
	}
	text := "Hello"
	n := boardTextCaretInset(st, &object.String{Value: text}, &object.Number{Value: 5})
	num, ok := n.(*object.Number)
	if !ok {
		t.Fatalf("expected number, got %v", n)
	}
	if math.Abs(num.Value) > 0.01 {
		t.Fatalf("RTL inset at end for LTR text wants 0, got %v", num.Value)
	}
	n0 := boardTextCaretInset(st, &object.String{Value: text}, &object.Number{Value: 0})
	num0, ok := n0.(*object.Number)
	if !ok {
		t.Fatalf("expected number at 0, got %v", n0)
	}
	full, _, err := measureBoardText(st, text)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(num0.Value-full) > 0.01 {
		t.Fatalf("RTL inset at start wants fullW=%v, got %v", full, num0.Value)
	}
}

func TestCaretOffsetLTRMode(t *testing.T) {
	face := basicfont.Face7x13
	text := "Hi"
	if v := caretOffsetFromTextLeft(face, text, 0, false); math.Abs(v) > 0.01 {
		t.Fatalf("LTR start: %v", v)
	}
	if v := caretOffsetFromTextLeft(face, text, 2, false); math.Abs(v-measureFaceAdvance(face, text)) > 0.01 {
		t.Fatalf("LTR end: %v", v)
	}
}
