package object

import (
	"fmt"
	"path/filepath"
	"strings"

	"yod/internal/ast"
	"yod/internal/code"
)

type Type string

const (
	NumberObj   Type = "מספר"
	StringObj   Type = "מחרוזת"
	BooleanObj  Type = "בוליאני"
	NullObj     Type = "ריק"
	ReturnObj   Type = "החזרה"
	ErrorObj    Type = "שגיאה"
	FunctionObj         Type = "פונקציה"
	CompiledFunctionObj Type = "פונקציה_מקומפלת"
	ClosureObj          Type = "סגירה"
	CellObj             Type = "תא"
	IteratorObj         Type = "איטרטור"
	BreakObj    Type = "עצור"
	ContinueObj Type = "המשך"
	BuiltinObj  Type = "מובנה"
	ClassObj    Type = "מחלקה"
	InstanceObj Type = "מופע"
	MethodObj   Type = "מתודה"
	ArrayObj    Type = "רשימה"
	ModuleObj   Type = "מודול"
	HashObj     Type = "מילון"
	GuiObj      Type = "רכיב_ממשק"
)

type Object interface {
	Type() Type
	Inspect() string
}

type Number struct {
	Value float64
}

func (n *Number) Type() Type { return NumberObj }
func (n *Number) Inspect() string {
	if n.Value == float64(int64(n.Value)) {
		return fmt.Sprintf("%d", int64(n.Value))
	}
	return fmt.Sprintf("%g", n.Value)
}

type String struct {
	Value string
}

func (s *String) Type() Type      { return StringObj }
func (s *String) Inspect() string { return s.Value }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() Type { return BooleanObj }
func (b *Boolean) Inspect() string {
	if b.Value {
		return "אמת"
	}
	return "שקר"
}

type Null struct{}

func (n *Null) Type() Type      { return NullObj }
func (n *Null) Inspect() string { return "ריק" }

// Nil — ערך ריק יחיד (להשוואות)
var Nil Object = &Null{}

type ReturnValue struct {
	Value Object
}

func (r *ReturnValue) Type() Type      { return ReturnObj }
func (r *ReturnValue) Inspect() string { return r.Value.Inspect() }

type Error struct {
	Message string
	Line    int    // מספר שורה (0 אם לא ידוע)
	File    string // נתיב קובץ המקור (ריק אם לא ידוע)
}

func (e *Error) Type() Type { return ErrorObj }

func (e *Error) Inspect() string {
	msg := e.Message
	// תאימות לאחור: אם Message כבר מתחיל ב־«שורה N:» — לא כופלים
	hasLineInMsg := e.Line > 0 && strings.HasPrefix(msg, "שורה ")
	switch {
	case e.File != "" && e.Line > 0 && !hasLineInMsg:
		return fmt.Sprintf("שגיאה בקובץ %s בשורה %d: %s", filepath.Base(e.File), e.Line, msg)
	case e.File != "" && hasLineInMsg:
		return fmt.Sprintf("שגיאה בקובץ %s: %s", filepath.Base(e.File), msg)
	case e.File != "":
		return fmt.Sprintf("שגיאה בקובץ %s: %s", filepath.Base(e.File), msg)
	case e.Line > 0 && !hasLineInMsg:
		return fmt.Sprintf("שגיאה בשורה %d: %s", e.Line, msg)
	default:
		return "שגיאה: " + msg
	}
}

type Break struct{}

func (b *Break) Type() Type      { return BreakObj }
func (b *Break) Inspect() string { return "עצור" }

type Continue struct{}

func (c *Continue) Type() Type      { return ContinueObj }
func (c *Continue) Inspect() string { return "המשך" }

type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() Type      { return BuiltinObj }
func (b *Builtin) Inspect() string { return "פונקציה מובנית" }

type Class struct {
	Name      string
	Parent    *Class
	Methods   map[string]*Method
	FieldDefs map[string]*FieldDef
	Env       *Environment
}

type Method struct {
	Fn         *Function
	CompiledFn *CompiledFunction // למכונה (bytecode)
	Public     bool
	Owner      *Class
}

type FieldDef struct {
	Expr    ast.Expression
	Default Object            // ערך ברירת מחדל קבוע (VM)
	Init    *CompiledFunction // אתחול מקומפל 0־ארגומנטים (VM)
	Public  bool
	Owner   *Class
}

func (c *Class) Type() Type      { return ClassObj }
func (c *Class) Inspect() string { return "מחלקה " + c.Name }

func (c *Class) FindMethod(name string) (*Method, bool) {
	for cur := c; cur != nil; cur = cur.Parent {
		if m, ok := cur.Methods[name]; ok {
			return m, true
		}
	}
	return nil, false
}

func (c *Class) FindField(name string) (*FieldDef, bool) {
	for cur := c; cur != nil; cur = cur.Parent {
		if f, ok := cur.FieldDefs[name]; ok {
			return f, true
		}
	}
	return nil, false
}

// CanAccess — גישה לשדה/מתודה פרטיים רק מאותה מחלקה שהגדירה אותם
func CanAccess(public bool, owner, caller *Class) bool {
	if public {
		return true
	}
	return caller != nil && owner != nil && caller == owner
}

type Instance struct {
	Class  *Class
	Fields map[string]Object
}

func (i *Instance) Type() Type { return InstanceObj }
func (i *Instance) Inspect() string {
	return "מופע של " + i.Class.Name
}

func (i *Instance) Get(name string) (Object, bool) {
	if val, ok := i.Fields[name]; ok {
		return val, true
	}
	if method, ok := i.Class.FindMethod(name); ok {
		return &BoundMethod{
			Instance:   i,
			Function:   method.Fn,
			CompiledFn: method.CompiledFn,
			Owner:      method.Owner,
		}, true
	}
	return nil, false
}

func (i *Instance) Set(name string, val Object) Object {
	i.Fields[name] = val
	return val
}

// ParentRef — הורה במתודה: הורה(...) לבנאי, הורה.שם למתודת הורה
type ParentRef struct {
	Instance *Instance
}

func (p *ParentRef) Type() Type      { return MethodObj }
func (p *ParentRef) Inspect() string { return "הורה" }

type BoundMethod struct {
	Instance   *Instance
	Function   *Function
	CompiledFn *CompiledFunction
	Owner      *Class // המחלקה שהגדירה את המתודה
}

func (m *BoundMethod) Type() Type      { return MethodObj }
func (m *BoundMethod) Inspect() string { return "מתודה" }

type Array struct {
	Elements []Object
}

func (a *Array) Type() Type { return ArrayObj }
func (a *Array) Inspect() string {
	parts := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		parts[i] = e.Inspect()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

type Module struct {
	Name  string
	Attrs map[string]Object
}

func (m *Module) Type() Type      { return ModuleObj }
func (m *Module) Inspect() string { return "מודול " + m.Name }

func (m *Module) Get(name string) (Object, bool) {
	v, ok := m.Attrs[name]
	return v, ok
}

// GuiWidget — רכיב ממשק (חלון/כפתור/...) עם מתודות מובנות
type GuiWidget struct {
	Kind  string
	Data  any
	Attrs map[string]Object
}

func (g *GuiWidget) Type() Type      { return GuiObj }
func (g *GuiWidget) Inspect() string { return "רכיב " + g.Kind }

func (g *GuiWidget) Get(name string) (Object, bool) {
	v, ok := g.Attrs[name]
	return v, ok
}

// InvokeFunction — נקבע ע״י המפרש כדי לאפשר קריאה לפונקציות יוד מאירועי GUI
var InvokeFunction func(fn *Function, args []Object) Object

// InvokeCallable — מפרש או מכונה: Function / Closure / CompiledFunction
var InvokeCallable func(fn Object, args []Object) Object

// Hash — מילון פשוט (לשורות SQL וכו')
type Hash struct {
	Pairs map[string]Object
}

func (h *Hash) Type() Type { return HashObj }
func (h *Hash) Inspect() string {
	parts := make([]string, 0, len(h.Pairs))
	for k, v := range h.Pairs {
		parts = append(parts, k+": "+v.Inspect())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (h *Hash) Get(name string) (Object, bool) {
	v, ok := h.Pairs[name]
	return v, ok
}

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() Type { return FunctionObj }
func (f *Function) Inspect() string {
	params := make([]string, len(f.Parameters))
	for i, p := range f.Parameters {
		params[i] = p.Value
	}
	return fmt.Sprintf("פונקציה(%s)", strings.Join(params, ", "))
}

type CompiledFunction struct {
	Instructions  code.Instructions
	NumLocals     int
	NumParameters int
}

func (f *CompiledFunction) Type() Type      { return CompiledFunctionObj }
func (f *CompiledFunction) Inspect() string { return "פונקציה מקומפלת" }

// Closure — פונקציה מקומפלת עם משתנים חופשיים (סגירה)
type Closure struct {
	Fn   *CompiledFunction
	Free []*Cell
}

func (c *Closure) Type() Type      { return ClosureObj }
func (c *Closure) Inspect() string { return "סגירה" }

// Cell — תא mutable למשתנה שנסגר עליו (שיתוף בין סגירות / סקופ חיצוני)
type Cell struct {
	Value Object
}

func (c *Cell) Type() Type      { return CellObj }
func (c *Cell) Inspect() string {
	if c.Value == nil {
		return "תא"
	}
	return c.Value.Inspect()
}

type Iterator struct {
	Values []Object
	Index  int
}

func (i *Iterator) Type() Type      { return IteratorObj }
func (i *Iterator) Inspect() string { return "איטרטור" }

type Environment struct {
	store        map[string]Object
	outer        *Environment
	BaseDir      string
	Included     map[string]bool
	CurrentClass *Class // מחלקה של המתודה הרצה כרגע (לבדיקת פרטי)
}

func NewEnvironment() *Environment {
	return &Environment{store: map[string]Object{}, Included: map[string]bool{}}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	if outer != nil {
		env.BaseDir = outer.BaseDir
		env.Included = outer.Included
		env.CurrentClass = outer.CurrentClass
	}
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

func (e *Environment) Assign(name string, val Object) (Object, bool) {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return val, true
	}
	if e.outer != nil {
		return e.outer.Assign(name, val)
	}
	return nil, false
}
