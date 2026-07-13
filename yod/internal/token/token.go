package token

import "sort"

type Type string

const (
	Illegal Type = "Illegal"
	EOF     Type = "EOF"

	Ident  Type = "Ident"
	Number Type = "Number"
	String Type = "String"

	Assign   Type = "="
	Plus     Type = "+"
	Minus    Type = "-"
	Asterisk Type = "*"
	Slash    Type = "/"
	Percent  Type = "%"

	PlusAssign     Type = "+="
	MinusAssign    Type = "-="
	AsteriskAssign Type = "*="
	SlashAssign    Type = "/="
	PercentAssign  Type = "%="

	Bang Type = "!"
	Eq   Type = "=="
	NotEq Type = "!="
	Lt   Type = "<"
	Gt   Type = ">"
	LtEq Type = "<="
	GtEq Type = ">="

	And Type = "וגם" // גם &&
	Or  Type = "או"  // גם ||
	Not Type = "לא"  // קידומת, כמו !

	Comma    Type = ","
	Semicolon Type = ";"
	LParen   Type = "("
	RParen   Type = ")"
	LBrace   Type = "{"
	RBrace   Type = "}"
	LBracket Type = "["
	RBracket Type = "]"
	Dot      Type = "."
	Colon    Type = ":"

	// מילות מפתח בעברית
	Var      Type = "משתנה"
	If       Type = "אם"
	Else     Type = "אחרת"
	ElseIf   Type = "אחרת_אם"
	While    Type = "כל_עוד"
	For      Type = "עבור"
	Break    Type = "עצור"
	Continue Type = "המשך"
	Function Type = "פונקציה"
	Return   Type = "החזר"
	True     Type = "אמת"
	False    Type = "שקר"
	Null     Type = "ריק"
	In       Type = "בתוך"
	Class    Type = "מחלקה"
	New      Type = "חדש"
	This     Type = "זה"
	Extends  Type = "מרחיב"
	Parent   Type = "הורה"
	Private  Type = "פרטי"
	Public   Type = "ציבורי"
	Include  Type = "כלול"
	Try      Type = "נסה"
	Catch    Type = "תפוס"
	Throw    Type = "זרוק"
	End      Type = "סוף" // סגירת בלוק (במקום })
	Switch   Type = "בחר"
	Case     Type = "מקרה"
	Default  Type = "ברירת_מחדל"

	Power      Type = "**"
	NullCoalesce Type = "??"
)

type Token struct {
	Type    Type
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]Type{
	"משתנה":   Var,
	"אם":      If,
	"אחרת":    Else,
	"אחרת_אם": ElseIf,
	"כל_עוד":  While,
	"עבור":    For,
	"עצור":    Break,
	"המשך":    Continue,
	"פונקציה": Function,
	"החזר":    Return,
	"אמת":     True,
	"שקר":     False,
	"ריק":     Null,
	"בתוך":    In,
	"מחלקה":   Class,
	"חדש":     New,
	"זה":      This,
	"מרחיב":   Extends,
	"הורה":    Parent,
	"פרטי":    Private,
	"ציבורי":  Public,
	"כלול":    Include,
	"נסה":     Try,
	"תפוס":    Catch,
	"זרוק":         Throw,
	"סוף":          End,
	"וגם":          And,
	"או":           Or,
	"לא":           Not,
	"בחר":          Switch,
	"מקרה":         Case,
	"ברירת_מחדל": Default,
}

func LookupIdent(ident string) Type {
	if t, ok := keywords[ident]; ok {
		return t
	}
	return Ident
}

// Keywords מחזיר את כל מילות המפתח בעברית (ממוין)
func Keywords() []string {
	out := make([]string, 0, len(keywords))
	for k := range keywords {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// IsKeyword — האם סוג הטוקן הוא מילת מפתח בעברית
func IsKeyword(t Type) bool {
	_, ok := keywords[string(t)]
	return ok
}
