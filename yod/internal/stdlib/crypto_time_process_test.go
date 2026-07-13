package stdlib

import (
	"testing"

	"yod/internal/object"
)

func TestCryptoModule(t *testing.T) {
	m := NewCryptoModule()
	b64 := m.Attrs["בסיס64"].(*object.Builtin)
	from := m.Attrs["מ_בסיס64"].(*object.Builtin)
	md5b := m.Attrs["MD5"].(*object.Builtin)
	sha := m.Attrs["SHA256"].(*object.Builtin)
	hmacB := m.Attrs["HMAC_SHA256"].(*object.Builtin)

	enc := b64.Fn(&object.String{Value: "abc"})
	s, ok := enc.(*object.String)
	if !ok || s.Value != "YWJj" {
		t.Fatalf("בסיס64: got %#v", enc)
	}
	dec := from.Fn(s)
	if ds, ok := dec.(*object.String); !ok || ds.Value != "abc" {
		t.Fatalf("מ_בסיס64: got %#v", dec)
	}
	h := md5b.Fn(&object.String{Value: "abc"})
	if hs, ok := h.(*object.String); !ok || hs.Value != "900150983cd24fb0d6963f7d28e17f72" {
		t.Fatalf("MD5: got %#v", h)
	}
	sh := sha.Fn(&object.String{Value: "abc"})
	if ss, ok := sh.(*object.String); !ok || ss.Value != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("SHA256: got %#v", sh)
	}
	hm := hmacB.Fn(&object.String{Value: "key"}, &object.String{Value: "msg"})
	if _, ok := hm.(*object.String); !ok {
		t.Fatalf("HMAC: got %#v", hm)
	}
}

func TestTimeParseAndParts(t *testing.T) {
	m := NewTimeModule()
	parse := m.Attrs["פרסר"].(*object.Builtin)
	weekday := m.Attrs["יום_בשבוע"].(*object.Builtin)
	year := m.Attrs["שנה"].(*object.Builtin)
	add := m.Attrs["הוסף"].(*object.Builtin)
	diff := m.Attrs["הפרש"].(*object.Builtin)

	ts := parse.Fn(&object.String{Value: "2026-07-13 12:00:00"})
	n, ok := ts.(*object.Number)
	if !ok {
		t.Fatalf("פרסר: %#v", ts)
	}
	wd := weekday.Fn(n)
	if w, ok := wd.(*object.String); !ok || w.Value != "שני" {
		t.Fatalf("יום_בשבוע: %#v", wd)
	}
	y := year.Fn(n)
	if yn, ok := y.(*object.Number); !ok || yn.Value != 2026 {
		t.Fatalf("שנה: %#v", y)
	}
	later := add.Fn(n, &object.Number{Value: 60})
	d := diff.Fn(later, n)
	if dn, ok := d.(*object.Number); !ok || dn.Value != 60 {
		t.Fatalf("הפרש: %#v", d)
	}
}

func TestSystemRun(t *testing.T) {
	m := NewSystemModule()
	run := m.Attrs["הפעל"].(*object.Builtin)
	res := run.Fn(&object.String{Value: "cmd"}, &object.String{Value: "/c"}, &object.String{Value: "echo ok"})
	h, ok := res.(*object.Hash)
	if !ok {
		t.Fatalf("הפעל: %#v", res)
	}
	code := h.Pairs["קוד"].(*object.Number)
	if code.Value != 0 {
		t.Fatalf("קוד: %v", code.Value)
	}
}
