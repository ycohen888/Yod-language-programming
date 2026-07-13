package format

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"yod/internal/token"
)

const indentUnit = "    " // 4 רווחים — הזחה ברורה בעברית RTL

type kind int

const (
	kEOF kind = iota
	kComment
	kIdent
	kNumber
	kString
	kOp
)

type tok struct {
	kind kind
	lit  string
	typ  token.Type // למילות מפתח / סימנים
}

// Source מסדר קוד יוד: הזחות, סוף, וקריאות בלי סוגריים מיותרים.
func Source(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	toks := scan(src)
	toks = normalizeMirroredBraces(toks)
	toks = preferHebrewShape(toks)
	return printToks(toks)
}

// preferHebrewShape — מסיר () מתנאים, עבור, ופרמטרי פונקציה
func preferHebrewShape(toks []tok) []tok {
	var out []tok
	for i := 0; i < len(toks); i++ {
		t := toks[i]

		// אם / כל_עוד / אחרת_אם (
		if isCondKeyword(t.typ) && i+1 < len(toks) && toks[i+1].typ == token.LParen {
			out = append(out, t)
			i += 2 // דלג על (
			depth := 1
			for i < len(toks) && depth > 0 {
				switch toks[i].typ {
				case token.LParen:
					depth++
					out = append(out, toks[i])
				case token.RParen:
					depth--
					if depth > 0 {
						out = append(out, toks[i])
					}
					// depth==0: דלג על ) הסוגר של התנאי
				default:
					out = append(out, toks[i])
				}
				if depth == 0 {
					break
				}
				i++
			}
			continue
		}

		// עבור (
		if t.typ == token.For && i+1 < len(toks) && toks[i+1].typ == token.LParen {
			out = append(out, t)
			i += 2
			depth := 1
			for i < len(toks) && depth > 0 {
				switch toks[i].typ {
				case token.LParen:
					depth++
					out = append(out, toks[i])
				case token.RParen:
					depth--
					if depth > 0 {
						out = append(out, toks[i])
					}
				default:
					out = append(out, toks[i])
				}
				if depth == 0 {
					break
				}
				i++
			}
			continue
		}

		// פונקציה [שם] (
		if t.typ == token.Function {
			out = append(out, t)
			j := i + 1
			if j < len(toks) && toks[j].typ == token.Ident {
				out = append(out, toks[j])
				j++
			}
			if j < len(toks) && toks[j].typ == token.LParen {
				j++ // דלג על (
				for j < len(toks) && toks[j].typ != token.RParen {
					out = append(out, toks[j])
					j++
				}
				if j < len(toks) && toks[j].typ == token.RParen {
					j++ // דלג על )
				}
				i = j - 1
				continue
			}
			i = j - 1
			continue
		}

		out = append(out, t)
	}
	return out
}

func scan(input string) []tok {
	var out []tok
	i := 0
	for i < len(input) {
		r, size := utf8.DecodeRuneInString(input[i:])

		if r == ' ' || r == '\t' || r == '\n' ||
			r == '\u200E' || r == '\u200F' || r == '\u2066' || r == '\u2069' {
			i += size
			continue
		}

		// הערה
		if r == '/' && i+1 < len(input) && input[i+1] == '/' {
			start := i
			i += 2
			for i < len(input) && input[i] != '\n' {
				_, s := utf8.DecodeRuneInString(input[i:])
				i += s
			}
			out = append(out, tok{kind: kComment, lit: strings.TrimRight(input[start:i], "\r")})
			continue
		}

		// מחרוזת
		if r == '"' || r == '\'' {
			quote := r
			start := i
			i += size
			for i < len(input) {
				cr, csz := utf8.DecodeRuneInString(input[i:])
				i += csz
				if cr == '\\' && i < len(input) {
					_, nsz := utf8.DecodeRuneInString(input[i:])
					i += nsz
					continue
				}
				if cr == quote {
					break
				}
			}
			out = append(out, tok{kind: kString, lit: input[start:i], typ: token.String})
			continue
		}

		// מספר
		if isDigit(r) {
			start := i
			i += size
			for i < len(input) {
				cr, csz := utf8.DecodeRuneInString(input[i:])
				if !isDigit(cr) && !(cr == '.' && i+csz < len(input) && isDigit(runeAt(input, i+csz))) {
					break
				}
				i += csz
			}
			out = append(out, tok{kind: kNumber, lit: input[start:i], typ: token.Number})
			continue
		}

		// מזהה / מילה
		if isIdentStart(r) {
			start := i
			i += size
			for i < len(input) {
				cr, csz := utf8.DecodeRuneInString(input[i:])
				if !isIdentPart(cr) {
					break
				}
				i += csz
			}
			lit := input[start:i]
			out = append(out, tok{kind: kIdent, lit: lit, typ: token.LookupIdent(lit)})
			continue
		}

		// אופרטור דו־תווי
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
			}
			if tt != "" {
				out = append(out, tok{kind: kOp, lit: two, typ: tt})
				i += 2
				continue
			}
		}

		tt := token.Illegal
		lit := string(r)
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
		out = append(out, tok{kind: kOp, lit: lit, typ: tt})
		i += size
	}
	return out
}

// normalizeMirroredBraces ממיר } גוף { ל־{ גוף } כשאין סוגריים בגוף (נפוץ ב־RTL)
func normalizeMirroredBraces(toks []tok) []tok {
	out := make([]tok, len(toks))
	copy(out, toks)
	for i := 0; i < len(out); i++ {
		if !isBlockKeyword(out[i].typ) {
			continue
		}
		j := i + 1
		depthParen := 0
		for j < len(out) {
			t := out[j]
			if t.kind == kComment {
				j++
				continue
			}
			switch t.typ {
			case token.LParen, token.LBracket:
				depthParen++
			case token.RParen, token.RBracket:
				if depthParen > 0 {
					depthParen--
				}
			case token.LBrace:
				j = -1 // כבר פתיחה תקינה
			case token.RBrace:
				if depthParen != 0 {
					break
				}
				// חפש { סוגר בלי סוגריים מסולסלים באמצע
				k := j + 1
				ok := false
				for k < len(out) {
					if out[k].typ == token.LBrace {
						ok = true
						break
					}
					if out[k].typ == token.RBrace || out[k].typ == token.LBrace {
						ok = false
						break
					}
					k++
				}
				if ok {
					out[j].typ = token.LBrace
					out[j].lit = "{"
					out[k].typ = token.RBrace
					out[k].lit = "}"
				}
				j = -1
			}
			if j < 0 {
				break
			}
			j++
		}
	}
	return out
}

func isBlockKeyword(t token.Type) bool {
	switch t {
	case token.If, token.Else, token.ElseIf, token.While, token.For,
		token.Function, token.Class, token.Try, token.Catch,
		token.Switch, token.Case, token.Default:
		return true
	}
	return false
}

func printToks(toks []tok) string {
	var b strings.Builder
	indent := 0
	atLineStart := true
	needSpace := false
	needCloseCond := 0 // כמה ) לסגור לפני בלוק (עטיפת תנאי)
	prev := tok{typ: token.Illegal}
	var braceIsHash []bool // מחסנית: האם { הנוכחי הוא מילון

	writeIndent := func() {
		for i := 0; i < indent; i++ {
			b.WriteString(indentUnit)
		}
	}
	newline := func() {
		b.WriteByte('\n')
		atLineStart = true
		needSpace = false
	}
	ensureSpace := func() {
		if atLineStart {
			writeIndent()
			atLineStart = false
		} else if needSpace {
			b.WriteByte(' ')
		}
		needSpace = false
	}
	writeLit := func(s string) {
		ensureSpace()
		b.WriteString(s)
	}
	endStatementIfNeeded := func() {
		if atLineStart {
			return
		}
		switch prev.typ {
		case token.Semicolon, token.RBrace, token.LBrace, token.End, token.Illegal:
			return
		}
		if endsExprOrStmt(prev) || prev.typ == token.Break || prev.typ == token.Continue {
			newline()
		}
	}
	closeCondParens := func() {
		for needCloseCond > 0 {
			b.WriteByte(')')
			needCloseCond--
		}
	}

	for i, t := range toks {
		next := tok{typ: token.EOF}
		if i+1 < len(toks) {
			next = toks[i+1]
		}

		switch {
		case t.kind == kComment:
			endStatementIfNeeded()
			if !atLineStart {
				newline()
			}
			writeIndent()
			b.WriteString(t.lit)
			newline()
			prev = t
			continue

		case t.typ == token.LBrace:
			closeCondParens()
			hash := isHashBrace(prev)
			braceIsHash = append(braceIsHash, hash)
			if hash {
				if atLineStart {
					writeIndent()
				}
				atLineStart = false
				needSpace = false
				b.WriteByte('{')
				needSpace = false
				prev = t
				continue
			}
			// בלוק קוד → סגנון סוף (בלי {)
			indent++
			newline()
			prev = t
			continue

		case t.typ == token.RBrace:
			hash := false
			if n := len(braceIsHash); n > 0 {
				hash = braceIsHash[n-1]
				braceIsHash = braceIsHash[:n-1]
			}
			if !hash {
				endStatementIfNeeded()
			}
			if hash {
				b.WriteByte('}')
				atLineStart = false
				needSpace = false
				prev = t
				continue
			}
			if !atLineStart {
				newline()
			}
			indent--
			if indent < 0 {
				indent = 0
			}
			if next.typ == token.Else || next.typ == token.ElseIf || next.typ == token.Catch {
				// סוף הביניים של אם/נסה — בלי לכתוב סוף (כבר הוקטן indent)
				prev = tok{kind: kOp, lit: "}", typ: token.RBrace}
				continue
			}
			writeIndent()
			b.WriteString("סוף")
			atLineStart = false
			newline()
			if indent == 0 && next.typ != token.EOF && next.kind != kComment {
				newline()
			}
			prev = tok{kind: kIdent, lit: "סוף", typ: token.End}
			continue

		case t.typ == token.End:
			endStatementIfNeeded()
			if !atLineStart {
				newline()
			}
			indent--
			if indent < 0 {
				indent = 0
			}
			writeIndent()
			b.WriteString("סוף")
			atLineStart = false
			newline()
			if indent == 0 && next.typ != token.EOF && next.kind != kComment &&
				next.typ != token.Else && next.typ != token.ElseIf && next.typ != token.Catch {
				newline()
			}
			prev = t
			continue

		case t.typ == token.Comma:
			b.WriteByte(',')
			needSpace = true
			atLineStart = false
			prev = t
			continue

		case t.typ == token.Semicolon:
			// ; אופציונלי — מסירים בעיצוב, פקודה אחת לשורה
			atLineStart = false
			needSpace = false
			prev = t
			newline()
			continue

		case t.typ == token.Dot:
			b.WriteByte('.')
			needSpace = false
			atLineStart = false
			prev = t
			continue

		case t.typ == token.LParen || t.typ == token.LBracket:
			if spaceBeforeParen(prev) {
				needSpace = true
			} else {
				needSpace = false
			}
			writeLit(t.lit)
			needSpace = false
			prev = t
			continue

		case t.typ == token.RParen || t.typ == token.RBracket:
			b.WriteString(t.lit)
			needSpace = false
			atLineStart = false
			prev = t
			if t.typ == token.RParen && sofBodyAfter(toks, i, next) {
				closeCondParens()
				indent++
				newline()
			}
			continue

		case t.typ == token.Colon:
			b.WriteByte(':')
			needSpace = true
			atLineStart = false
			prev = t
			continue

		case isBinaryOp(t.typ):
			needSpace = true
			writeLit(t.lit)
			needSpace = true
			prev = t
			continue

		case t.typ == token.Bang || t.typ == token.Minus:
			if isPrefixContext(prev) {
				if atLineStart {
					writeIndent()
					atLineStart = false
				} else if needSpace {
					b.WriteByte(' ')
				}
				b.WriteString(t.lit)
				needSpace = false
			} else {
				needSpace = true
				writeLit(t.lit)
				needSpace = true
			}
			prev = t
			continue
		}

		if shouldBreakBefore(prev, t, atLineStart) {
			oldPrev := prev
			endStatementIfNeeded()
			if !atLineStart {
				newline()
			}
			if indent == 0 && isTopLevelBreak(oldPrev, t) {
				newline()
			}
		}
		// אחרת / אחרת_אם / תפוס — באותה רמה כמו אם/נסה (לא בתוך הגוף)
		if (t.typ == token.Else || t.typ == token.ElseIf || t.typ == token.Catch) && prev.typ != token.RBrace {
			if !atLineStart {
				newline()
			}
			indent--
			if indent < 0 {
				indent = 0
			}
		}
		if atLineStart {
			needSpace = false
		} else if prev.typ == token.RBrace && (t.typ == token.Else || t.typ == token.ElseIf || t.typ == token.Catch) {
			needSpace = false
		} else if prev.typ == token.End && (t.typ == token.Else || t.typ == token.ElseIf || t.typ == token.Catch) {
			needSpace = false
		} else if prev.typ == token.Semicolon {
			needSpace = true
		} else {
			needSpace = needSpaceAfter(prev)
		}
		writeLit(renderTok(t))
		oldPrev := prev
		if sofOpensAfter(oldPrev, t, next) {
			closeCondParens()
			indent++
			newline()
			needSpace = false
			prev = t
			continue
		}
		needSpace = true
		prev = t
	}
	endStatementIfNeeded()

	out := b.String()
	out = strings.TrimRight(out, "\n") + "\n"
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return out
}

func isHashBrace(prev tok) bool {
	switch prev.typ {
	case token.Assign, token.Comma, token.Colon, token.LBracket, token.LParen,
		token.Return, token.Throw:
		return true
	}
	return false
}

func sofBodyAfter(toks []tok, rparenIdx int, next tok) bool {
	if next.typ == token.LBrace || next.typ == token.RBrace || next.typ == token.End ||
		next.typ == token.Semicolon || next.typ == token.Comma || next.typ == token.Dot ||
		next.typ == token.RParen || next.typ == token.RBracket || next.typ == token.EOF {
		return false
	}
	if next.kind == kOp && next.typ != token.Bang {
		return false
	}
	return isBlockHeaderParen(toks, rparenIdx)
}

func isBlockHeaderParen(toks []tok, rparenIdx int) bool {
	depth := 0
	for j := rparenIdx; j >= 0; j-- {
		switch toks[j].typ {
		case token.RParen:
			if j != rparenIdx {
				depth++
			}
		case token.LParen:
			if depth > 0 {
				depth--
				continue
			}
			for k := j - 1; k >= 0; k-- {
				if toks[k].kind == kComment {
					continue
				}
				switch toks[k].typ {
				case token.Function, token.If, token.ElseIf, token.While, token.For:
					return true
				case token.Ident:
					continue
				default:
					return false
				}
			}
			return false
		}
	}
	return false
}

func sofOpensAfter(prev, t, next tok) bool {
	if next.typ == token.LBrace || next.typ == token.RBrace || next.typ == token.End || next.typ == token.EOF {
		return false
	}
	switch t.typ {
	case token.Else, token.Try:
		return true
	case token.Ident:
		switch prev.typ {
		case token.Catch:
			return true
		case token.Class:
			return next.typ != token.Extends
		case token.Extends:
			return true
		case token.Function:
			// פונקציה שם — בלי פרמטרים
			return next.typ != token.Ident && next.typ != token.LParen
		case token.Comma:
			// פרמטר אחרון: פונקציה כפל x, y → גוף
			return isSofBodyStart(next)
		case token.Ident:
			// פרמטר יחיד: פונקציה כפל x → גוף
			return isSofBodyStart(next)
		}
	}
	return false
}

func isSofBodyStart(t tok) bool {
	switch t.typ {
	case token.Comma, token.LParen, token.Colon, token.Dot, token.Assign:
		return false
	}
	return true
}

func isCondKeyword(t token.Type) bool {
	switch t {
	case token.If, token.ElseIf, token.While, token.For:
		return true
	}
	return false
}

func renderTok(t tok) string {
	if t.kind == kString {
		return quoteString(t.lit)
	}
	return t.lit
}

func quoteString(raw string) string {
	// lit כבר כולל את המרכאות מהסורק
	if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') {
		return raw
	}
	return `"` + raw + `"`
}

func isBinaryOp(t token.Type) bool {
	switch t {
	case token.Assign, token.PlusAssign, token.MinusAssign, token.AsteriskAssign,
		token.SlashAssign, token.PercentAssign,
		token.Plus, token.Asterisk, token.Slash, token.Percent, token.Minus,
		token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq,
		token.And, token.Or, token.NullCoalesce, token.Power:
		return true
	}
	return false
}

func isPrefixContext(prev tok) bool {
	switch prev.typ {
	case token.Illegal, token.Assign, token.PlusAssign, token.MinusAssign,
		token.AsteriskAssign, token.SlashAssign, token.PercentAssign,
		token.Plus, token.Minus, token.Asterisk,
		token.Slash, token.Percent, token.Eq, token.NotEq, token.Lt, token.Gt,
		token.LtEq, token.GtEq, token.Bang, token.Not, token.And, token.Or,
		token.Comma, token.LParen,
		token.LBracket, token.Colon, token.Return, token.If, token.While,
		token.For, token.Throw:
		return true
	}
	return prev.kind == kComment
}

func spaceBeforeParen(prev tok) bool {
	// רווח לפני ( לקיבוץ; קריאות כבר הומרו ל־:
	switch prev.typ {
	case token.If, token.While, token.For, token.Catch, token.Function,
		token.Assign, token.PlusAssign, token.MinusAssign, token.AsteriskAssign,
		token.SlashAssign, token.PercentAssign,
		token.Plus, token.Minus, token.Asterisk, token.Slash,
		token.Percent, token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq,
		token.And, token.Or, token.Bang, token.Not, token.Comma, token.Colon,
		token.Return, token.Throw:
		return true
	}
	return false
}

func needSpaceAfter(prev tok) bool {
	switch prev.typ {
	case token.LParen, token.LBracket, token.Dot, token.Bang:
		return false
	}
	return true
}

func shouldBreakBefore(prev, cur tok, atLineStart bool) bool {
	if atLineStart {
		return false
	}
	// מרחיב / בתוך / פרמטרים חשופים — המשך כותרת
	if cur.typ == token.Extends || cur.typ == token.In {
		return false
	}
	if cur.typ == token.Ident && (prev.typ == token.Ident || prev.typ == token.Comma || prev.typ == token.Function) {
		return false
	}
	// סוף / } אחרת / } תפוס — באותה שורה לוגית
	if prev.typ == token.RBrace || prev.typ == token.End {
		switch cur.typ {
		case token.Else, token.ElseIf, token.Catch:
			return false
		}
	}
	// אופרטור / סוגריים / נקודה — ממשיכים ביטוי
	if cur.kind == kOp {
		return false
	}
	// אחרי מילה שמצפה להמשך באותה שורה
	if expectsContinue(prev) {
		return false
	}
	// סוף משפט קודם + התחלת משפט חדש
	if endsExprOrStmt(prev) && (isStmtStart(cur) || cur.kind == kIdent) {
		return true
	}
	return false
}

func isTopLevelBreak(prev, cur tok) bool {
	// אחרי ביטוי/קריאה — רווח לפני בלוק/הצהרה חדשים
	if !endsExprOrStmt(prev) {
		return false
	}
	switch cur.typ {
	case token.Function, token.Class, token.If, token.While, token.For, token.Try, token.Var:
		return true
	}
	return false
}

func isStmtStart(t tok) bool {
	switch t.typ {
	case token.Var, token.If, token.While, token.For, token.Function, token.Class,
		token.Return, token.Break, token.Continue, token.Include, token.Try,
		token.Throw, token.Private, token.Public, token.Else, token.ElseIf:
		return true
	}
	return t.kind == kIdent
}

func expectsContinue(prev tok) bool {
	switch prev.typ {
	case token.If, token.Else, token.ElseIf, token.While, token.For, token.Return,
		token.Throw, token.Var, token.Function, token.Class, token.New, token.Extends,
		token.Include, token.Private, token.Public, token.Catch, token.In, token.Try,
		token.Assign, token.Plus, token.Minus, token.Asterisk, token.Slash, token.Percent,
		token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.Bang,
		token.Comma, token.Colon, token.Dot, token.LParen, token.LBracket, token.LBrace,
		token.End:
		return true
	}
	return false
}

func endsExprOrStmt(prev tok) bool {
	switch prev.typ {
	case token.RBrace, token.End, token.Semicolon, token.RParen, token.RBracket,
		token.Ident, token.Number, token.String, token.True, token.False,
		token.Null, token.This:
		return true
	}
	return prev.kind == kIdent || prev.kind == kNumber || prev.kind == kString
}

func runeAt(s string, i int) rune {
	r, _ := utf8.DecodeRuneInString(s[i:])
	return r
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
