package lint

import (
	"strings"
	"testing"
)

func TestBraceBlocksAllowed(t *testing.T) {
	issues := CheckSource("t.יוד", `
אם אמת {
  הדפס: 1
}
`)
	for _, iss := range issues {
		if strings.Contains(iss.Message, "סוף") || strings.Contains(iss.Message, "{ }") {
			t.Fatalf("braces should not warn: %v", iss)
		}
	}
}

func TestHashNoWarning(t *testing.T) {
	issues := CheckSource("t.יוד", `משתנה מ = { "א": 1 }
הדפס: מ
`)
	for _, iss := range issues {
		if strings.Contains(iss.Message, "סוף") {
			t.Fatalf("dict braces should not warn: %v", iss)
		}
	}
}
