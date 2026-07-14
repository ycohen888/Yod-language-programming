package ast

import "yod/internal/token"

type Node interface {
	TokenLiteral() string
	Line() int
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) Line() int {
	if len(p.Statements) > 0 {
		return p.Statements[0].Line()
	}
	return 1
}

type Visibility int

const (
	VisPublic Visibility = iota
	VisPrivate
)

type VarStatement struct {
	Tok        token.Token
	Name       *Identifier
	Value      Expression
	Visibility Visibility
}

func (s *VarStatement) statementNode()       {}
func (s *VarStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *VarStatement) Line() int            { return s.Tok.Line }

type AssignStatement struct {
	Tok   token.Token
	Name  *Identifier
	Value Expression
}

func (s *AssignStatement) statementNode()       {}
func (s *AssignStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *AssignStatement) Line() int            { return s.Tok.Line }

type ReturnStatement struct {
	Tok   token.Token
	Value Expression
}

func (s *ReturnStatement) statementNode()       {}
func (s *ReturnStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ReturnStatement) Line() int            { return s.Tok.Line }

type ExpressionStatement struct {
	Tok  token.Token
	Expr Expression
}

func (s *ExpressionStatement) statementNode()       {}
func (s *ExpressionStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ExpressionStatement) Line() int            { return s.Tok.Line }

type BlockStatement struct {
	Tok        token.Token
	Statements []Statement
}

func (s *BlockStatement) statementNode()       {}
func (s *BlockStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *BlockStatement) Line() int            { return s.Tok.Line }

type IfStatement struct {
	Tok         token.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative Statement // Block או If (אחרת_אם)
}

func (s *IfStatement) statementNode()       {}
func (s *IfStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *IfStatement) Line() int            { return s.Tok.Line }

type SwitchCase struct {
	Values []Expression
	Body   *BlockStatement
}

type SwitchStatement struct {
	Tok     token.Token
	Value   Expression
	Cases   []SwitchCase
	Default *BlockStatement
}

func (s *SwitchStatement) statementNode()       {}
func (s *SwitchStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *SwitchStatement) Line() int            { return s.Tok.Line }

type WhileStatement struct {
	Tok       token.Token
	Condition Expression
	Body      *BlockStatement
}

func (s *WhileStatement) statementNode()       {}
func (s *WhileStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *WhileStatement) Line() int            { return s.Tok.Line }

type BreakStatement struct {
	Tok token.Token
}

func (s *BreakStatement) statementNode()       {}
func (s *BreakStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *BreakStatement) Line() int            { return s.Tok.Line }

type ContinueStatement struct {
	Tok token.Token
}

func (s *ContinueStatement) statementNode()       {}
func (s *ContinueStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ContinueStatement) Line() int            { return s.Tok.Line }

type FunctionLiteral struct {
	Tok        token.Token
	Name       *Identifier // אופציונלי בהצהרה
	Parameters []*Parameter
	ReturnType string // אופציונלי: -> מספר
	Body       *BlockStatement
	Visibility Visibility
}

// Parameter — פרמטר לפונקציה, עם ברירת מחדל והערת טיפוס אופציונליות.
type Parameter struct {
	Name     *Identifier
	TypeName string     // אופציונלי: א: מספר
	Default  Expression // nil = חובה
}

func (f *FunctionLiteral) expressionNode()      {}
func (f *FunctionLiteral) statementNode()       {}
func (f *FunctionLiteral) TokenLiteral() string { return f.Tok.Literal }
func (f *FunctionLiteral) Line() int            { return f.Tok.Line }

type Identifier struct {
	Tok  token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Tok.Literal }
func (i *Identifier) Line() int            { return i.Tok.Line }

type NumberLiteral struct {
	Tok   token.Token
	Value float64
}

func (n *NumberLiteral) expressionNode()      {}
func (n *NumberLiteral) TokenLiteral() string { return n.Tok.Literal }
func (n *NumberLiteral) Line() int            { return n.Tok.Line }

type StringLiteral struct {
	Tok   token.Token
	Value string
}

func (s *StringLiteral) expressionNode()      {}
func (s *StringLiteral) TokenLiteral() string { return s.Tok.Literal }
func (s *StringLiteral) Line() int            { return s.Tok.Line }

// TemplateLiteral — `טקסט ${ביטוי}` כמו ב־JS
type TemplateLiteral struct {
	Tok    token.Token
	Quasis []string     // len = len(Exprs)+1
	Exprs  []Expression
}

func (t *TemplateLiteral) expressionNode()      {}
func (t *TemplateLiteral) TokenLiteral() string { return t.Tok.Literal }
func (t *TemplateLiteral) Line() int            { return t.Tok.Line }

type BooleanLiteral struct {
	Tok   token.Token
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string { return b.Tok.Literal }
func (b *BooleanLiteral) Line() int            { return b.Tok.Line }

type NullLiteral struct {
	Tok token.Token
}

func (n *NullLiteral) expressionNode()      {}
func (n *NullLiteral) TokenLiteral() string { return n.Tok.Literal }
func (n *NullLiteral) Line() int            { return n.Tok.Line }

type PrefixExpression struct {
	Tok      token.Token
	Operator string
	Right    Expression
}

func (p *PrefixExpression) expressionNode()      {}
func (p *PrefixExpression) TokenLiteral() string { return p.Tok.Literal }
func (p *PrefixExpression) Line() int            { return p.Tok.Line }

type InfixExpression struct {
	Tok      token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (i *InfixExpression) expressionNode()      {}
func (i *InfixExpression) TokenLiteral() string { return i.Tok.Literal }
func (i *InfixExpression) Line() int            { return i.Tok.Line }

type CallExpression struct {
	Tok       token.Token
	Function  Expression
	Arguments []Expression
}

func (c *CallExpression) expressionNode()      {}
func (c *CallExpression) TokenLiteral() string { return c.Tok.Literal }
func (c *CallExpression) Line() int            { return c.Tok.Line }

type ClassStatement struct {
	Tok    token.Token
	Name   *Identifier
	Parent *Identifier // מרחיב שם_הורה (אופציונלי)
	Body   *BlockStatement
}

func (s *ClassStatement) statementNode()       {}
func (s *ClassStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ClassStatement) Line() int            { return s.Tok.Line }

type ThisLiteral struct {
	Tok token.Token
}

func (t *ThisLiteral) expressionNode()      {}
func (t *ThisLiteral) TokenLiteral() string { return t.Tok.Literal }
func (t *ThisLiteral) Line() int            { return t.Tok.Line }

type ParentLiteral struct {
	Tok token.Token
}

func (t *ParentLiteral) expressionNode()      {}
func (t *ParentLiteral) TokenLiteral() string { return t.Tok.Literal }
func (t *ParentLiteral) Line() int            { return t.Tok.Line }

type MemberExpression struct {
	Tok      token.Token
	Object   Expression
	Property *Identifier
}

func (m *MemberExpression) expressionNode()      {}
func (m *MemberExpression) TokenLiteral() string { return m.Tok.Literal }
func (m *MemberExpression) Line() int            { return m.Tok.Line }

type NewExpression struct {
	Tok       token.Token
	Name      *Identifier
	Arguments []Expression
}

func (n *NewExpression) expressionNode()      {}
func (n *NewExpression) TokenLiteral() string { return n.Tok.Literal }
func (n *NewExpression) Line() int            { return n.Tok.Line }

type AssignExpression struct {
	Tok   token.Token
	Left  Expression
	Value Expression
}

func (a *AssignExpression) expressionNode()      {}
func (a *AssignExpression) TokenLiteral() string { return a.Tok.Literal }
func (a *AssignExpression) Line() int            { return a.Tok.Line }

type ArrayLiteral struct {
	Tok      token.Token
	Elements []Expression
}

func (a *ArrayLiteral) expressionNode()      {}
func (a *ArrayLiteral) TokenLiteral() string { return a.Tok.Literal }
func (a *ArrayLiteral) Line() int            { return a.Tok.Line }

type IndexExpression struct {
	Tok   token.Token
	Left  Expression
	Index Expression
}

func (i *IndexExpression) expressionNode()      {}
func (i *IndexExpression) TokenLiteral() string { return i.Tok.Literal }
func (i *IndexExpression) Line() int            { return i.Tok.Line }

type ForInStatement struct {
	Tok      token.Token
	Name     *Identifier
	Iterable Expression
	Body     *BlockStatement
}

func (s *ForInStatement) statementNode()       {}
func (s *ForInStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ForInStatement) Line() int            { return s.Tok.Line }

// ForRangeStatement — עבור i מ 0 עד 10 [בצע 2]
// הקצוות Inclusive; צעד ברירת מחדל 1.
type ForRangeStatement struct {
	Tok   token.Token
	Name  *Identifier
	Start Expression
	End   Expression
	Step  Expression // nil = 1
	Body  *BlockStatement
}

func (s *ForRangeStatement) statementNode()       {}
func (s *ForRangeStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ForRangeStatement) Line() int            { return s.Tok.Line }

type IncludeStatement struct {
	Tok  token.Token
	Path string
}

func (s *IncludeStatement) statementNode()       {}
func (s *IncludeStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *IncludeStatement) Line() int            { return s.Tok.Line }

// ModuleStatement — מודול שם (כותרת קובץ)
type ModuleStatement struct {
	Tok  token.Token
	Name *Identifier
}

func (s *ModuleStatement) statementNode()       {}
func (s *ModuleStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ModuleStatement) Line() int            { return s.Tok.Line }

// ExportStatement — יצא פונקציה/משתנה/מחלקה
type ExportStatement struct {
	Tok  token.Token
	Stmt Statement
}

func (s *ExportStatement) statementNode()       {}
func (s *ExportStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ExportStatement) Line() int            { return s.Tok.Line }

// ImportStatement — יבא כורים מתוך "קובץ.יוד" | יבא { סכום } מתוך "..."
type ImportStatement struct {
	Tok       token.Token
	Alias     *Identifier   // nil אם ייבוא שמות בודדים
	Names     []*Identifier // לייבוא { א, ב }
	Path      string
}

func (s *ImportStatement) statementNode()       {}
func (s *ImportStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ImportStatement) Line() int            { return s.Tok.Line }

// EnumStatement — סדרה איכות { חלש, טוב }
type EnumStatement struct {
	Tok     token.Token
	Name    *Identifier
	Members []*Identifier
}

func (s *EnumStatement) statementNode()       {}
func (s *EnumStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *EnumStatement) Line() int            { return s.Tok.Line }

type HashPair struct {
	Key   Expression
	Value Expression
}

type HashLiteral struct {
	Tok   token.Token
	Pairs []HashPair
}

func (h *HashLiteral) expressionNode()      {}
func (h *HashLiteral) TokenLiteral() string { return h.Tok.Literal }
func (h *HashLiteral) Line() int            { return h.Tok.Line }

type TryStatement struct {
	Tok       token.Token
	Body      *BlockStatement
	CatchName *Identifier
	CatchBody *BlockStatement
}

func (s *TryStatement) statementNode()       {}
func (s *TryStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *TryStatement) Line() int            { return s.Tok.Line }

type ThrowStatement struct {
	Tok   token.Token
	Value Expression
}

func (s *ThrowStatement) statementNode()       {}
func (s *ThrowStatement) TokenLiteral() string { return s.Tok.Literal }
func (s *ThrowStatement) Line() int            { return s.Tok.Line }
