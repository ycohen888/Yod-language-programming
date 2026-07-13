package evaluator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"yod/internal/ast"
	"yod/internal/console"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/parser"
	"yod/internal/stdlib"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.BlockStatement:
		return evalBlock(node, env)
	case *ast.ExpressionStatement:
		result := Eval(node.Expr, env)
		return object.ResolveValue(result)
	case *ast.VarStatement:
		var val object.Object = NULL
		if node.Value != nil {
			val = Eval(node.Value, env)
			if isError(val) {
				return val
			}
			val = object.ResolveValue(val)
		}
		env.Set(node.Name.Value, val)
		return val
	case *ast.AssignStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		val = object.ResolveValue(val)
		if _, ok := env.Assign(node.Name.Value, val); !ok {
			// כמו PHP: השמה יוצרת משתנה אם לא קיים
			env.Set(node.Name.Value, val)
		}
		return val
	case *ast.ReturnStatement:
		var val object.Object = NULL
		if node.Value != nil {
			val = Eval(node.Value, env)
			if isError(val) {
				return val
			}
			val = object.ResolveValue(val)
		}
		return &object.ReturnValue{Value: val}
	case *ast.IfStatement:
		return evalIf(node, env)
	case *ast.SwitchStatement:
		return evalSwitch(node, env)
	case *ast.WhileStatement:
		return evalWhile(node, env)
	case *ast.ForInStatement:
		return evalForIn(node, env)
	case *ast.BreakStatement:
		return &object.Break{}
	case *ast.ContinueStatement:
		return &object.Continue{}
	case *ast.NumberLiteral:
		return &object.Number{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.BooleanLiteral:
		return nativeBool(node.Value)
	case *ast.NullLiteral:
		return NULL
	case *ast.InfixExpression:
		if node.Operator == "וגם" || node.Operator == "&&" {
			left := object.ResolveValue(Eval(node.Left, env))
			if isError(left) {
				return left
			}
			if !isTruthy(left) {
				return left
			}
			return object.ResolveValue(Eval(node.Right, env))
		}
		if node.Operator == "או" || node.Operator == "||" {
			left := object.ResolveValue(Eval(node.Left, env))
			if isError(left) {
				return left
			}
			if isTruthy(left) {
				return left
			}
			return object.ResolveValue(Eval(node.Right, env))
		}
		if node.Operator == "??" {
			left := object.ResolveValue(Eval(node.Left, env))
			if isError(left) {
				return left
			}
			if _, ok := left.(*object.Null); ok {
				return object.ResolveValue(Eval(node.Right, env))
			}
			return left
		}
		left := object.ResolveValue(Eval(node.Left, env))
		if isError(left) {
			return left
		}
		right := object.ResolveValue(Eval(node.Right, env))
		if isError(right) {
			return right
		}
		return evalInfix(node.Operator, left, right, node.Line())
	case *ast.PrefixExpression:
		right := object.ResolveValue(Eval(node.Right, env))
		if isError(right) {
			return right
		}
		return evalPrefix(node.Operator, right, node.Line())
	case *ast.Identifier:
		return evalIdent(node, env)
	case *ast.FunctionLiteral:
		return &object.Function{Parameters: node.Parameters, Body: node.Body, Env: env}
	case *ast.ClassStatement:
		return evalClass(node, env)
	case *ast.IncludeStatement:
		return evalInclude(node, env)
	case *ast.TryStatement:
		return evalTry(node, env)
	case *ast.ThrowStatement:
		return evalThrow(node, env)
	case *ast.ThisLiteral:
		if val, ok := env.Get("זה"); ok {
			return val
		}
		return newError(node.Line(), "שימוש ב־זה מחוץ למתודה")
	case *ast.ParentLiteral:
		return evalParentLiteral(node, env)
	case *ast.MemberExpression:
		return evalMember(node, env)
	case *ast.NewExpression:
		return evalNew(node, env)
	case *ast.AssignExpression:
		return evalAssignExpr(node, env)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	case *ast.IndexExpression:
		return evalIndex(node, env)
	case *ast.CallExpression:
		fn := Eval(node.Function, env)
		if isError(fn) {
			return fn
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(fn, args, node.Line())
	}
	return newError(1, "צומת לא נתמך עדיין")
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object = NULL
	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
		switch r := result.(type) {
		case *object.ReturnValue:
			return r.Value
		case *object.Error:
			return r
		}
	}
	return result
}

func evalBlock(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object = NULL
	for _, stmt := range block.Statements {
		result = Eval(stmt, env)
		if result != nil {
			t := result.Type()
			if t == object.ReturnObj || t == object.ErrorObj || t == object.BreakObj || t == object.ContinueObj {
				return result
			}
		}
	}
	return result
}

func evalIf(stmt *ast.IfStatement, env *object.Environment) object.Object {
	cond := object.ResolveValue(Eval(stmt.Condition, env))
	if isError(cond) {
		return cond
	}
	if isTruthy(cond) {
		return Eval(stmt.Consequence, env)
	}
	if stmt.Alternative != nil {
		return Eval(stmt.Alternative, env)
	}
	return NULL
}

func evalSwitch(stmt *ast.SwitchStatement, env *object.Environment) object.Object {
	val := Eval(stmt.Value, env)
	if isError(val) {
		return val
	}
	for _, cas := range stmt.Cases {
		for _, vExpr := range cas.Values {
			cv := Eval(vExpr, env)
			if isError(cv) {
				return cv
			}
			if equalObjects(val, cv) {
				return Eval(cas.Body, env)
			}
		}
	}
	if stmt.Default != nil {
		return Eval(stmt.Default, env)
	}
	return NULL
}

func powFloat(a, b float64) float64 {
	return math.Pow(a, b)
}

func evalWhile(stmt *ast.WhileStatement, env *object.Environment) object.Object {
	var result object.Object = NULL
	for {
		cond := object.ResolveValue(Eval(stmt.Condition, env))
		if isError(cond) {
			return cond
		}
		if !isTruthy(cond) {
			break
		}
		result = Eval(stmt.Body, env)
		if result != nil {
			switch result.Type() {
			case object.ReturnObj, object.ErrorObj:
				return result
			case object.BreakObj:
				return NULL
			case object.ContinueObj:
				continue
			}
		}
	}
	return result
}

func evalForIn(stmt *ast.ForInStatement, env *object.Environment) object.Object {
	iterable := Eval(stmt.Iterable, env)
	if isError(iterable) {
		return iterable
	}
	var elements []object.Object
	switch it := iterable.(type) {
	case *object.Array:
		elements = it.Elements
	case *object.Hash:
		elements = make([]object.Object, 0, len(it.Pairs))
		for k := range it.Pairs {
			elements = append(elements, &object.String{Value: k})
		}
	case *object.String:
		runes := []rune(it.Value)
		elements = make([]object.Object, len(runes))
		for i, r := range runes {
			elements[i] = &object.String{Value: string(r)}
		}
	default:
		return newError(stmt.Line(), "עבור...בתוך עובד על רשימה, מילון או מחרוזת")
	}
	var result object.Object = NULL
	for _, el := range elements {
		env.Set(stmt.Name.Value, el)
		result = Eval(stmt.Body, env)
		if result != nil {
			switch result.Type() {
			case object.ReturnObj, object.ErrorObj:
				return result
			case object.BreakObj:
				return NULL
			case object.ContinueObj:
				continue
			}
		}
	}
	return result
}

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	pairs := map[string]object.Object{}
	for _, pair := range node.Pairs {
		keyObj := Eval(pair.Key, env)
		if isError(keyObj) {
			return keyObj
		}
		key, ok := keyObj.(*object.String)
		if !ok {
			return newError(node.Line(), "מפתח במילון חייב להיות מחרוזת")
		}
		val := Eval(pair.Value, env)
		if isError(val) {
			return val
		}
		pairs[key.Value] = val
	}
	return &object.Hash{Pairs: pairs}
}

func evalIndex(node *ast.IndexExpression, env *object.Environment) object.Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}
	index := Eval(node.Index, env)
	if isError(index) {
		return index
	}
	switch left := left.(type) {
	case *object.Array:
		idx, ok := index.(*object.Number)
		if !ok {
			return newError(node.Line(), "אינדקס של רשימה חייב להיות מספר")
		}
		i := int(idx.Value)
		if i < 0 || i >= len(left.Elements) {
			return newError(node.Line(), fmt.Sprintf("אינדקס מחוץ לטווח: %d", i))
		}
		return left.Elements[i]
	case *object.Hash:
		key, ok := index.(*object.String)
		if !ok {
			return newError(node.Line(), "אינדקס של מילון חייב להיות מחרוזת")
		}
		val, ok := left.Get(key.Value)
		if !ok {
			return NULL
		}
		return val
	case *object.String:
		idx, ok := index.(*object.Number)
		if !ok {
			return newError(node.Line(), "אינדקס של מחרוזת חייב להיות מספר")
		}
		runes := []rune(left.Value)
		i := int(idx.Value)
		if i < 0 || i >= len(runes) {
			return newError(node.Line(), fmt.Sprintf("אינדקס מחוץ לטווח: %d", i))
		}
		return &object.String{Value: string(runes[i])}
	default:
		return newError(node.Line(), "אינדקס לא נתמך על "+string(left.Type()))
	}
}

func evalPrefix(op string, right object.Object, line int) object.Object {
	switch op {
	case "!":
		return nativeBool(!isTruthy(right))
	case "-":
		n, ok := right.(*object.Number)
		if !ok {
			return newError(line, "האופרטור - עובד רק על מספרים")
		}
		return &object.Number{Value: -n.Value}
	default:
		return newError(line, "אופרטור לא מוכר: "+op)
	}
}

func evalInfix(op string, left, right object.Object, line int) object.Object {
	switch {
	case left.Type() == object.NumberObj && right.Type() == object.NumberObj:
		return evalNumberInfix(op, left.(*object.Number), right.(*object.Number), line)
	case op == "+" && (left.Type() == object.StringObj || right.Type() == object.StringObj):
		return &object.String{Value: left.Inspect() + right.Inspect()}
	case op == "==":
		return nativeBool(equalObjects(left, right))
	case op == "!=":
		return nativeBool(!equalObjects(left, right))
	default:
		return newError(line, fmt.Sprintf("לא ניתן לבצע %s בין %s ל־%s", op, left.Type(), right.Type()))
	}
}

func equalObjects(a, b object.Object) bool {
	switch a := a.(type) {
	case *object.Number:
		b, ok := b.(*object.Number)
		return ok && a.Value == b.Value
	case *object.String:
		b, ok := b.(*object.String)
		return ok && a.Value == b.Value
	case *object.Boolean:
		b, ok := b.(*object.Boolean)
		return ok && a.Value == b.Value
	case *object.Null:
		_, ok := b.(*object.Null)
		return ok
	default:
		return a == b
	}
}

func evalNumberInfix(op string, left, right *object.Number, line int) object.Object {
	switch op {
	case "+":
		return &object.Number{Value: left.Value + right.Value}
	case "-":
		return &object.Number{Value: left.Value - right.Value}
	case "*":
		return &object.Number{Value: left.Value * right.Value}
	case "**":
		return &object.Number{Value: powFloat(left.Value, right.Value)}
	case "/":
		if right.Value == 0 {
			return newError(line, "חילוק באפס")
		}
		return &object.Number{Value: left.Value / right.Value}
	case "%":
		if right.Value == 0 {
			return newError(line, "מודולו באפס")
		}
		return &object.Number{Value: float64(int64(left.Value) % int64(right.Value))}
	case "<":
		return nativeBool(left.Value < right.Value)
	case ">":
		return nativeBool(left.Value > right.Value)
	case "<=":
		return nativeBool(left.Value <= right.Value)
	case ">=":
		return nativeBool(left.Value >= right.Value)
	case "==":
		return nativeBool(left.Value == right.Value)
	case "!=":
		return nativeBool(left.Value != right.Value)
	default:
		return newError(line, "אופרטור לא מוכר בין מספרים: "+op)
	}
}

func evalIdent(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}
	return newError(node.Line(), fmt.Sprintf("המשתנה %q לא מוגדר", node.Value))
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	result := make([]object.Object, 0, len(exps))
	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		evaluated = object.ResolveValue(evaluated)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func applyFunction(fn object.Object, args []object.Object, line int) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		return callUserFunction(fn, args, nil, nil, line)
	case *object.BoundMethod:
		return callUserFunction(fn.Function, args, fn.Instance, fn.Owner, line)
	case *object.BoundBuiltin:
		return fn.Call(args...)
	case *object.ParentRef:
		return callParentConstructor(fn.Instance, args, line)
	case *object.Builtin:
		return fn.Fn(args...)
	default:
		return newError(line, "זה לא פונקציה: "+string(fn.Type()))
	}
}

func callParentConstructor(inst *object.Instance, args []object.Object, line int) object.Object {
	if inst.Class.Parent == nil {
		return newError(line, "אין מחלקת הורה לקריאה")
	}
	ctor, ok := inst.Class.Parent.FindMethod("בנאי")
	if !ok {
		if len(args) > 0 {
			return newError(line, "למחלקת ההורה אין בנאי")
		}
		return NULL
	}
	return callUserFunction(ctor.Fn, args, inst, ctor.Owner, line)
}

func evalParentLiteral(node *ast.ParentLiteral, env *object.Environment) object.Object {
	val, ok := env.Get("זה")
	if !ok {
		return newError(node.Line(), "שימוש ב־הורה מחוץ למתודה")
	}
	inst, ok := val.(*object.Instance)
	if !ok {
		return newError(node.Line(), "שימוש ב־הורה מחוץ למתודה")
	}
	if inst.Class.Parent == nil {
		return newError(node.Line(), fmt.Sprintf("למחלקה %q אין הורה", inst.Class.Name))
	}
	return &object.ParentRef{Instance: inst}
}

func callUserFunction(fn *object.Function, args []object.Object, this *object.Instance, owner *object.Class, line int) object.Object {
	nParams := len(fn.Parameters)
	nRequired := 0
	seenDefault := false
	for _, p := range fn.Parameters {
		if p == nil {
			continue
		}
		if p.Default != nil {
			seenDefault = true
		} else {
			if seenDefault {
				return newError(line, "פרמטר בלי ברירת מחדל אחרי פרמטר עם ברירת מחדל")
			}
			nRequired++
		}
	}
	if len(args) < nRequired || len(args) > nParams {
		if nRequired == nParams {
			return newError(line, fmt.Sprintf("מספר ארגומנטים שגוי: ציפיתי ל־%d קיבלתי %d", nParams, len(args)))
		}
		return newError(line, fmt.Sprintf("מספר ארגומנטים שגוי: ציפיתי ל־%d…%d קיבלתי %d", nRequired, nParams, len(args)))
	}
	extended := object.NewEnclosedEnvironment(fn.Env)
	if this != nil {
		extended.Set("זה", this)
	}
	if owner != nil {
		extended.CurrentClass = owner
	}
	for i, param := range fn.Parameters {
		if param == nil || param.Name == nil {
			continue
		}
		var val object.Object
		if i < len(args) {
			val = args[i]
		} else if param.Default != nil {
			val = Eval(param.Default, extended)
			if isError(val) {
				return val
			}
		} else {
			val = NULL
		}
		extended.Set(param.Name.Value, val)
	}
	evaluated := Eval(fn.Body, extended)
	return unwrapReturn(evaluated)
}

func evalClass(node *ast.ClassStatement, env *object.Environment) object.Object {
	class := &object.Class{
		Name:      node.Name.Value,
		Methods:   map[string]*object.Method{},
		FieldDefs: map[string]*object.FieldDef{},
		Env:       env,
	}
	if node.Parent != nil {
		raw, ok := env.Get(node.Parent.Value)
		if !ok {
			return newError(node.Line(), fmt.Sprintf("מחלקת ההורה %q לא מוגדרת", node.Parent.Value))
		}
		parent, ok := raw.(*object.Class)
		if !ok {
			return newError(node.Line(), fmt.Sprintf("%q אינה מחלקה", node.Parent.Value))
		}
		class.Parent = parent
	}
	if node.Body != nil {
		for _, stmt := range node.Body.Statements {
			switch s := stmt.(type) {
			case *ast.VarStatement:
				if fl, ok := s.Value.(*ast.FunctionLiteral); ok {
					vis := s.Visibility
					if fl.Visibility == ast.VisPrivate {
						vis = ast.VisPrivate
					}
					class.Methods[s.Name.Value] = &object.Method{
						Fn: &object.Function{
							Parameters: fl.Parameters,
							Body:       fl.Body,
							Env:        env,
						},
						Public: vis != ast.VisPrivate,
						Owner:  class,
					}
				} else {
					class.FieldDefs[s.Name.Value] = &object.FieldDef{
						Expr:   s.Value,
						Public: s.Visibility != ast.VisPrivate,
						Owner:  class,
					}
				}
			case *ast.FunctionLiteral:
				name := ""
				if s.Name != nil {
					name = s.Name.Value
				}
				class.Methods[name] = &object.Method{
					Fn: &object.Function{
						Parameters: s.Parameters,
						Body:       s.Body,
						Env:        env,
					},
					Public: s.Visibility != ast.VisPrivate,
					Owner:  class,
				}
			}
		}
	}
	env.Set(node.Name.Value, class)
	return class
}

func canAccess(public bool, owner, caller *object.Class) bool {
	return object.CanAccess(public, owner, caller)
}

func evalMember(node *ast.MemberExpression, env *object.Environment) object.Object {
	obj := Eval(node.Object, env)
	if isError(obj) {
		return obj
	}
	name := node.Property.Value
	caller := env.CurrentClass
	switch obj := obj.(type) {
	case *object.Instance:
		if val, ok := obj.Fields[name]; ok {
			if fd, ok := obj.Class.FindField(name); ok {
				if !canAccess(fd.Public, fd.Owner, caller) {
					return newError(node.Line(), fmt.Sprintf("השדה %q הוא פרטי", name))
				}
			}
			return val
		}
		if method, ok := obj.Class.FindMethod(name); ok {
			if !canAccess(method.Public, method.Owner, caller) {
				return newError(node.Line(), fmt.Sprintf("המתודה %q היא פרטית", name))
			}
			return &object.BoundMethod{Instance: obj, Function: method.Fn, Owner: method.Owner}
		}
		return newError(node.Line(), fmt.Sprintf("למופע אין שדה/מתודה בשם %q", name))
	case *object.ParentRef:
		if obj.Instance.Class.Parent == nil {
			return newError(node.Line(), "אין מחלקת הורה")
		}
		method, ok := obj.Instance.Class.Parent.FindMethod(name)
		if !ok {
			return newError(node.Line(), fmt.Sprintf("להורה אין מתודה בשם %q", name))
		}
		if !canAccess(method.Public, method.Owner, caller) {
			return newError(node.Line(), fmt.Sprintf("המתודה %q היא פרטית", name))
		}
		return &object.BoundMethod{Instance: obj.Instance, Function: method.Fn, Owner: method.Owner}
	case *object.Module:
		val, ok := obj.Get(name)
		if !ok {
			return newError(node.Line(), fmt.Sprintf("במודול אין %q", name))
		}
		return val
	case *object.GuiWidget:
		val, ok := obj.Get(name)
		if !ok {
			return newError(node.Line(), fmt.Sprintf("לרכיב אין %q", name))
		}
		return val
	case *object.Hash:
		if m := object.LookupMethod(obj, name); m != nil {
			return m
		}
		val, ok := obj.Get(name)
		if !ok {
			return newError(node.Line(), fmt.Sprintf("במילון אין מפתח %q", name))
		}
		return val
	case *object.Array:
		if m := object.LookupMethod(obj, name); m != nil {
			return m
		}
		return newError(node.Line(), fmt.Sprintf("לרשימה אין מתודה בשם %q", name))
	case *object.String, *object.Number, *object.Boolean, *object.Null:
		if m := object.LookupMethod(obj, name); m != nil {
			return m
		}
		return newError(node.Line(), fmt.Sprintf("ל־%s אין מתודה בשם %q", obj.Type(), name))
	default:
		return newError(node.Line(), "גישה לנקודה לא נתמכת על "+string(obj.Type()))
	}
}

func evalNew(node *ast.NewExpression, env *object.Environment) object.Object {
	raw, ok := env.Get(node.Name.Value)
	if !ok {
		return newError(node.Line(), fmt.Sprintf("המחלקה %q לא מוגדרת", node.Name.Value))
	}
	class, ok := raw.(*object.Class)
	if !ok {
		return newError(node.Line(), fmt.Sprintf("%q אינה מחלקה", node.Name.Value))
	}

	inst := &object.Instance{
		Class:  class,
		Fields: map[string]object.Object{},
	}

	for _, fi := range collectFieldOrder(class) {
		if fi.expr == nil {
			inst.Fields[fi.name] = NULL
			continue
		}
		val := Eval(fi.expr, env)
		if isError(val) {
			return val
		}
		inst.Fields[fi.name] = val
	}

	args := evalExpressions(node.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}

	if ctor, ok := class.Methods["בנאי"]; ok {
		ret := callUserFunction(ctor.Fn, args, inst, ctor.Owner, node.Line())
		if isError(ret) {
			return ret
		}
	} else if ctor, ok := class.FindMethod("בנאי"); ok {
		ret := callUserFunction(ctor.Fn, args, inst, ctor.Owner, node.Line())
		if isError(ret) {
			return ret
		}
	} else if len(args) > 0 {
		return newError(node.Line(), "למחלקה אין בנאי, אבל הועברו ארגומנטים")
	}
	return inst
}

type fieldInit struct {
	name string
	expr ast.Expression
}

func collectFieldOrder(class *object.Class) []fieldInit {
	var chain []*object.Class
	for c := class; c != nil; c = c.Parent {
		chain = append(chain, c)
	}
	out := []fieldInit{}
	seen := map[string]bool{}
	for i := len(chain) - 1; i >= 0; i-- {
		c := chain[i]
		for name, fd := range c.FieldDefs {
			if seen[name] {
				for j := range out {
					if out[j].name == name {
						out[j].expr = fd.Expr
						break
					}
				}
			} else {
				seen[name] = true
				out = append(out, fieldInit{name: name, expr: fd.Expr})
			}
		}
	}
	return out
}

func evalAssignExpr(node *ast.AssignExpression, env *object.Environment) object.Object {
	val := object.ResolveValue(Eval(node.Value, env))
	if isError(val) {
		return val
	}
	switch left := node.Left.(type) {
	case *ast.Identifier:
		if _, ok := env.Assign(left.Value, val); !ok {
			env.Set(left.Value, val)
		}
		return val
	case *ast.MemberExpression:
		obj := Eval(left.Object, env)
		if isError(obj) {
			return obj
		}
		switch obj := obj.(type) {
		case *object.Instance:
			name := left.Property.Value
			if fd, ok := obj.Class.FindField(name); ok {
				if !canAccess(fd.Public, fd.Owner, env.CurrentClass) {
					return newError(node.Line(), fmt.Sprintf("השדה %q הוא פרטי", name))
				}
			}
			return obj.Set(name, val)
		case *object.Hash:
			obj.Pairs[left.Property.Value] = val
			return val
		default:
			return newError(node.Line(), "השמה לנקודה אפשרית רק על מופע מחלקה או מילון")
		}
	case *ast.IndexExpression:
		obj := Eval(left.Left, env)
		if isError(obj) {
			return obj
		}
		idxObj := Eval(left.Index, env)
		if isError(idxObj) {
			return idxObj
		}
		switch obj := obj.(type) {
		case *object.Array:
			idx, ok := idxObj.(*object.Number)
			if !ok {
				return newError(node.Line(), "אינדקס חייב להיות מספר")
			}
			i := int(idx.Value)
			if i < 0 || i >= len(obj.Elements) {
				return newError(node.Line(), fmt.Sprintf("אינדקס מחוץ לטווח: %d", i))
			}
			obj.Elements[i] = val
			return val
		case *object.Hash:
			key, ok := idxObj.(*object.String)
			if !ok {
				return newError(node.Line(), "מפתח מילון חייב להיות מחרוזת")
			}
			obj.Pairs[key.Value] = val
			return val
		default:
			return newError(node.Line(), "השמה באינדקס אפשרית רק על רשימה או מילון")
		}
	default:
		return newError(node.Line(), "יעד השמה לא חוקי")
	}
}

func unwrapReturn(obj object.Object) object.Object {
	if ret, ok := obj.(*object.ReturnValue); ok {
		return ret.Value
	}
	return obj
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL, FALSE:
		return false
	case TRUE:
		return true
	default:
		if n, ok := obj.(*object.Number); ok {
			return n.Value != 0
		}
		if s, ok := obj.(*object.String); ok {
			return s.Value != ""
		}
		if b, ok := obj.(*object.Boolean); ok {
			return b.Value
		}
		return true
	}
}

func nativeBool(v bool) *object.Boolean {
	if v {
		return TRUE
	}
	return FALSE
}

func isError(obj object.Object) bool {
	return obj != nil && obj.Type() == object.ErrorObj
}

func newError(line int, msg string) *object.Error {
	return &object.Error{
		Message: fmt.Sprintf("שורה %d: %s", line, msg),
		Line:    line,
		File:    currentSourceFile(),
	}
}

var builtins = map[string]*object.Builtin{
	"הדפס": {
		Fn: func(args ...object.Object) object.Object {
			parts := make([]any, 0, len(args))
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			if len(parts) == 0 {
				console.Println()
			} else {
				console.Println(parts...)
			}
			return NULL
		},
	},
	"קלט": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) > 0 {
				console.Print(args[0].Inspect())
			}
			var s string
			fmt.Scanln(&s)
			return &object.String{Value: s}
		},
	},
	"אורך": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: "שורה ?: אורך מצפה לארגומנט אחד"}
			}
			switch a := args[0].(type) {
			case *object.Array:
				return &object.Number{Value: float64(len(a.Elements))}
			case *object.String:
				return &object.Number{Value: float64(len([]rune(a.Value)))}
			case *object.Hash:
				return &object.Number{Value: float64(len(a.Pairs))}
			default:
				return &object.Error{Message: "שורה ?: אורך עובד על רשימה, מילון או מחרוזת"}
			}
		},
	},
	"הוסף": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: "שורה ?: הוסף מצפה לרשימה וערך"}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Message: "שורה ?: הארגומנט הראשון של הוסף חייב להיות רשימה"}
			}
			for _, v := range args[1:] {
				arr.Elements = append(arr.Elements, v)
			}
			return arr
		},
	},
	"למספר": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: "שורה ?: למספר מצפה לארגומנט אחד"}
			}
			switch a := args[0].(type) {
			case *object.Number:
				return a
			case *object.String:
				s := strings.TrimSpace(a.Value)
				n, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("שורה ?: לא הצלחתי להמיר %q למספר", a.Value)}
				}
				return &object.Number{Value: n}
			case *object.Boolean:
				if a.Value {
					return &object.Number{Value: 1}
				}
				return &object.Number{Value: 0}
			default:
				return &object.Error{Message: "שורה ?: למספר לא תומך בטיפוס " + string(a.Type())}
			}
		},
	},
	"למחרוזת": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: "שורה ?: למחרוזת מצפה לארגומנט אחד"}
			}
			return &object.String{Value: args[0].Inspect()}
		},
	},
	"סוג": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: "סוג מצפה לארגומנט אחד"}
			}
			return &object.String{Value: string(args[0].Type())}
		},
	},
	"טווח": {
		Fn: object.Range,
	},
	"אקראי": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{Message: "אקראי מצפה ל־0 ארגומנטים"}
			}
			return object.Random01()
		},
	},
	"אקראי_בין": {
		Fn: object.RandomBetween,
	},
}

func NewGlobalEnv(baseDir string) *object.Environment {
	env := object.NewEnvironment()
	env.BaseDir = baseDir
	stdlib.SetAppBaseDir(baseDir)
	env.Included = map[string]bool{}
	for name, b := range builtins {
		env.Set(name, b)
	}
	object.InvokeFunction = func(fn *object.Function, args []object.Object) object.Object {
		return callUserFunction(fn, args, nil, nil, 1)
	}
	object.InvokeCallable = func(fn object.Object, args []object.Object) object.Object {
		switch f := fn.(type) {
		case *object.Function:
			return callUserFunction(f, args, nil, nil, 1)
		case *object.Builtin:
			return f.Fn(args...)
		default:
			return &object.Error{Message: "מצופה לפונקציה, קיבל " + string(fn.Type())}
		}
	}
	return env
}

func evalInclude(node *ast.IncludeStatement, env *object.Environment) object.Object {
	path := node.Path

	// ספרייה מובנית
	if mod, err := stdlib.LoadBuiltin(path); err == nil {
		m := mod.(*object.Module)
		env.Set(m.Name, m)
		return m
	}

	// קובץ .יוד של משתמש
	full := path
	if !filepath.IsAbs(full) {
		base := env.BaseDir
		if base == "" {
			base = "."
		}
		full = filepath.Join(base, path)
	}
	full = filepath.Clean(full)

	if env.Included[full] {
		return NULL // כבר נכלל
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return newError(node.Line(), fmt.Sprintf("לא הצלחתי לכלול את %q: %v", path, err))
	}
	env.Included[full] = true

	prevBase := env.BaseDir
	env.BaseDir = filepath.Dir(full)
	defer func() { env.BaseDir = prevBase }()

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		return &object.Error{
			Message: errs[0],
			Line:    extractLineNum(errs[0]),
			File:    full,
		}
	}
	PushSourceFile(full)
	defer PopSourceFile()
	result := Eval(program, env)
	if isError(result) {
		if e, ok := result.(*object.Error); ok && e.File == "" {
			e.File = full
		}
		return result
	}
	return NULL
}

func evalTry(stmt *ast.TryStatement, env *object.Environment) object.Object {
	result := Eval(stmt.Body, env)
	if !isError(result) {
		return result
	}
	err := result.(*object.Error)
	catchEnv := object.NewEnclosedEnvironment(env)
	catchEnv.Set(stmt.CatchName.Value, &object.String{Value: err.Message})
	return Eval(stmt.CatchBody, catchEnv)
}

func evalThrow(stmt *ast.ThrowStatement, env *object.Environment) object.Object {
	val := Eval(stmt.Value, env)
	if isError(val) {
		return val
	}
	msg := val.Inspect()
	if s, ok := val.(*object.String); ok {
		msg = s.Value
	}
	return &object.Error{Message: fmt.Sprintf("שורה %d: %s", stmt.Line(), msg)}
}
