package highlight

import (
	"unicode"
	"unicode/utf8"
	"unicode/utf16"

	"yod/internal/token"
)

// צבעי Dark+ / PHP כמו ב־Cursor (theme-defaults)
const (
	ColorBackground = 0x001E1E1E // #1E1E1E
	ColorForeground = 0x00D4D4D4 // #D4D4D4
	ColorGutter     = 0x00151515
	ColorLineNum    = 0x00858585
	ColorComment    = 0x0055996A // #6A9955 → BGR
	ColorString     = 0x007891CE // #CE9178
	ColorNumber     = 0x00A8CEB5 // #B5CEA8
	ColorControl    = 0x00C086C5 // #C586C0
	ColorKeyword    = 0x00D69C56 // #569CD6
	ColorFunction   = 0x00AADCDC // #DCDCAA
	ColorVariable   = 0x00FECD9C // #9CDCFE
	ColorClass      = 0x00B0C94E // #4EC9B0
	ColorOperator   = 0x00D4D4D4 // #D4D4D4
)

// Hex CSS (RGB) לפאנל HTML
const (
	CSSBg       = "#1E1E1E"
	CSSFg       = "#D4D4D4"
	CSSComment  = "#6A9955"
	CSSString   = "#CE9178"
	CSSNumber   = "#B5CEA8"
	CSSControl  = "#C586C0"
	CSSKeyword  = "#569CD6"
	CSSFunction = "#DCDCAA"
	CSSVariable = "#9CDCFE"
	CSSClass    = "#4EC9B0"
	CSSOperator = "#D4D4D4"
	CSSLineNum  = "#858585"
)

// Kind סוג טוקן להדגשה
type Kind int

const (
	KindText Kind = iota
	KindComment
	KindString
	KindNumber
	KindControl  // אם, החזר, כל_עוד…
	KindKeyword  // פונקציה, מחלקה, משתנה, פרטי…
	KindFunction // שם פונקציה / קריאה
	KindVariable // משתנים / מזהים
	KindClass    // שמות מחלקה
	KindConstant // אמת, שקר, ריק, זה
	KindOperator
)

// Span טווח UTF-16 לצביעה ב־RichEdit
type Span struct {
	Start int
	End   int
	Kind  Kind
}

// ColorBGR מחזיר COLORREF (0x00BBGGRR) ל־Win32
func (k Kind) ColorBGR() uint32 {
	switch k {
	case KindComment:
		return ColorComment
	case KindString:
		return ColorString
	case KindNumber:
		return ColorNumber
	case KindControl:
		return ColorControl
	case KindKeyword, KindConstant:
		return ColorKeyword
	case KindFunction:
		return ColorFunction
	case KindVariable:
		return ColorVariable
	case KindClass:
		return ColorClass
	case KindOperator:
		return ColorOperator
	default:
		return ColorForeground
	}
}

func (k Kind) CSS() string {
	switch k {
	case KindComment:
		return CSSComment
	case KindString:
		return CSSString
	case KindNumber:
		return CSSNumber
	case KindControl:
		return CSSControl
	case KindKeyword, KindConstant:
		return CSSKeyword
	case KindFunction:
		return CSSFunction
	case KindVariable:
		return CSSVariable
	case KindClass:
		return CSSClass
	case KindOperator:
		return CSSOperator
	default:
		return CSSFg
	}
}

type rawTok struct {
	kind    Kind
	lit     string
	start16 int
	end16   int
	tokType token.Type
}

// Spans מנתח קוד יוד ומחזיר טווחי צבע (אינדקסי UTF-16 רגילים, CRLF=2)
func Spans(source string) []Span {
	return buildSpans(source, false)
}

// SpansRichEdit כמו Spans אבל אינדקסים ל־RichEdit (CRLF נספר כתו אחד)
func SpansRichEdit(source string) []Span {
	return buildSpans(source, true)
}

func buildSpans(source string, richEdit bool) []Span {
	raw := scan(source, richEdit)
	classify(raw)
	out := make([]Span, 0, len(raw))
	for _, t := range raw {
		if t.end16 <= t.start16 {
			continue
		}
		if t.kind == KindText {
			continue
		}
		out = append(out, Span{Start: t.start16, End: t.end16, Kind: t.kind})
	}
	return MergeSpans(out)
}

// MergeSpans מאחד טווחים סמוכים מאותו סוג — פחות קריאות צביעה ל־RichEdit
func MergeSpans(spans []Span) []Span {
	if len(spans) <= 1 {
		return spans
	}
	out := make([]Span, 0, len(spans))
	cur := spans[0]
	for i := 1; i < len(spans); i++ {
		s := spans[i]
		if s.Kind == cur.Kind && s.Start <= cur.End {
			if s.End > cur.End {
				cur.End = s.End
			}
			continue
		}
		out = append(out, cur)
		cur = s
	}
	out = append(out, cur)
	return out
}

func classify(toks []rawTok) {
	for i := range toks {
		t := &toks[i]
		if t.tokType != token.Ident {
			continue
		}
		// ברירת מחדל — משתנה (כמו $var ב־PHP)
		t.kind = KindVariable

		// אחרי מילת מפתח
		if i > 0 {
			prev := toks[i-1].tokType
			switch prev {
			case token.Function:
				t.kind = KindFunction
				continue
			case token.Class, token.Extends, token.New:
				t.kind = KindClass
				continue
			case token.Var:
				t.kind = KindVariable
				continue
			}
		}
		// קריאת פונקציה: שם(
		if nextSignificant(toks, i+1) == token.LParen {
			t.kind = KindFunction
		}
	}
}

func nextSignificant(toks []rawTok, i int) token.Type {
	for j := i; j < len(toks); j++ {
		if toks[j].kind == KindComment {
			continue
		}
		return toks[j].tokType
	}
	return token.EOF
}

func scan(input string, richEdit bool) []rawTok {
	var toks []rawTok
	i := 0
	u16 := 0
	for i < len(input) {
		// הערה //
		if i+1 < len(input) && input[i] == '/' && input[i+1] == '/' {
			start := u16
			startByte := i
			i += 2
			u16 += 2
			for i < len(input) && input[i] != '\n' && input[i] != '\r' {
				_, size := utf8.DecodeRuneInString(input[i:])
				u16 += utf16RuneLen(input[i : i+size])
				i += size
			}
			toks = append(toks, rawTok{
				kind: KindComment, lit: input[startByte:i],
				start16: start, end16: u16, tokType: token.Illegal,
			})
			continue
		}
		// הערת בלוק /* ... */
		if i+1 < len(input) && input[i] == '/' && input[i+1] == '*' {
			start := u16
			startByte := i
			i += 2
			u16 += 2
			for i+1 < len(input) {
				if input[i] == '*' && input[i+1] == '/' {
					u16 += 2
					i += 2
					break
				}
				_, size := utf8.DecodeRuneInString(input[i:])
				u16 += utf16RuneLen(input[i : i+size])
				i += size
			}
			toks = append(toks, rawTok{
				kind: KindComment, lit: input[startByte:i],
				start16: start, end16: u16, tokType: token.Illegal,
			})
			continue
		}

		r, size := utf8.DecodeRuneInString(input[i:])

		// שבירת שורה — ב־RichEdit, CRLF = תו בחירה אחד
		if r == '\r' || r == '\n' {
			if r == '\r' && i+size < len(input) && input[i+size] == '\n' {
				i += size + 1
				if richEdit {
					u16++
				} else {
					u16 += 2
				}
			} else {
				i += size
				u16++
			}
			continue
		}

		// רווח
		if r == ' ' || r == '\t' {
			u16 += utf16RuneLen(string(r))
			i += size
			continue
		}

		start16 := u16
		startByte := i

		// מחרוזת רגילה או תבנית `
		if r == '"' || r == '\'' || r == '`' {
			quote := r
			i += size
			u16 += 1
			for i < len(input) {
				cr, csz := utf8.DecodeRuneInString(input[i:])
				if cr == '\\' && i+csz < len(input) {
					u16 += utf16RuneLen(string(cr))
					i += csz
					nr, nsz := utf8.DecodeRuneInString(input[i:])
					u16 += utf16RuneLen(string(nr))
					i += nsz
					continue
				}
				u16 += utf16RuneLen(string(cr))
				i += csz
				if cr == quote {
					break
				}
			}
			toks = append(toks, rawTok{
				kind: KindString, lit: input[startByte:i],
				start16: start16, end16: u16, tokType: token.String,
			})
			continue
		}

		// מספר
		if isDigit(r) {
			for i < len(input) {
				cr, csz := utf8.DecodeRuneInString(input[i:])
				if !isDigit(cr) && !(cr == '.' && i+csz < len(input) && isDigit(rune(input[i+csz]))) {
					break
				}
				u16 += utf16RuneLen(string(cr))
				i += csz
			}
			toks = append(toks, rawTok{
				kind: KindNumber, lit: input[startByte:i],
				start16: start16, end16: u16, tokType: token.Number,
			})
			continue
		}

		// מזהה / מילת מפתח
		if isIdentStart(r) {
			for i < len(input) {
				cr, csz := utf8.DecodeRuneInString(input[i:])
				if !isIdentPart(cr) {
					break
				}
				u16 += utf16RuneLen(string(cr))
				i += csz
			}
			lit := input[startByte:i]
			tt := token.LookupIdent(lit)
			k := KindVariable
			switch tt {
			case token.If, token.Else, token.ElseIf, token.While, token.For,
				token.Return, token.Break, token.Continue, token.Try, token.Catch,
				token.Throw, token.In, token.End, token.And, token.Or,
				token.Switch, token.Case, token.Default:
				k = KindControl
			case token.Function, token.Class, token.Var, token.New, token.Extends,
				token.Private, token.Public, token.Include, token.Module, token.Export,
				token.Import, token.From, token.Enum, token.Parent, token.Not:
				k = KindKeyword
			case token.True, token.False, token.Null, token.This:
				k = KindConstant
			case token.Ident:
				k = KindVariable
			default:
				k = KindKeyword
			}
			toks = append(toks, rawTok{
				kind: k, lit: lit, start16: start16, end16: u16, tokType: tt,
			})
			continue
		}

		// אופרטורים דו־תוויים
		if i+1 < len(input) {
			two := input[i : i+2]
			var tt token.Type
			switch two {
			case "==":
				tt = token.Eq
			case "!=":
				tt = token.NotEq
			case "<=":
				tt = token.LtEq
			case ">=":
				tt = token.GtEq
			case "&&":
				tt = token.And
			case "||":
				tt = token.Or
			case "+=":
				tt = token.PlusAssign
			case "-=":
				tt = token.MinusAssign
			case "*=":
				tt = token.AsteriskAssign
			case "/=":
				tt = token.SlashAssign
			case "%=":
				tt = token.PercentAssign
			case "**":
				tt = token.Power
			case "??":
				tt = token.NullCoalesce
			case "->":
				tt = token.Arrow
			}
			if tt != "" {
				toks = append(toks, rawTok{
					kind: KindOperator, lit: two,
					start16: start16, end16: start16 + 2, tokType: tt,
				})
				i += 2
				u16 += 2
				continue
			}
		}

		// תו בודד
		tt := token.Illegal
		switch r {
		case '=':
			tt = token.Assign
		case '+':
			tt = token.Plus
		case '-':
			tt = token.Minus
		case '*':
			tt = token.Asterisk
		case '/':
			tt = token.Slash
		case '%':
			tt = token.Percent
		case '!':
			tt = token.Bang
		case '<':
			tt = token.Lt
		case '>':
			tt = token.Gt
		case ',':
			tt = token.Comma
		case ';':
			tt = token.Semicolon
		case '(':
			tt = token.LParen
		case ')':
			tt = token.RParen
		case '{':
			tt = token.LBrace
		case '}':
			tt = token.RBrace
		case '[':
			tt = token.LBracket
		case ']':
			tt = token.RBracket
		case '.':
			tt = token.Dot
		case ':':
			tt = token.Colon
		}
		n16 := utf16RuneLen(string(r))
		toks = append(toks, rawTok{
			kind: KindOperator, lit: string(r),
			start16: start16, end16: start16 + n16, tokType: tt,
		})
		i += size
		u16 += n16
	}
	return toks
}

// RichEditIndex ממיר אינדקס rune בטקסט לאינדקס תו של RichEdit (CRLF=1)
func RichEditIndex(src string, runeIndex int) int {
	if runeIndex <= 0 {
		return 0
	}
	n := 0
	i := 0
	runes := []rune(src)
	for i < len(runes) && i < runeIndex {
		r := runes[i]
		if r == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			n++
			i += 2
			continue
		}
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
		i++
	}
	return n
}

func utf16RuneLen(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_' || r == '־'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || unicode.IsDigit(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
