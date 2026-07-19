package stdlib

import (
	"testing"

	"yod/internal/object"
)

func TestParseTextOverlayHash(t *testing.T) {
	h := object.NewHash()
	h.Set("טקסט", &object.String{Value: "שלום עולם"})
	h.Set("גודל", &object.Number{Value: 48})
	h.Set("צבע", &object.String{Value: "לבן"})
	h.Set("מיקום", &object.String{Value: "למטה_מרכז"})
	h.Set("רקע", &object.String{Value: "#000000"})
	h.Set("שקיפות_רקע", &object.Number{Value: 0.5})
	h.Set("צל", &object.Boolean{Value: true})
	h.Set("התחלה", &object.Number{Value: 1})
	h.Set("סיום", &object.Number{Value: 8})

	o, err := parseTextOverlayHash(h)
	if err != nil {
		t.Fatal(err)
	}
	if o.Text != "שלום עולם" || o.Size != 48 || o.Color != "#ffffff" {
		t.Fatalf("unexpected overlay: %+v", o)
	}
	if !o.Box || o.BoxOpacity != 0.5 || o.Start != 1 || o.End != 8 {
		t.Fatalf("box/time: %+v", o)
	}
	x, y := overlayXYExpr(o)
	if x != "(w-text_w)/2" || y != "h-text_h-40" {
		t.Fatalf("xy expr: %s %s", x, y)
	}
}

func TestParseTextOverlaysEmptyFails(t *testing.T) {
	_, err := parseTextOverlays(&object.Array{})
	if err == nil {
		t.Fatal("expected error for empty list")
	}
}

func TestNormalizeOverlayColor(t *testing.T) {
	if normalizeOverlayColor("אדום") != "#ef4444" {
		t.Fatal("hebrew red")
	}
	if normalizeOverlayColor("#abc") != "#abc" {
		t.Fatal("hex passthrough")
	}
}
