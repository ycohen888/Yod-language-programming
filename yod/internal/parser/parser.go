package parser

import (
	"fmt"
	"strconv"

	"yod/internal/ast"
	"yod/internal/lexer"
	"yod/internal/token"
)

const (
	_ int = iota
	lowest
	assign      // = += -= …
	coalesce    // ??
	logicalOr   // או ||
	logicalAnd  // וגם &&
	equals      // == !=
	lessgreater // > < >= <=
	sum         // + -
	product     // * / %
	power       // **
	prefix      // -x !x לא x
	call        // fn()
	index       // . []
)

var precedences = map[token.Type]int{
	token.Assign:         assign,
	token.PlusAssign:     assign,
	token.MinusAssign:    assign,
	token.AsteriskAssign: assign,
	token.SlashAssign:    assign,
	token.PercentAssign:  assign,
	token.NullCoalesce:   coalesce,
	token.Or:             logicalOr,
	token.And:            logicalAnd,
	token.Eq:             equals,
	token.NotEq:          equals,
	token.Lt:             lessgreater,
	token.Gt:             lessgreater,
	token.LtEq:           lessgreater,
	token.GtEq:           lessgreater,
	token.Plus:           sum,
	token.Minus:          sum,
	token.Slash:          product,
	token.Asterisk:       product,
	token.Percent:        product,
	token.Power:          power,
	token.LParen:         call,
	token.Colon:          call, // קריאה: הדפס: "שלום"
	token.Dot:            index,
	token.LBracket:       index,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	p.prefixParseFns = map[token.Type]prefixParseFn{
		token.Ident:  p.parseIdentifier,
		token.Number: p.parseNumberLiteral,
		token.String: p.parseStringLiteral,
		token.True:   p.parseBoolean,
		token.False:  p.parseBoolean,
		token.Null:   p.parseNull,
		token.This:   p.parseThis,
		token.Parent: p.parseParent,
		token.Bang:   p.parsePrefixExpression,
		token.Not:    p.parsePrefixExpression,
		token.Minus:  p.parsePrefixExpression,
		token.LParen: p.parseGroupedExpression,
		token.New:      p.parseNewExpression,
		token.LBracket: p.parseArrayLiteral,
		token.LBrace:   p.parseHashLiteral,
		token.Function: p.parseFunctionLiteral,
	}
	p.infixParseFns = map[token.Type]infixParseFn{
		token.Plus:           p.parseInfixExpression,
		token.Minus:          p.parseInfixExpression,
		token.Slash:          p.parseInfixExpression,
		token.Asterisk:       p.parseInfixExpression,
		token.Percent:        p.parseInfixExpression,
		token.Eq:             p.parseInfixExpression,
		token.NotEq:          p.parseInfixExpression,
		token.Lt:             p.parseInfixExpression,
		token.Gt:             p.parseInfixExpression,
		token.LtEq:           p.parseInfixExpression,
		token.GtEq:           p.parseInfixExpression,
		token.And:            p.parseLogicalInfix,
		token.Or:             p.parseLogicalInfix,
		token.NullCoalesce:   p.parseNullCoalesce,
		token.Power:          p.parsePowerInfix,
		token.LParen:         p.parseCallExpression,
		token.Colon:          p.parseColonCallExpression,
		token.Dot:            p.parseMemberExpression,
		token.Assign:         p.parseAssignExpression,
		token.PlusAssign:     p.parseCompoundAssign,
		token.MinusAssign:    p.parseCompoundAssign,
		token.AsteriskAssign: p.parseCompoundAssign,
		token.SlashAssign:    p.parseCompoundAssign,
		token.PercentAssign:  p.parseCompoundAssign,
		token.LBracket:       p.parseIndexExpression,
	}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curIs(t token.Type) bool  { return p.curToken.Type == t }
func (p *Parser) peekIs(t token.Type) bool { return p.peekToken.Type == t }

func (p *Parser) expectPeek(t token.Type) bool {
	if p.peekIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// blockStyle — בלוק עם {} או עם מילת סוף
type blockStyle int

const (
	blockBrace blockStyle = iota
	blockSof
)

// startBlock: אחרי כותרת בלוק — { / } מראה, או בלוק סוף (בלי פותח)
func (p *Parser) startBlock() blockStyle {
	if p.peekIs(token.LBrace) || p.peekIs(token.RBrace) {
		p.nextToken()
		return blockBrace
	}
	return blockSof
}

func (p *Parser) blockCloseOf(open token.Type) token.Type {
	if open == token.RBrace {
		return token.LBrace
	}
	return token.RBrace
}

func (p *Parser) curIsAny(types ...token.Type) bool {
	for _, t := range types {
		if p.curIs(t) {
			return true
		}
	}
	return false
}

func (p *Parser) peekError(t token.Type) {
	msg := fmt.Sprintf("שורה %d: ציפיתי ל־%q אבל קיבלתי %q", p.peekToken.Line, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// expectSemicolon מסיים משפט: ; אופציונלי, או סוף שורה / סוגר בלוק.
// כמה פקודות באותה שורה עדיין אפשריות עם ;
func (p *Parser) expectSemicolon() bool {
	if p.peekIs(token.Semicolon) {
		p.nextToken()
		return true
	}
	if p.curIs(token.End) {
		return true
	}
	if p.peekIsStatementEnd() {
		return true
	}
	p.addError(fmt.Sprintf("שורה %d: פקודה חדשה באותה שורה — העבירו לשורה הבאה (או ; ביניהן)", p.peekToken.Line))
	return false
}

// peekIsStatementEnd — סוף משפט בלי ; (שורה חדשה או סוגר מבנה)
func (p *Parser) peekIsStatementEnd() bool {
	switch p.peekToken.Type {
	case token.EOF, token.End, token.RBrace, token.Else, token.ElseIf, token.Catch,
		token.Case, token.Default:
		return true
	}
	return p.peekToken.Line > p.curToken.Line
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return lowest
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return lowest
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Statement{}}
	for !p.curIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.Var:
		return p.parseVarStatement()
	case token.Return:
		return p.parseReturnStatement()
	case token.If:
		return p.parseIfStatement()
	case token.Switch:
		return p.parseSwitchStatement()
	case token.While:
		return p.parseWhileStatement()
	case token.For:
		return p.parseForInStatement()
	case token.Break:
		tok := p.curToken
		if p.peekIs(token.Semicolon) {
			p.nextToken()
		} else if !p.peekIsStatementEnd() {
			p.addError(fmt.Sprintf("שורה %d: אחרי עצור — שורה חדשה או ;", p.peekToken.Line))
		}
		return &ast.BreakStatement{Tok: tok}
	case token.Continue:
		tok := p.curToken
		if p.peekIs(token.Semicolon) {
			p.nextToken()
		} else if !p.peekIsStatementEnd() {
			p.addError(fmt.Sprintf("שורה %d: אחרי המשך — שורה חדשה או ;", p.peekToken.Line))
		}
		return &ast.ContinueStatement{Tok: tok}
	case token.Function:
		return p.parseFunctionStatement()
	case token.Class:
		return p.parseClassStatement()
	case token.Include:
		return p.parseIncludeStatement()
	case token.Try:
		return p.parseTryStatement()
	case token.Throw:
		return p.parseThrowStatement()
	case token.Ident:
		if p.peekIs(token.Assign) {
			return p.parseAssignStatement()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseVarStatement() *ast.VarStatement {
	stmt := &ast.VarStatement{Tok: p.curToken}
	if !p.expectPeek(token.Ident) {
		return nil
	}
	stmt.Name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	if p.peekIs(token.Assign) {
		p.nextToken() // =
		p.nextToken() // תחילת ערך
		stmt.Value = p.parseExpression(lowest)
	} else if !p.peekIs(token.Semicolon) && !p.peekIsStatementEnd() {
		p.addError(fmt.Sprintf("שורה %d: אחרי שם משתנה צפוי = ערך, או סוף שורה", p.peekToken.Line))
	}
	p.expectSemicolon()
	return stmt
}

func (p *Parser) parseAssignStatement() *ast.AssignStatement {
	stmt := &ast.AssignStatement{Tok: p.curToken}
	stmt.Name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	p.nextToken() // =
	p.nextToken()
	stmt.Value = p.parseExpression(lowest)
	p.expectSemicolon()
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Tok: p.curToken}
	line := p.curToken.Line
	if p.peekIs(token.Semicolon) {
		p.nextToken()
		stmt.Value = nil
		return stmt
	}
	// החזר לבד — סוף שורה / סוף בלוק
	if p.peekToken.Line > line || p.peekIs(token.EOF) || p.peekIs(token.End) ||
		p.peekIs(token.RBrace) || p.peekIs(token.Else) || p.peekIs(token.ElseIf) ||
		p.peekIs(token.Catch) || p.peekIs(token.Case) || p.peekIs(token.Default) {
		stmt.Value = nil
		return stmt
	}
	p.nextToken()
	if p.curIs(token.Colon) {
		p.nextToken()
	}
	stmt.Value = p.parseExpression(lowest)
	p.expectSemicolon()
	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Tok: p.curToken}
	stmt.Expr = p.parseExpression(lowest)
	p.expectSemicolon()
	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	return p.parseBraceBlock()
}

func (p *Parser) parseBraceBlock() *ast.BlockStatement {
	block := &ast.BlockStatement{Tok: p.curToken, Statements: []ast.Statement{}}
	closeType := p.blockCloseOf(p.curToken.Type)
	p.nextToken()
	for !p.curIs(closeType) && !p.curIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	if !p.curIs(closeType) {
		want := "}"
		if closeType == token.LBrace {
			want = "{"
		}
		p.addError(fmt.Sprintf("שורה %d: שכחת לסגור בלוק עם %s", p.curToken.Line, want))
	}
	return block
}

// parseSofBlock קורא משפטים עד מילת עצירה (סוף / אחרת / …). cur נשאר על מילת העצירה.
func (p *Parser) parseSofBlock(stop ...token.Type) *ast.BlockStatement {
	block := &ast.BlockStatement{Tok: p.curToken, Statements: []ast.Statement{}}
	p.nextToken()
	for !p.curIsAny(stop...) && !p.curIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	return block
}

func (p *Parser) expectSofEnd(what string) {
	if p.curIs(token.End) {
		return
	}
	p.addError(fmt.Sprintf("שורה %d: שכחת «סוף» אחרי %s", p.curToken.Line, what))
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	return p.parseIfChain(true)
}

func (p *Parser) parseSwitchStatement() *ast.SwitchStatement {
	stmt := &ast.SwitchStatement{Tok: p.curToken}
	stmt.Value = p.parseCondition()
	if stmt.Value == nil {
		return nil
	}
	// אחרי הביטוי — מצפים ל־מקרה / ברירת_מחדל / סוף
	p.nextToken()
	for !p.curIs(token.End) && !p.curIs(token.EOF) {
		switch p.curToken.Type {
		case token.Case:
			cas := ast.SwitchCase{}
			p.nextToken()
			cas.Values = append(cas.Values, p.parseExpression(lowest))
			for p.peekIs(token.Comma) {
				p.nextToken() // ,
				p.nextToken()
				cas.Values = append(cas.Values, p.parseExpression(lowest))
			}
			cas.Body = p.parseSofBlock(token.Case, token.Default, token.End)
			stmt.Cases = append(stmt.Cases, cas)
		case token.Default:
			stmt.Default = p.parseSofBlock(token.End)
		default:
			p.addError(fmt.Sprintf("שורה %d: בתוך בחר צפוי מקרה / ברירת_מחדל / סוף, קיבלתי %q", p.curToken.Line, p.curToken.Literal))
			p.nextToken()
		}
	}
	p.expectSofEnd("בחר")
	return stmt
}

// parseIfChain — wantEnd=true דורש «סוף» בסוף שרשרת אם/אחרת (בסגנון סוף)
func (p *Parser) parseIfChain(wantEnd bool) *ast.IfStatement {
	stmt := &ast.IfStatement{Tok: p.curToken}
	stmt.Condition = p.parseCondition()
	if stmt.Condition == nil {
		return nil
	}

	style := p.startBlock()
	if style == blockBrace {
		stmt.Consequence = p.parseBraceBlock()
		if p.peekIs(token.ElseIf) {
			p.nextToken()
			stmt.Alternative = p.parseIfChain(false)
		} else if p.peekIs(token.Else) {
			p.nextToken()
			if p.startBlock() != blockBrace {
				p.addError(fmt.Sprintf("שורה %d: אחרי אחרת בסגנון {} צריך {", p.curToken.Line))
				return stmt
			}
			stmt.Alternative = p.parseBraceBlock()
		}
		return stmt
	}

	stmt.Consequence = p.parseSofBlock(token.Else, token.ElseIf, token.End)
	if p.curIs(token.ElseIf) {
		stmt.Alternative = p.parseIfChain(false)
	} else if p.curIs(token.Else) {
		stmt.Alternative = p.parseSofBlock(token.End)
	}
	if wantEnd {
		p.expectSofEnd("אם")
	}
	return stmt
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Tok: p.curToken}
	stmt.Condition = p.parseCondition()
	if stmt.Condition == nil {
		return nil
	}
	if p.startBlock() == blockBrace {
		stmt.Body = p.parseBraceBlock()
		return stmt
	}
	stmt.Body = p.parseSofBlock(token.End)
	p.expectSofEnd("כל_עוד")
	return stmt
}

// parseCondition — תנאי עם או בלי סוגריים: אם (x) או אם x
func (p *Parser) parseCondition() ast.Expression {
	if p.peekIs(token.LParen) {
		p.nextToken() // (
		p.nextToken()
		cond := p.parseExpression(lowest)
		if !p.expectPeek(token.RParen) {
			return nil
		}
		return cond
	}
	p.nextToken()
	return p.parseExpression(lowest)
}

func (p *Parser) parseFunctionStatement() *ast.VarStatement {
	// פונקציה שם(...) { ... } → משתנה שם = פונקציה...
	tok := p.curToken
	if !p.expectPeek(token.Ident) {
		return nil
	}
	name := &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	fn := p.parseFunctionLiteralFrom(tok, name)
	if fn == nil {
		return nil
	}
	return &ast.VarStatement{Tok: tok, Name: name, Value: fn}
}

// parseFunctionLiteral — פונקציה אנונימית או עם שם, בביטוי (למשל בלחיצה(פונקציה(){...}))
func (p *Parser) parseFunctionLiteral() ast.Expression {
	tok := p.curToken
	var name *ast.Identifier
	if p.peekIs(token.Ident) {
		p.nextToken()
		name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	}
	return p.parseFunctionLiteralFrom(tok, name)
}

func (p *Parser) parseFunctionLiteralFrom(tok token.Token, name *ast.Identifier) *ast.FunctionLiteral {
	fn := &ast.FunctionLiteral{Tok: tok, Name: name}
	if p.peekIs(token.LParen) {
		p.nextToken() // (
		fn.Parameters = p.parseFunctionParameters()
	} else if p.peekIs(token.Ident) && name != nil && p.peekToken.Line == name.Tok.Line {
		// פרמטרים חשופים באותה שורה: פונקציה כפל x, y
		fn.Parameters = p.parseBareParameters()
	} else if p.peekIs(token.Ident) && name == nil && p.peekToken.Line == tok.Line {
		// אנונימית עם פרמטרים באותה שורה — נדיר; עדיף פונקציה(x)
		fn.Parameters = p.parseBareParameters()
	} else {
		fn.Parameters = nil
	}
	if p.startBlock() == blockBrace {
		fn.Body = p.parseBraceBlock()
		return fn
	}
	fn.Body = p.parseSofBlock(token.End)
	p.expectSofEnd("פונקציה")
	return fn
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	ids := []*ast.Identifier{}
	if p.peekIs(token.RParen) {
		p.nextToken()
		return ids
	}
	p.nextToken()
	ids = append(ids, &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal})
	for p.peekIs(token.Comma) {
		p.nextToken()
		p.nextToken()
		ids = append(ids, &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal})
	}
	if !p.expectPeek(token.RParen) {
		return nil
	}
	return ids
}

// parseBareParameters — פונקציה כפל x, y  (בלי סוגריים)
func (p *Parser) parseBareParameters() []*ast.Identifier {
	ids := []*ast.Identifier{}
	p.nextToken()
	ids = append(ids, &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal})
	for p.peekIs(token.Comma) {
		p.nextToken()
		if !p.expectPeek(token.Ident) {
			return ids
		}
		ids = append(ids, &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal})
	}
	return ids
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.addError(fmt.Sprintf("שורה %d: לא הצלחתי לפרש את %q", p.curToken.Line, p.curToken.Literal))
		return nil
	}
	left := prefix()
	for !p.peekIs(token.Semicolon) && !p.peekEndsExpressionByNewline() {
		// קריאה בלי סוגריים: הדפס "שלום" / כפל 2, 7
		if precedence < call && p.canJuxtaCall(left) && p.isJuxtaArgStart() {
			left = p.parseJuxtaCall(left)
			continue
		}
		if precedence >= p.peekPrecedence() {
			break
		}
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return left
		}
		p.nextToken()
		left = infix(left)
	}
	return left
}

// peekEndsExpressionByNewline — לא ממשיכים ביטוי לשורה הבאה כשאין אופרטור ממתין.
// אופרטורים אינפיקס (כולל , בתוך רשימות דרך parseExpressionList) ממשיכים כרגיל
// כי הם נצרכים לפני הבדיקה. כאן עוצרים רק כשהפקודה הבאה מתחילה בשורה חדשה.
func (p *Parser) peekEndsExpressionByNewline() bool {
	if p.peekToken.Line <= p.curToken.Line {
		return false
	}
	// שורה חדשה עם אופרטור אינפיקס — ממשיכים (1 +\n 2)
	if _, ok := p.infixParseFns[p.peekToken.Type]; ok {
		return false
	}
	return true
}

func (p *Parser) canJuxtaCall(left ast.Expression) bool {
	switch left.(type) {
	case *ast.Identifier, *ast.MemberExpression:
		return true
	}
	return false
}

func (p *Parser) isJuxtaArgStart() bool {
	// רק באותה שורה — פקודה אחת לשורה
	if p.peekToken.Line != p.curToken.Line {
		return false
	}
	// רק ליטרלים — לא מזהים (כדי לא לבלוע את גוף הלולאה: עבור x בתוך רשימה / הדפס)
	// לא Minus: אחרת «x - y» נפרש כקריאה juxta «x(-y)» במקום חיסור.
	switch p.peekToken.Type {
	case token.String, token.Number, token.True, token.False, token.Null,
		token.Bang:
		return true
	}
	return false
}

func (p *Parser) parseJuxtaCall(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Tok: p.peekToken, Function: function}
	exp.Arguments = p.parseBareArgList()
	return exp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{Tok: p.curToken}
	v, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError(fmt.Sprintf("שורה %d: מספר לא תקין %q", p.curToken.Line, p.curToken.Literal))
		return nil
	}
	lit.Value = v
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Tok: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.BooleanLiteral{Tok: p.curToken, Value: p.curIs(token.True)}
}

func (p *Parser) parseNull() ast.Expression {
	return &ast.NullLiteral{Tok: p.curToken}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	op := p.curToken.Literal
	if p.curToken.Type == token.Not || op == "לא" {
		op = "!"
	}
	expr := &ast.PrefixExpression{Tok: p.curToken, Operator: op}
	p.nextToken()
	expr.Right = p.parseExpression(prefix)
	return expr
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Tok:      p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)
	return expr
}

func (p *Parser) parseLogicalInfix(left ast.Expression) ast.Expression {
	op := "וגם"
	if p.curToken.Type == token.Or || p.curToken.Literal == "||" || p.curToken.Literal == "או" {
		op = "או"
	}
	expr := &ast.InfixExpression{
		Tok:      p.curToken,
		Operator: op,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)
	return expr
}

func (p *Parser) parseNullCoalesce(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Tok:      p.curToken,
		Operator: "??",
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)
	return expr
}

func (p *Parser) parsePowerInfix(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Tok:      p.curToken,
		Operator: "**",
		Left:     left,
	}
	// אסוציאטיביות ימין: 2**3**2 = 2**(3**2)
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence - 1)
	return expr
}

func (p *Parser) parseCompoundAssign(left ast.Expression) ast.Expression {
	tok := p.curToken
	arith := "+"
	switch tok.Type {
	case token.PlusAssign:
		arith = "+"
	case token.MinusAssign:
		arith = "-"
	case token.AsteriskAssign:
		arith = "*"
	case token.SlashAssign:
		arith = "/"
	case token.PercentAssign:
		arith = "%"
	}
	p.nextToken()
	right := p.parseExpression(lowest)
	return &ast.AssignExpression{
		Tok:  tok,
		Left: left,
		Value: &ast.InfixExpression{
			Tok:      tok,
			Operator: arith,
			Left:     left,
			Right:    right,
		},
	}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(lowest)
	if !p.expectPeek(token.RParen) {
		return nil
	}
	return exp
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Tok: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RParen)
	return exp
}

// parseColonCallExpression — הדפס: "שלום" / כפל: 2, 7  (בלי סוגריים)
func (p *Parser) parseColonCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Tok: p.curToken, Function: function}
	exp.Arguments = p.parseBareArgList()
	return exp
}

func (p *Parser) parseBareArgList() []ast.Expression {
	list := []ast.Expression{}
	if p.isBareArgEnd() {
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(lowest))
	for p.peekIs(token.Comma) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(lowest))
	}
	return list
}

func (p *Parser) isBareArgEnd() bool {
	if p.peekToken.Line > p.curToken.Line {
		return true
	}
	switch p.peekToken.Type {
	case token.Semicolon, token.EOF, token.End, token.Else, token.ElseIf, token.Catch,
		token.Case, token.Default,
		token.RParen, token.RBrace, token.RBracket, token.LBrace:
		return true
	}
	return false
}

func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	list := []ast.Expression{}
	if p.peekIs(end) {
		p.nextToken()
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(lowest))
	for p.peekIs(token.Comma) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(lowest))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

func (p *Parser) parseClassStatement() *ast.ClassStatement {
	stmt := &ast.ClassStatement{Tok: p.curToken}
	if !p.expectPeek(token.Ident) {
		return nil
	}
	stmt.Name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	if p.peekIs(token.Extends) {
		p.nextToken() // מרחיב
		if !p.expectPeek(token.Ident) {
			p.addError(fmt.Sprintf("שורה %d: אחרי מרחיב צריך שם מחלקת הורה", p.curToken.Line))
			return nil
		}
		stmt.Parent = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	}
	if p.startBlock() == blockBrace {
		stmt.Body = p.parseClassBodyBrace()
		return stmt
	}
	stmt.Body = p.parseClassBodySof()
	p.expectSofEnd("מחלקה")
	return stmt
}

func (p *Parser) parseClassBodyBrace() *ast.BlockStatement {
	block := &ast.BlockStatement{Tok: p.curToken, Statements: []ast.Statement{}}
	closeType := p.blockCloseOf(p.curToken.Type)
	p.nextToken()
	for !p.curIs(closeType) && !p.curIs(token.EOF) {
		if !p.parseClassMember(block) {
			continue
		}
		p.nextToken()
	}
	if !p.curIs(closeType) {
		want := "}"
		if closeType == token.LBrace {
			want = "{"
		}
		p.addError(fmt.Sprintf("שורה %d: שכחת לסגור מחלקה עם %s", p.curToken.Line, want))
	}
	return block
}

func (p *Parser) parseClassBodySof() *ast.BlockStatement {
	block := &ast.BlockStatement{Tok: p.curToken, Statements: []ast.Statement{}}
	p.nextToken()
	for !p.curIs(token.End) && !p.curIs(token.EOF) {
		if !p.parseClassMember(block) {
			continue
		}
		p.nextToken()
	}
	return block
}

func (p *Parser) parseClassMember(block *ast.BlockStatement) bool {
	vis := ast.VisPublic
	switch p.curToken.Type {
	case token.Private:
		vis = ast.VisPrivate
		p.nextToken()
	case token.Public:
		vis = ast.VisPublic
		p.nextToken()
	}

	var stmt ast.Statement
	switch p.curToken.Type {
	case token.Var:
		vs := p.parseVarStatement()
		if vs != nil {
			vs.Visibility = vis
			stmt = vs
		}
	case token.Function:
		fs := p.parseFunctionStatement()
		if fs != nil {
			if fl, ok := fs.Value.(*ast.FunctionLiteral); ok {
				fl.Visibility = vis
			}
			fs.Visibility = vis
			stmt = fs
		}
	default:
		p.addError(fmt.Sprintf("שורה %d: בגוף מחלקה צפוי משתנה או פונקציה (אופציונלי פרטי/ציבורי לפני)", p.curToken.Line))
		p.nextToken()
		return false
	}
	if stmt != nil {
		block.Statements = append(block.Statements, stmt)
	}
	return true
}

func (p *Parser) parseThis() ast.Expression {
	return &ast.ThisLiteral{Tok: p.curToken}
}

func (p *Parser) parseParent() ast.Expression {
	return &ast.ParentLiteral{Tok: p.curToken}
}

func (p *Parser) parseNewExpression() ast.Expression {
	expr := &ast.NewExpression{Tok: p.curToken}
	if !p.expectPeek(token.Ident) {
		return nil
	}
	expr.Name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	if p.peekIs(token.LParen) {
		p.nextToken()
		expr.Arguments = p.parseExpressionList(token.RParen)
	} else if p.peekIs(token.Colon) {
		p.nextToken() // :
		expr.Arguments = p.parseBareArgList()
	}
	return expr
}

func (p *Parser) parseMemberExpression(object ast.Expression) ast.Expression {
	exp := &ast.MemberExpression{Tok: p.curToken, Object: object}
	// אחרי נקודה מותר גם מילת מפתח כשם שדה/מתודה (למשל רשימה.ריק)
	if p.peekToken.Type != token.Ident && !token.IsKeyword(p.peekToken.Type) {
		p.peekError(token.Ident)
		return nil
	}
	p.nextToken()
	exp.Property = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	return exp
}

func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	exp := &ast.AssignExpression{Tok: p.curToken, Left: left}
	p.nextToken()
	exp.Value = p.parseExpression(lowest)
	return exp
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Tok: p.curToken}
	array.Elements = p.parseExpressionList(token.RBracket)
	return array
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Tok: p.curToken, Pairs: []ast.HashPair{}}
	if p.peekIs(token.RBrace) {
		p.nextToken()
		return hash
	}
	for {
		p.nextToken()
		// מפתח מילון — בלי לבלוע את : כקריאה (הדפס: ארג)
		key := p.parseExpression(call)
		if id, ok := key.(*ast.Identifier); ok {
			key = &ast.StringLiteral{Tok: id.Tok, Value: id.Value}
		}
		if !p.expectPeek(token.Colon) {
			p.addError(fmt.Sprintf("שורה %d: במילון צריך מפתח:ערך (חסר :)", p.curToken.Line))
			return nil
		}
		p.nextToken()
		value := p.parseExpression(lowest)
		hash.Pairs = append(hash.Pairs, ast.HashPair{Key: key, Value: value})
		if p.peekIs(token.Comma) {
			p.nextToken()
			continue
		}
		break
	}
	if !p.expectPeek(token.RBrace) {
		return nil
	}
	return hash
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Tok: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(lowest)
	if !p.expectPeek(token.RBracket) {
		return nil
	}
	return exp
}

func (p *Parser) parseForInStatement() *ast.ForInStatement {
	stmt := &ast.ForInStatement{Tok: p.curToken}
	if p.peekIs(token.LParen) {
		p.nextToken() // (
		if !p.expectPeek(token.Ident) {
			return nil
		}
		stmt.Name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
		if !p.expectPeek(token.In) {
			return nil
		}
		p.nextToken()
		stmt.Iterable = p.parseExpression(lowest)
		if !p.expectPeek(token.RParen) {
			return nil
		}
	} else {
		if !p.expectPeek(token.Ident) {
			p.addError(fmt.Sprintf("שורה %d: אחרי עבור צריך שם, למשל עבור שם בתוך רשימה", p.peekToken.Line))
			return nil
		}
		stmt.Name = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
		if !p.expectPeek(token.In) {
			return nil
		}
		p.nextToken()
		stmt.Iterable = p.parseExpression(lowest)
	}
	if p.startBlock() == blockBrace {
		stmt.Body = p.parseBraceBlock()
		return stmt
	}
	stmt.Body = p.parseSofBlock(token.End)
	p.expectSofEnd("עבור")
	return stmt
}

func (p *Parser) parseIncludeStatement() *ast.IncludeStatement {
	stmt := &ast.IncludeStatement{Tok: p.curToken}
	if !p.expectPeek(token.String) {
		p.addError(fmt.Sprintf("שורה %d: אחרי כלול צריך מחרוזת, למשל כלול \"קבצים\"", p.curToken.Line))
		return nil
	}
	stmt.Path = p.curToken.Literal
	p.expectSemicolon()
	return stmt
}

func (p *Parser) parseTryStatement() *ast.TryStatement {
	stmt := &ast.TryStatement{Tok: p.curToken}
	style := p.startBlock()
	if style == blockBrace {
		stmt.Body = p.parseBraceBlock()
		if !p.expectPeek(token.Catch) {
			p.addError(fmt.Sprintf("שורה %d: אחרי נסה {..} חובה תפוס", p.curToken.Line))
			return nil
		}
		if !p.expectPeek(token.Ident) {
			p.addError(fmt.Sprintf("שורה %d: אחרי תפוס צריך שם משתנה לשגיאה", p.curToken.Line))
			return nil
		}
		stmt.CatchName = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
		if p.startBlock() != blockBrace {
			p.addError(fmt.Sprintf("שורה %d: אחרי תפוס בסגנון {} צריך {", p.curToken.Line))
			return nil
		}
		stmt.CatchBody = p.parseBraceBlock()
		return stmt
	}

	stmt.Body = p.parseSofBlock(token.Catch, token.End)
	if !p.curIs(token.Catch) {
		p.addError(fmt.Sprintf("שורה %d: אחרי נסה חובה תפוס", p.curToken.Line))
		return nil
	}
	if !p.expectPeek(token.Ident) {
		p.addError(fmt.Sprintf("שורה %d: אחרי תפוס צריך שם משתנה לשגיאה", p.curToken.Line))
		return nil
	}
	stmt.CatchName = &ast.Identifier{Tok: p.curToken, Value: p.curToken.Literal}
	stmt.CatchBody = p.parseSofBlock(token.End)
	p.expectSofEnd("נסה/תפוס")
	return stmt
}

func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	stmt := &ast.ThrowStatement{Tok: p.curToken}
	p.nextToken()
	if p.curIs(token.Colon) {
		p.nextToken()
	}
	if p.curIs(token.RBrace) || p.curIs(token.End) || p.curIs(token.EOF) || p.curIs(token.Semicolon) {
		p.addError(fmt.Sprintf("שורה %d: אחרי זרוק צריך ערך, למשל זרוק \"שגיאה\"", stmt.Tok.Line))
		return nil
	}
	stmt.Value = p.parseExpression(lowest)
	p.expectSemicolon()
	return stmt
}
