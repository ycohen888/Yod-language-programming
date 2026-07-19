package stdlib

import (
	"path/filepath"
	"testing"

	"yod/internal/object"
)

func TestSpreadsheetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	mod := NewSpreadsheetModule()
	created := mod.Attrs["חדש"].(*object.Builtin).Fn()
	handle, ok := created.(*object.Module)
	if !ok {
		t.Fatalf("חדש: %T %v", created, created)
	}

	if err := handle.Attrs["קבע_תא"].(*object.Builtin).Fn(
		&object.String{Value: "A1"},
		&object.String{Value: "שלום"},
	); isYodErr(err) {
		t.Fatal(err)
	}
	if err := handle.Attrs["קבע_תא"].(*object.Builtin).Fn(
		&object.String{Value: "B1"},
		&object.Number{Value: 42},
	); isYodErr(err) {
		t.Fatal(err)
	}
	if err := handle.Attrs["שמור_כ"].(*object.Builtin).Fn(&object.String{Value: path}); isYodErr(err) {
		t.Fatal(err)
	}
	if err := handle.Attrs["סגור"].(*object.Builtin).Fn(); isYodErr(err) {
		t.Fatal(err)
	}

	openedObj := mod.Attrs["פתח"].(*object.Builtin).Fn(&object.String{Value: path})
	opened, ok := openedObj.(*object.Module)
	if !ok {
		t.Fatalf("פתח: %T %v", openedObj, openedObj)
	}
	namesObj := opened.Attrs["שמות_גיליונות"].(*object.Builtin).Fn()
	names, ok := namesObj.(*object.Array)
	if !ok || len(names.Elements) < 1 {
		t.Fatalf("שמות_גיליונות: %#v", namesObj)
	}
	a1 := opened.Attrs["קרא_תא"].(*object.Builtin).Fn(&object.String{Value: "A1"})
	s, ok := a1.(*object.String)
	if !ok || s.Value != "שלום" {
		t.Fatalf("A1 = %#v", a1)
	}
	b1 := opened.Attrs["קרא_תא"].(*object.Builtin).Fn(&object.String{Value: "B1"})
	n, ok := b1.(*object.Number)
	if !ok || n.Value != 42 {
		t.Fatalf("B1 = %#v", b1)
	}
	_ = opened.Attrs["סגור"].(*object.Builtin).Fn()
}

func TestSpreadsheetStyleAndChart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chart.xlsx")
	mod := NewSpreadsheetModule()
	created := mod.Attrs["חדש"].(*object.Builtin).Fn()
	handle := created.(*object.Module)
	set := handle.Attrs["קבע_תא"].(*object.Builtin).Fn
	_ = set(&object.String{Value: "A1"}, &object.String{Value: "חודש"})
	_ = set(&object.String{Value: "B1"}, &object.String{Value: "מכירות"})
	_ = set(&object.String{Value: "A2"}, &object.String{Value: "א"})
	_ = set(&object.String{Value: "B2"}, &object.Number{Value: 10})
	_ = set(&object.String{Value: "A3"}, &object.String{Value: "ב"})
	_ = set(&object.String{Value: "B3"}, &object.Number{Value: 20})

	style := object.NewHash()
	style.Set("מודגש", &object.Boolean{Value: true})
	style.Set("רקע", &object.String{Value: "#1F4E79"})
	style.Set("צבע", &object.String{Value: "#FFFFFF"})
	if err := handle.Attrs["קבע_סגנון"].(*object.Builtin).Fn(&object.String{Value: "A1"}, style); isYodErr(err) {
		t.Fatal(err)
	}
	ch := object.NewHash()
	ch.Set("סוג", &object.String{Value: "עמודות"})
	ch.Set("נתונים", &object.String{Value: "A1:B3"})
	ch.Set("מיקום", &object.String{Value: "E2"})
	ch.Set("כותרת", &object.String{Value: "בדיקה"})
	if err := handle.Attrs["גרף"].(*object.Builtin).Fn(ch); isYodErr(err) {
		t.Fatal(err)
	}
	if err := handle.Attrs["שמור_כ"].(*object.Builtin).Fn(&object.String{Value: path}); isYodErr(err) {
		t.Fatal(err)
	}
	_ = handle.Attrs["סגור"].(*object.Builtin).Fn()
}

func isYodErr(o object.Object) bool {
	_, ok := o.(*object.Error)
	return ok
}
