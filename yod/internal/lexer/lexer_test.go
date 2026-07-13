package lexer

import (
	"testing"

	"yod/internal/token"
)

func TestNextTokenBasic(t *testing.T) {
	input := `משתנה סכום = 42
הדפס("שלום")
`
	tests := []struct {
		typ     token.Type
		literal string
	}{
		{token.Var, "משתנה"},
		{token.Ident, "סכום"},
		{token.Assign, "="},
		{token.Number, "42"},
		{token.Ident, "הדפס"},
		{token.LParen, "("},
		{token.String, "שלום"},
		{token.RParen, ")"},
		{token.EOF, ""},
	}
	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.typ {
			t.Fatalf("tests[%d] type: got %q want %q (lit=%q)", i, tok.Type, tt.typ, tok.Literal)
		}
		if tok.Literal != tt.literal {
			t.Fatalf("tests[%d] literal: got %q want %q", i, tok.Literal, tt.literal)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := `אם אחרת כל_עוד עבור פונקציה מחלקה אמת שקר ריק`
	want := []token.Type{
		token.If, token.Else, token.While, token.For,
		token.Function, token.Class, token.True, token.False, token.Null, token.EOF,
	}
	l := New(input)
	for i, w := range want {
		tok := l.NextToken()
		if tok.Type != w {
			t.Fatalf("[%d] got %q want %q", i, tok.Type, w)
		}
	}
}

func TestCommentsSkipped(t *testing.T) {
	l := New("// הערה\nמשתנה א = 1")
	tok := l.NextToken()
	if tok.Type != token.Var {
		t.Fatalf("expected משתנה after comment, got %q", tok.Type)
	}
}
