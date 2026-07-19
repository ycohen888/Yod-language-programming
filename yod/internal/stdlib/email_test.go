package stdlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yod/internal/object"
)

func TestBuildMIMEMessageUTF8AndAttach(t *testing.T) {
	dir := t.TempDir()
	attach := filepath.Join(dir, "דוח.txt")
	if err := os.WriteFile(attach, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	msg, err := buildMIMEMessageForTest(
		"from@example.com",
		[]string{"to@example.com"},
		"נושא בעברית",
		"תוכן שלום",
		[]string{attach},
	)
	if err != nil {
		t.Fatal(err)
	}
	s := string(msg)
	if !strings.Contains(s, "Subject:") {
		t.Fatal("חסר Subject")
	}
	if !strings.Contains(s, "utf-8") && !strings.Contains(s, "UTF-8") {
		t.Fatal("חסר UTF-8 בכותרות/גוף")
	}
	if !strings.Contains(s, "דוח.txt") {
		t.Fatal("חסר שם קובץ מצורף")
	}
	if !strings.Contains(s, "תוכן שלום") {
		t.Fatal("חסר גוף ההודעה")
	}
}

func TestGmailDefaults(t *testing.T) {
	h := object.NewHash()
	h.Set("משתמש", &object.String{Value: "me@gmail.com"})
	h.Set("סיסמה", &object.String{Value: "abcd efgh ijkl mnop"})
	h.Set("אל", &object.String{Value: "you@example.com"})
	h.Set("נושא", &object.String{Value: "בדיקה"})
	out := emailApplyGmailDefaults(h)
	got, ok := out.(*object.Hash)
	if !ok {
		t.Fatalf("%T %v", out, out)
	}
	if hashStr(got, "שרת") != "smtp.gmail.com" {
		t.Fatal(hashStr(got, "שרת"))
	}
	if hashStr(got, "מאת") != "me@gmail.com" {
		t.Fatal(hashStr(got, "מאת"))
	}
	if hashStr(got, "סיסמה") != "abcdefghijklmnop" {
		t.Fatalf("סיסמה עם רווחים לא נוקתה: %q", hashStr(got, "סיסמה"))
	}
	if hashInt(got, "פורט", 0) != 587 {
		t.Fatal(hashInt(got, "פורט", 0))
	}
}

func TestBuildMIMEMessageHTML(t *testing.T) {
	msg, err := buildMIMEMessage(
		"from@example.com",
		[]string{"to@example.com"},
		"נושא",
		"טקסט",
		"<b>HTML</b>",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	s := string(msg)
	if !strings.Contains(s, "text/html") {
		t.Fatal("חסר text/html")
	}
	if !strings.Contains(s, "<b>HTML</b>") {
		t.Fatal("חסר גוף HTML")
	}
	if !strings.Contains(s, "multipart/alternative") {
		t.Fatal("חסר multipart/alternative")
	}
}
