package stdlib

import (
	"testing"
)

func TestParsePOBasic(t *testing.T) {
	src := `
msgid ""
msgstr ""
"Language: en\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

msgid "שמור"
msgstr "Save"

msgctxt "זכר"
msgid "ברוך הבא"
msgstr "Welcome (m)"

msgctxt "נקבה"
msgid "ברוך הבא"
msgstr "Welcome (f)"

msgid "קובץ אחד"
msgid_plural "%d קבצים"
msgstr[0] "One file"
msgstr[1] "%d files"
`
	msgs, header, err := parsePOFile([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if header == "" {
		t.Fatal("expected Plural-Forms header")
	}
	if m := msgs[i18nKey("", "שמור")]; m == nil || m.Strs[0] != "Save" {
		t.Fatalf("שמור: %+v", m)
	}
	if m := msgs[i18nKey("זכר", "ברוך הבא")]; m == nil || m.Strs[0] != "Welcome (m)" {
		t.Fatalf("זכר: %+v", m)
	}
	if m := msgs[i18nKey("נקבה", "ברוך הבא")]; m == nil || m.Strs[0] != "Welcome (f)" {
		t.Fatalf("נקבה: %+v", m)
	}
	if m := msgs[i18nKey("", "קובץ אחד")]; m == nil || len(m.Strs) < 2 {
		t.Fatalf("plural: %+v", m)
	}
}

func TestI18nLookupPluralGender(t *testing.T) {
	i18nMu.Lock()
	i18nCatalogs = map[string]*i18nCatalog{}
	i18nLang = "en"
	i18nGender = ""
	i18nMu.Unlock()

	po := `
msgid ""
msgstr ""
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

msgid "קובץ אחד"
msgid_plural "%d קבצים"
msgstr[0] "One file"
msgstr[1] "%d files"

msgctxt "זכר"
msgid "שלום {שם}"
msgstr "Hello Mr {שם}"

msgctxt "נקבה"
msgid "שלום {שם}"
msgstr "Hello Ms {שם}"
`
	if err := i18nLoadFileInto("en", ""); err == nil {
		// load via register from parsed
	}
	msgs, header, err := parsePOFile([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	i18nMu.Lock()
	i18nCatalogs["en"] = &i18nCatalog{
		msgs:   msgs,
		header: header,
		plural: pluralFuncFromHeader(header, "en"),
	}
	i18nLang = "en"
	i18nMu.Unlock()

	got := i18nLookup("", "קובץ אחד", 1, nil)
	if got != "One file" {
		t.Fatalf("n=1: %q", got)
	}
	got = i18nLookup("", "קובץ אחד", 5, map[string]string{"כמות": "5"})
	if got != "5 files" {
		t.Fatalf("n=5: %q", got)
	}
	got = i18nLookup("זכר", "שלום {שם}", -1, map[string]string{"שם": "דן"})
	if got != "Hello Mr דן" {
		t.Fatalf("gender m: %q", got)
	}
	got = i18nLookup("נקבה", "שלום {שם}", -1, map[string]string{"שם": "דנה"})
	if got != "Hello Ms דנה" {
		t.Fatalf("gender f: %q", got)
	}

	if I18nDirection() != "ltr" {
		t.Fatalf("dir for en: %s", I18nDirection())
	}
	i18nMu.Lock()
	i18nLang = "he"
	i18nMu.Unlock()
	if I18nDirection() != "rtl" {
		t.Fatalf("dir for he: %s", I18nDirection())
	}
}

func TestI18nFallbackToKey(t *testing.T) {
	i18nMu.Lock()
	i18nCatalogs = map[string]*i18nCatalog{}
	i18nLang = "fr"
	i18nMu.Unlock()
	got := i18nLookup("", "טקסט לא קיים", -1, nil)
	if got != "טקסט לא קיים" {
		t.Fatalf("fallback: %q", got)
	}
}

func TestI18nHebrewPlurals(t *testing.T) {
	po := `
msgid ""
msgstr ""
"Plural-Forms: nplurals=3; plural=(n==1 ? 0 : n==2 ? 1 : 2);\n"

msgid "קובץ אחד"
msgid_plural "%d קבצים"
msgstr[0] "קובץ אחד"
msgstr[1] "שני קבצים"
msgstr[2] "%d קבצים"
`
	msgs, header, err := parsePOFile([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	i18nMu.Lock()
	i18nCatalogs = map[string]*i18nCatalog{
		"he": {msgs: msgs, header: header, plural: pluralFuncFromHeader(header, "he")},
	}
	i18nLang = "he"
	i18nMu.Unlock()

	if g := i18nLookup("", "קובץ אחד", 1, nil); g != "קובץ אחד" {
		t.Fatalf("n=1: %q", g)
	}
	if g := i18nLookup("", "קובץ אחד", 2, nil); g != "שני קבצים" {
		t.Fatalf("n=2: %q", g)
	}
	if g := i18nLookup("", "קובץ אחד", 7, nil); g != "7 קבצים" {
		t.Fatalf("n=7: %q", g)
	}
}

func TestI18nNormalizeLang(t *testing.T) {
	if normalizeLangCode("עברית") != "he" {
		t.Fatalf("עברית → %s", normalizeLangCode("עברית"))
	}
	if normalizeLangCode("en_US") != "en_us" {
		t.Fatalf("en_US → %s", normalizeLangCode("en_US"))
	}
}

func TestI18nIsRTL(t *testing.T) {
	if i18nDirForLang("he") != "rtl" {
		t.Fatal("he should be rtl")
	}
	if i18nDirForLang("ar") != "rtl" {
		t.Fatal("ar should be rtl")
	}
	if i18nDirForLang("en") != "ltr" {
		t.Fatal("en should be ltr")
	}
	if i18nDirForLang("fr") != "ltr" {
		t.Fatal("fr should be ltr")
	}
	if i18nDirForLang("עברית") != "rtl" {
		t.Fatal("עברית should normalize to rtl")
	}
}
