package lexer

import (
	"unicode"
	"unicode/utf8"

	"yod/internal/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
	readColumn   int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0, readColumn: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = l.readPosition
		return
	}
	r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
	l.ch = r
	l.position = l.readPosition
	l.readPosition += size
	if r == '\n' {
		l.line++
		l.readColumn = 0
	} else {
		l.readColumn++
	}
	l.column = l.readColumn
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()

	tok := token.Token{Line: l.line, Column: l.column}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.Eq
			tok.Literal = "=="
		} else {
			tok.Type = token.Assign
			tok.Literal = "="
		}
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.PlusAssign
			tok.Literal = "+="
		} else {
			tok.Type = token.Plus
			tok.Literal = "+"
		}
	case '-':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.MinusAssign
			tok.Literal = "-="
		} else {
			tok.Type = token.Minus
			tok.Literal = "-"
		}
	case '*':
		if l.peekChar() == '*' {
			l.readChar()
			tok.Type = token.Power
			tok.Literal = "**"
		} else if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.AsteriskAssign
			tok.Literal = "*="
		} else {
			tok.Type = token.Asterisk
			tok.Literal = "*"
		}
	case '?':
		if l.peekChar() == '?' {
			l.readChar()
			tok.Type = token.NullCoalesce
			tok.Literal = "??"
		} else {
			tok.Type = token.Illegal
			tok.Literal = string(l.ch)
		}
	case '/':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.SlashAssign
			tok.Literal = "/="
		} else {
			tok.Type = token.Slash
			tok.Literal = "/"
		}
	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.PercentAssign
			tok.Literal = "%="
		} else {
			tok.Type = token.Percent
			tok.Literal = "%"
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.NotEq
			tok.Literal = "!="
		} else {
			tok.Type = token.Bang
			tok.Literal = "!"
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok.Type = token.And
			tok.Literal = "&&"
		} else {
			tok.Type = token.Illegal
			tok.Literal = string(l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok.Type = token.Or
			tok.Literal = "||"
		} else {
			tok.Type = token.Illegal
			tok.Literal = string(l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.LtEq
			tok.Literal = "<="
		} else {
			tok.Type = token.Lt
			tok.Literal = "<"
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = token.GtEq
			tok.Literal = ">="
		} else {
			tok.Type = token.Gt
			tok.Literal = ">"
		}
	case ',':
		tok.Type = token.Comma
		tok.Literal = ","
	case ';':
		tok.Type = token.Semicolon
		tok.Literal = ";"
	case '(':
		tok.Type = token.LParen
		tok.Literal = "("
	case ')':
		tok.Type = token.RParen
		tok.Literal = ")"
	case '{':
		tok.Type = token.LBrace
		tok.Literal = "{"
	case '}':
		tok.Type = token.RBrace
		tok.Literal = "}"
	case '[':
		tok.Type = token.LBracket
		tok.Literal = "["
	case ']':
		tok.Type = token.RBracket
		tok.Literal = "]"
	case '.':
		tok.Type = token.Dot
		tok.Literal = "."
	case ':':
		tok.Type = token.Colon
		tok.Literal = ":"
	case '"', '\'':
		quote := l.ch
		tok.Type = token.String
		tok.Literal = l.readString(quote)
		return tok
	case 0:
		tok.Type = token.EOF
		tok.Literal = ""
	default:
		if isIdentStart(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		}
		if isDigit(l.ch) {
			tok.Type = token.Number
			tok.Literal = l.readNumber()
			return tok
		}
		tok.Type = token.Illegal
		tok.Literal = string(l.ch)
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' ||
			l.ch == '\u200E' || l.ch == '\u200F' || l.ch == '\u2066' || l.ch == '\u2069' {
			l.readChar()
		}
		// הערה: // עד סוף שורה
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}
		break
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isIdentPart(l.ch) {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readNumber() string {
	start := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[start:l.position]
}

func (l *Lexer) readString(quote rune) string {
	l.readChar() // פתיחה
	start := l.position
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	lit := l.input[start:l.position]
	if l.ch == quote {
		l.readChar() // סגירה
	}
	return unescape(lit)
}

func unescape(s string) string {
	out := make([]rune, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\\' && i+size < len(s) {
			n, nsize := utf8.DecodeRuneInString(s[i+size:])
			switch n {
			case 'n':
				out = append(out, '\n')
			case 't':
				out = append(out, '\t')
			case '\\', '"', '\'':
				out = append(out, n)
			default:
				out = append(out, '\\', n)
			}
			i += size + nsize
			continue
		}
		out = append(out, r)
		i += size
	}
	return string(out)
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
