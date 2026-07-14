package compiler

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"yod/internal/ast"
	"yod/internal/code"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/parser"
	"yod/internal/stdlib"
)

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

type LoopContext struct {
	continuePos int
	breaks      []int
}

type Compiler struct {
	constants   []object.Object
	symbolTable *SymbolTable
	scopes      []CompilationScope
	scopeIndex  int
	loops       []LoopContext
	classes     map[string]*object.Class
	baseDir     string
	sourceFile  string
	included    map[string]bool
}

func New(baseDir string) *Compiler {
	if baseDir == "" {
		baseDir = "."
	}
	stdlib.SetAppBaseDir(baseDir)
	mainScope := CompilationScope{}
	return &Compiler{
		symbolTable: NewSymbolTable(),
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
		loops:       []LoopContext{},
		classes:     map[string]*object.Class{},
		baseDir:     baseDir,
		included:    map[string]bool{},
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			if err := c.Compile(s); err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(code.OpPop)
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			if err := c.Compile(s); err != nil {
				return err
			}
		}
	case *ast.VarStatement:
		if node.Value != nil {
			if err := c.Compile(node.Value); err != nil {
				return err
			}
		} else {
			c.emit(code.OpNull)
		}
		symbol := c.symbolTable.Define(node.Name.Value)
		c.loadSet(symbol)
	case *ast.AssignStatement:
		if err := c.Compile(node.Value); err != nil {
			return err
		}
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			symbol = c.symbolTable.Define(node.Name.Value)
		}
		c.loadSet(symbol)
	case *ast.ReturnStatement:
		if node.Value != nil {
			if err := c.Compile(node.Value); err != nil {
				return err
			}
		} else {
			c.emit(code.OpNull)
		}
		c.emit(code.OpReturnValue)
	case *ast.FunctionLiteral:
		compiledFn, freeSymbols, err := c.compileFunction(node, false)
		if err != nil {
			return err
		}
		fnIndex := c.addConstant(compiledFn)
		for _, s := range freeSymbols {
			c.loadFreeSource(s)
		}
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))
	case *ast.ClassStatement:
		if err := c.compileClass(node); err != nil {
			return err
		}
	case *ast.IncludeStatement:
		if err := c.compileInclude(node); err != nil {
			return err
		}
	case *ast.ModuleStatement:
		return nil // שם מודול — רלוונטי בעיקר במפרש / יבא
	case *ast.ExportStatement:
		return c.Compile(node.Stmt)
	case *ast.ImportStatement:
		return fmt.Errorf("יבא לא נתמך במכונה עדיין")
	case *ast.EnumStatement:
		return fmt.Errorf("סדרה לא נתמכת במכונה עדיין")
	case *ast.NewExpression:
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			return fmt.Errorf("המחלקה %q לא מוגדרת", node.Name.Value)
		}
		c.loadGet(symbol)
		for _, a := range node.Arguments {
			if err := c.Compile(a); err != nil {
				return err
			}
		}
		c.emit(code.OpNew, len(node.Arguments))
	case *ast.ThisLiteral:
		symbol, ok := c.symbolTable.Resolve("זה")
		if !ok {
			return fmt.Errorf("'זה' מותר רק בתוך מתודה")
		}
		c.loadGet(symbol)
	case *ast.ParentLiteral:
		symbol, ok := c.symbolTable.Resolve("זה")
		if !ok {
			return fmt.Errorf("'הורה' מותר רק בתוך מתודה")
		}
		c.loadGet(symbol)
		c.emit(code.OpParentRef)
	case *ast.IfStatement:
		if err := c.Compile(node.Condition); err != nil {
			return err
		}
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)
		if err := c.Compile(node.Consequence); err != nil {
			return err
		}
		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		}
		jumpPos := c.emit(code.OpJump, 9999)
		afterConsequence := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequence)

		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			if err := c.Compile(node.Alternative); err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}
		afterAlternative := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternative)
		c.emit(code.OpPop)

	case *ast.SwitchStatement:
		if err := c.Compile(node.Value); err != nil {
			return err
		}
		var endJumps []int
		for _, cas := range node.Cases {
			if len(cas.Values) == 0 {
				continue
			}
			var matchJumps []int
			for i, v := range cas.Values {
				c.emit(code.OpDup)
				if err := c.Compile(v); err != nil {
					return err
				}
				c.emit(code.OpEqual)
				if i < len(cas.Values)-1 {
					matchJumps = append(matchJumps, c.emit(code.OpJumpTruthy, 9999))
				}
			}
			// ערך אחרון: אם לא שווה — למקרה הבא
			nextCase := c.emit(code.OpJumpNotTruthy, 9999)
			matchedPos := len(c.currentInstructions())
			for _, j := range matchJumps {
				c.changeOperand(j, matchedPos)
			}
			c.emit(code.OpPop)
			if err := c.Compile(cas.Body); err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
			endJumps = append(endJumps, c.emit(code.OpJump, 9999))
			c.changeOperand(nextCase, len(c.currentInstructions()))
		}
		c.emit(code.OpPop)
		if node.Default != nil {
			if err := c.Compile(node.Default); err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		} else {
			c.emit(code.OpNull)
		}
		endPos := len(c.currentInstructions())
		for _, j := range endJumps {
			c.changeOperand(j, endPos)
		}
		c.emit(code.OpPop)

	case *ast.WhileStatement:
		loopStart := len(c.currentInstructions())
		c.pushLoop(loopStart)
		if err := c.Compile(node.Condition); err != nil {
			return err
		}
		jumpOut := c.emit(code.OpJumpNotTruthy, 9999)
		if err := c.Compile(node.Body); err != nil {
			return err
		}
		c.emit(code.OpJump, loopStart)
		after := len(c.currentInstructions())
		c.changeOperand(jumpOut, after)
		c.popLoop(after)

	case *ast.ForInStatement:
		if err := c.Compile(node.Iterable); err != nil {
			return err
		}
		c.emit(code.OpMakeIter)
		loopStart := len(c.currentInstructions())
		c.pushLoop(loopStart)
		c.emit(code.OpHasIter)
		jumpOut := c.emit(code.OpJumpNotTruthy, 9999)
		c.emit(code.OpIterValue)
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			symbol = c.symbolTable.Define(node.Name.Value)
		}
		c.loadSet(symbol)
		if err := c.Compile(node.Body); err != nil {
			return err
		}
		c.emit(code.OpJump, loopStart)
		afterBody := len(c.currentInstructions())
		c.changeOperand(jumpOut, afterBody)
		c.emit(code.OpPop) // הסרת האיטרטור
		c.popLoop(afterBody) // break קופץ ל־Pop של האיטרטור

	case *ast.ForRangeStatement:
		if err := c.compileForRange(node); err != nil {
			return err
		}
	case *ast.BreakStatement:
		if len(c.loops) == 0 {
			return fmt.Errorf("עצור מחוץ ללולאה")
		}
		pos := c.emit(code.OpJump, 9999)
		c.loops[len(c.loops)-1].breaks = append(c.loops[len(c.loops)-1].breaks, pos)

	case *ast.ContinueStatement:
		if len(c.loops) == 0 {
			return fmt.Errorf("המשך מחוץ ללולאה")
		}
		c.emit(code.OpJump, c.loops[len(c.loops)-1].continuePos)

	case *ast.TryStatement:
		setupPos := c.emit(code.OpSetupTry, 9999)
		if err := c.Compile(node.Body); err != nil {
			return err
		}
		c.emit(code.OpEndTry)
		jumpAfter := c.emit(code.OpJump, 9999)
		catchPos := len(c.currentInstructions())
		c.changeOperand(setupPos, catchPos)
		// על המחסנית: מחרוזת השגיאה
		symbol, ok := c.symbolTable.Resolve(node.CatchName.Value)
		if !ok {
			symbol = c.symbolTable.Define(node.CatchName.Value)
		}
		c.loadSet(symbol)
		if err := c.Compile(node.CatchBody); err != nil {
			return err
		}
		afterCatch := len(c.currentInstructions())
		c.changeOperand(jumpAfter, afterCatch)

	case *ast.ThrowStatement:
		if err := c.Compile(node.Value); err != nil {
			return err
		}
		c.emit(code.OpThrow)

	case *ast.PrefixExpression:
		if folded, ok := tryFoldExpr(node); ok {
			emitFolded(c, folded)
			return nil
		}
		if err := c.Compile(node.Right); err != nil {
			return err
		}
		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("אופרטור קידומת לא נתמך במכונה: %s", node.Operator)
		}
	case *ast.InfixExpression:
		if folded, ok := tryFoldExpr(node); ok {
			emitFolded(c, folded)
			return nil
		}
		if node.Operator == "??" {
			if err := c.Compile(node.Left); err != nil {
				return err
			}
			jumpPos := c.emit(code.OpJumpNotNullKeep, 9999)
			if err := c.Compile(node.Right); err != nil {
				return err
			}
			c.changeOperand(jumpPos, len(c.currentInstructions()))
			return nil
		}
		if node.Operator == "וגם" || node.Operator == "&&" {
			if err := c.Compile(node.Left); err != nil {
				return err
			}
			jumpPos := c.emit(code.OpJumpFalsyKeep, 9999)
			if err := c.Compile(node.Right); err != nil {
				return err
			}
			c.changeOperand(jumpPos, len(c.currentInstructions()))
			return nil
		}
		if node.Operator == "או" || node.Operator == "||" {
			if err := c.Compile(node.Left); err != nil {
				return err
			}
			jumpPos := c.emit(code.OpJumpTruthyKeep, 9999)
			if err := c.Compile(node.Right); err != nil {
				return err
			}
			c.changeOperand(jumpPos, len(c.currentInstructions()))
			return nil
		}
		if node.Operator == "<" || node.Operator == "<=" {
			if err := c.Compile(node.Right); err != nil {
				return err
			}
			if err := c.Compile(node.Left); err != nil {
				return err
			}
			if node.Operator == "<" {
				c.emit(code.OpGreaterThan)
			} else {
				c.emit(code.OpGreaterEqual)
			}
			return nil
		}
		if err := c.Compile(node.Left); err != nil {
			return err
		}
		if err := c.Compile(node.Right); err != nil {
			return err
		}
		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "%":
			c.emit(code.OpMod)
		case "**":
			c.emit(code.OpPow)
		case ">":
			c.emit(code.OpGreaterThan)
		case ">=":
			c.emit(code.OpGreaterEqual)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("אופרטור לא נתמך במכונה: %s", node.Operator)
		}
	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("המשתנה %q לא מוגדר", node.Value)
		}
		c.loadGet(symbol)
	case *ast.NumberLiteral:
		c.emit(code.OpConstant, c.addConstant(&object.Number{Value: node.Value}))
	case *ast.StringLiteral:
		c.emit(code.OpConstant, c.addConstant(&object.String{Value: node.Value}))
	case *ast.TemplateLiteral:
		return fmt.Errorf("תבנית מחרוזת לא נתמכת במכונה עדיין")
	case *ast.BooleanLiteral:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	case *ast.NullLiteral:
		c.emit(code.OpNull)
	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			if err := c.Compile(el); err != nil {
				return err
			}
		}
		c.emit(code.OpArray, len(node.Elements))
	case *ast.HashLiteral:
		for _, pair := range node.Pairs {
			if err := c.Compile(pair.Key); err != nil {
				return err
			}
			if err := c.Compile(pair.Value); err != nil {
				return err
			}
		}
		c.emit(code.OpHash, len(node.Pairs))
	case *ast.IndexExpression:
		if err := c.Compile(node.Left); err != nil {
			return err
		}
		if err := c.Compile(node.Index); err != nil {
			return err
		}
		c.emit(code.OpIndex)
	case *ast.MemberExpression:
		if err := c.Compile(node.Object); err != nil {
			return err
		}
		c.emit(code.OpConstant, c.addConstant(&object.String{Value: node.Property.Value}))
		c.emit(code.OpIndex)
	case *ast.AssignExpression:
		switch left := node.Left.(type) {
		case *ast.Identifier:
			if err := c.Compile(node.Value); err != nil {
				return err
			}
			symbol, ok := c.symbolTable.Resolve(left.Value)
			if !ok {
				symbol = c.symbolTable.Define(left.Value)
			}
			c.loadSet(symbol)
			c.loadGet(symbol) // ערך ההשמה נשאר על המחסנית (כמו ביטוי)
		case *ast.IndexExpression:
			if err := c.Compile(left.Left); err != nil {
				return err
			}
			if err := c.Compile(left.Index); err != nil {
				return err
			}
			if err := c.Compile(node.Value); err != nil {
				return err
			}
			c.emit(code.OpSetIndex)
		case *ast.MemberExpression:
			if err := c.Compile(left.Object); err != nil {
				return err
			}
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: left.Property.Value}))
			if err := c.Compile(node.Value); err != nil {
				return err
			}
			c.emit(code.OpSetIndex)
		default:
			return fmt.Errorf("יעד השמה לא נתמך במכונה: %T", node.Left)
		}
	case *ast.CallExpression:
		if ident, ok := node.Function.(*ast.Identifier); ok {
			if builtinIdx, ok := BuiltinIndex(ident.Value); ok {
				for _, a := range node.Arguments {
					if err := c.Compile(a); err != nil {
						return err
					}
				}
				c.emit(code.OpCallBuiltin, builtinIdx, len(node.Arguments))
				return nil
			}
		}
		if err := c.Compile(node.Function); err != nil {
			return err
		}
		for _, a := range node.Arguments {
			if err := c.Compile(a); err != nil {
				return err
			}
		}
		c.emit(code.OpCall, len(node.Arguments))
	default:
		return fmt.Errorf("צומת לא נתמך במכונה עדיין: %T", node)
	}
	return nil
}

func (c *Compiler) compileFunction(node *ast.FunctionLiteral, asMethod bool) (*object.CompiledFunction, []Symbol, error) {
	c.enterScope()
	if asMethod {
		c.symbolTable.Define("זה")
	}
	defaults := make([]object.Object, len(node.Parameters))
	numRequired := 0
	seenDefault := false
	for i, p := range node.Parameters {
		if p == nil || p.Name == nil {
			continue
		}
		c.symbolTable.Define(p.Name.Value)
		if p.Default != nil {
			seenDefault = true
			def, ok := constantObject(p.Default)
			if !ok {
				c.leaveScope()
				return nil, nil, fmt.Errorf("ברירת מחדל לפרמטר %q חייבת להיות קבוע (מספר/מחרוזת/אמת/שקר/ריק)", p.Name.Value)
			}
			defaults[i] = def
		} else {
			if seenDefault {
				c.leaveScope()
				return nil, nil, fmt.Errorf("פרמטר בלי ברירת מחדל אחרי פרמטר עם ברירת מחדל")
			}
			numRequired++
		}
	}
	if err := c.Compile(node.Body); err != nil {
		c.leaveScope()
		return nil, nil, err
	}
	if c.lastInstructionIs(code.OpPop) {
		c.replaceLastPopWithReturn()
	}
	if !c.lastInstructionIs(code.OpReturnValue) && !c.lastInstructionIs(code.OpReturn) {
		c.emit(code.OpReturn)
	}
	freeSymbols := c.symbolTable.FreeSymbols
	numLocals := c.symbolTable.numDefs
	instructions := c.leaveScope()
	numParams := len(node.Parameters)
	if asMethod {
		numParams++
		numRequired++
		// שיטת מחלקה: this הוא הארגומנט הראשון — מרחיבים Defaults
		defaults = append([]object.Object{nil}, defaults...)
	}
	return &object.CompiledFunction{
		Instructions:  instructions,
		NumLocals:     numLocals,
		NumParameters: numParams,
		NumRequired:   numRequired,
		Defaults:      defaults,
	}, freeSymbols, nil
}

func (c *Compiler) compileClass(node *ast.ClassStatement) error {
	class := &object.Class{
		Name:      node.Name.Value,
		Methods:   map[string]*object.Method{},
		FieldDefs: map[string]*object.FieldDef{},
	}
	if node.Parent != nil {
		parent, ok := c.classes[node.Parent.Value]
		if !ok {
			return fmt.Errorf("מחלקת ההורה %q לא מוגדרת", node.Parent.Value)
		}
		class.Parent = parent
	}
	if node.Body != nil {
		for _, stmt := range node.Body.Statements {
			switch s := stmt.(type) {
			case *ast.VarStatement:
				if fl, ok := s.Value.(*ast.FunctionLiteral); ok {
					fn, frees, err := c.compileFunction(fl, true)
					if err != nil {
						return err
					}
					if len(frees) > 0 {
						return fmt.Errorf("מתודת מחלקה לא יכולה לסגור על משתנים מקומיים חיצוניים")
					}
					vis := s.Visibility
					if fl.Visibility == ast.VisPrivate {
						vis = ast.VisPrivate
					}
					class.Methods[s.Name.Value] = &object.Method{
						CompiledFn: fn,
						Public:     vis != ast.VisPrivate,
						Owner:      class,
					}
				} else {
					fd := &object.FieldDef{
						Expr:   s.Value,
						Public: s.Visibility != ast.VisPrivate,
						Owner:  class,
					}
					if s.Value == nil {
						fd.Default = &object.Null{}
					} else if def, ok := constantObject(s.Value); ok {
						fd.Default = def
					} else {
						initFn, err := c.compileFieldInit(s.Value)
						if err != nil {
							return err
						}
						fd.Init = initFn
					}
					class.FieldDefs[s.Name.Value] = fd
				}
			case *ast.FunctionLiteral:
				name := ""
				if s.Name != nil {
					name = s.Name.Value
				}
				fn, frees, err := c.compileFunction(s, true)
				if err != nil {
					return err
				}
				if len(frees) > 0 {
					return fmt.Errorf("מתודת מחלקה לא יכולה לסגור על משתנים מקומיים חיצוניים")
				}
				class.Methods[name] = &object.Method{
					CompiledFn: fn,
					Public:     s.Visibility != ast.VisPrivate,
					Owner:      class,
				}
			default:
				return fmt.Errorf("בגוף מחלקה נתמכים רק שדות ומתודות")
			}
		}
	}
	c.classes[class.Name] = class
	c.emit(code.OpConstant, c.addConstant(class))
	symbol := c.symbolTable.Define(node.Name.Value)
	c.loadSet(symbol)
	return nil
}

func (c *Compiler) compileInclude(node *ast.IncludeStatement) error {
	path := node.Path
	if mod, err := stdlib.LoadBuiltin(path); err == nil {
		m := mod.(*object.Module)
		c.emit(code.OpConstant, c.addConstant(m))
		symbol := c.symbolTable.Define(m.Name)
		c.loadSet(symbol)
		return nil
	}

	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(c.baseDir, path)
	}
	full = filepath.Clean(full)
	if c.included[full] {
		return nil
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Errorf("לא הצלחתי לכלול את %q: %v", path, err)
	}
	c.included[full] = true

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		return fmt.Errorf("שגיאה בקובץ %s: %s", filepath.Base(full), errs[0])
	}

	prev := c.baseDir
	prevFile := c.sourceFile
	c.baseDir = filepath.Dir(full)
	c.sourceFile = full
	defer func() {
		c.baseDir = prev
		c.sourceFile = prevFile
	}()
	if err := c.Compile(program); err != nil {
		return fmt.Errorf("שגיאה בקובץ %s: %v", filepath.Base(full), err)
	}
	return nil
}

func constantObject(expr ast.Expression) (object.Object, bool) {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		return &object.Number{Value: e.Value}, true
	case *ast.StringLiteral:
		return &object.String{Value: e.Value}, true
	case *ast.BooleanLiteral:
		return &object.Boolean{Value: e.Value}, true
	case *ast.NullLiteral:
		return &object.Null{}, true
	default:
		return nil, false
	}
}

func (c *Compiler) compileFieldInit(expr ast.Expression) (*object.CompiledFunction, error) {
	c.enterScope()
	if err := c.Compile(expr); err != nil {
		c.leaveScope()
		return nil, err
	}
	c.emit(code.OpReturnValue)
	numLocals := c.symbolTable.numDefs
	instructions := c.leaveScope()
	return &object.CompiledFunction{
		Instructions:  instructions,
		NumLocals:     numLocals,
		NumParameters: 0,
	}, nil
}

func emitFolded(c *Compiler, obj object.Object) {
	switch o := obj.(type) {
	case *object.Boolean:
		if o.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	case *object.Null:
		c.emit(code.OpNull)
	default:
		c.emit(code.OpConstant, c.addConstant(obj))
	}
}

// tryFoldExpr — קיפול קבועים בזמן קומפילציה (אופטימיזציה בסיסית)
func tryFoldExpr(expr ast.Expression) (object.Object, bool) {
	switch e := expr.(type) {
	case *ast.NumberLiteral, *ast.StringLiteral, *ast.BooleanLiteral, *ast.NullLiteral:
		return constantObject(e)
	case *ast.PrefixExpression:
		inner, ok := tryFoldExpr(e.Right)
		if !ok {
			return nil, false
		}
		switch e.Operator {
		case "-":
			n, ok := inner.(*object.Number)
			if !ok {
				return nil, false
			}
			return &object.Number{Value: -n.Value}, true
		case "!":
			return &object.Boolean{Value: !isTruthyObj(inner)}, true
		default:
			return nil, false
		}
	case *ast.InfixExpression:
		if e.Operator == "וגם" || e.Operator == "&&" {
			left, ok1 := tryFoldExpr(e.Left)
			if !ok1 {
				return nil, false
			}
			if !isTruthyObj(left) {
				return left, true
			}
			return tryFoldExpr(e.Right)
		}
		if e.Operator == "או" || e.Operator == "||" {
			left, ok1 := tryFoldExpr(e.Left)
			if !ok1 {
				return nil, false
			}
			if isTruthyObj(left) {
				return left, true
			}
			return tryFoldExpr(e.Right)
		}
		if e.Operator == "??" {
			left, ok1 := tryFoldExpr(e.Left)
			if !ok1 {
				return nil, false
			}
			if _, ok := left.(*object.Null); ok {
				return tryFoldExpr(e.Right)
			}
			return left, true
		}
		left, ok1 := tryFoldExpr(e.Left)
		right, ok2 := tryFoldExpr(e.Right)
		if !ok1 || !ok2 {
			return nil, false
		}
		return foldInfix(e.Operator, left, right)
	default:
		return nil, false
	}
}

func isTruthyObj(obj object.Object) bool {
	switch o := obj.(type) {
	case *object.Null:
		return false
	case *object.Boolean:
		return o.Value
	case *object.Number:
		return o.Value != 0
	case *object.String:
		return o.Value != ""
	default:
		return true
	}
}

func foldInfix(op string, left, right object.Object) (object.Object, bool) {
	if op == "+" {
		if _, ok := left.(*object.String); ok {
			return &object.String{Value: left.Inspect() + right.Inspect()}, true
		}
		if _, ok := right.(*object.String); ok {
			return &object.String{Value: left.Inspect() + right.Inspect()}, true
		}
	}
	ln, ok1 := left.(*object.Number)
	rn, ok2 := right.(*object.Number)
	if ok1 && ok2 {
		switch op {
		case "+":
			return &object.Number{Value: ln.Value + rn.Value}, true
		case "-":
			return &object.Number{Value: ln.Value - rn.Value}, true
		case "*":
			return &object.Number{Value: ln.Value * rn.Value}, true
		case "**":
			return &object.Number{Value: math.Pow(ln.Value, rn.Value)}, true
		case "/":
			if rn.Value == 0 {
				return nil, false
			}
			return &object.Number{Value: ln.Value / rn.Value}, true
		case "%":
			if rn.Value == 0 {
				return nil, false
			}
			return &object.Number{Value: float64(int64(ln.Value) % int64(rn.Value))}, true
		case ">":
			return &object.Boolean{Value: ln.Value > rn.Value}, true
		case ">=":
			return &object.Boolean{Value: ln.Value >= rn.Value}, true
		case "<":
			return &object.Boolean{Value: ln.Value < rn.Value}, true
		case "<=":
			return &object.Boolean{Value: ln.Value <= rn.Value}, true
		case "==":
			return &object.Boolean{Value: ln.Value == rn.Value}, true
		case "!=":
			return &object.Boolean{Value: ln.Value != rn.Value}, true
		}
	}
	switch op {
	case "==":
		return &object.Boolean{Value: objectsEqualFold(left, right)}, true
	case "!=":
		return &object.Boolean{Value: !objectsEqualFold(left, right)}, true
	}
	return nil, false
}

func objectsEqualFold(a, b object.Object) bool {
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
		return false
	}
}

func (c *Compiler) loadGet(symbol Symbol) {
	switch symbol.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, symbol.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, symbol.Index)
	case FreeScope:
		c.emit(code.OpGetFree, symbol.Index)
	}
}

// compileForRange — עבור i מ start עד end [בצע step], כולל קצוות.
func (c *Compiler) compileForRange(node *ast.ForRangeStatement) error {
	endName := fmt.Sprintf("__יוד_סוף_%d", len(c.loops))
	stepName := fmt.Sprintf("__יוד_צעד_%d", len(c.loops))

	if err := c.Compile(node.Start); err != nil {
		return err
	}
	sym, ok := c.symbolTable.Resolve(node.Name.Value)
	if !ok {
		sym = c.symbolTable.Define(node.Name.Value)
	}
	c.loadSet(sym)

	if err := c.Compile(node.End); err != nil {
		return err
	}
	endSym := c.symbolTable.Define(endName)
	c.loadSet(endSym)

	if node.Step != nil {
		if err := c.Compile(node.Step); err != nil {
			return err
		}
	} else {
		c.emit(code.OpConstant, c.addConstant(&object.Number{Value: 1}))
	}
	stepSym := c.symbolTable.Define(stepName)
	c.loadSet(stepSym)

	jumpToCond := c.emit(code.OpJump, 9999)

	incPos := len(c.currentInstructions())
	c.pushLoop(incPos)
	c.loadGet(sym)
	c.loadGet(stepSym)
	c.emit(code.OpAdd)
	c.loadSet(sym)

	condPos := len(c.currentInstructions())
	c.changeOperand(jumpToCond, condPos)

	// (צעד > 0 && i <= סוף) או (צעד <= 0 && i >= סוף)
	c.loadGet(stepSym)
	c.emit(code.OpConstant, c.addConstant(&object.Number{Value: 0}))
	c.emit(code.OpGreaterThan)
	jumpNeg := c.emit(code.OpJumpNotTruthy, 9999)

	c.loadGet(endSym)
	c.loadGet(sym)
	c.emit(code.OpGreaterEqual)
	jumpMerged := c.emit(code.OpJump, 9999)

	negPos := len(c.currentInstructions())
	c.changeOperand(jumpNeg, negPos)
	c.loadGet(sym)
	c.loadGet(endSym)
	c.emit(code.OpGreaterEqual)

	merged := len(c.currentInstructions())
	c.changeOperand(jumpMerged, merged)

	jumpOut := c.emit(code.OpJumpNotTruthy, 9999)
	if err := c.Compile(node.Body); err != nil {
		return err
	}
	c.emit(code.OpJump, incPos)
	after := len(c.currentInstructions())
	c.changeOperand(jumpOut, after)
	c.popLoop(after)
	return nil
}

func (c *Compiler) loadSet(symbol Symbol) {
	switch symbol.Scope {
	case GlobalScope:
		c.emit(code.OpSetGlobal, symbol.Index)
	case LocalScope:
		c.emit(code.OpSetLocal, symbol.Index)
	case FreeScope:
		c.emit(code.OpSetFree, symbol.Index)
	}
}

// loadFreeSource — דוחף Cell לסגירה חדשה (שיתוף mutable עם הסקופ החיצוני)
func (c *Compiler) loadFreeSource(symbol Symbol) {
	switch symbol.Scope {
	case LocalScope:
		c.emit(code.OpGetLocalCell, symbol.Index)
	case FreeScope:
		c.emit(code.OpGetFreeCell, symbol.Index)
	default:
		c.loadGet(symbol)
	}
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)
	c.setLastInstruction(op, pos)
	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNew := len(c.currentInstructions())
	updated := append(c.currentInstructions(), ins...)
	c.scopes[c.scopeIndex].instructions = updated
	return posNew
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	prev := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}
	c.scopes[c.scopeIndex].previousInstruction = prev
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	prev := c.scopes[c.scopeIndex].previousInstruction
	old := c.currentInstructions()
	c.scopes[c.scopeIndex].instructions = old[:last.Position]
	c.scopes[c.scopeIndex].lastInstruction = prev
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))
	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos])
	c.replaceInstruction(opPos, code.Make(op, operand))
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()
	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) enterScope() {
	c.scopes = append(c.scopes, CompilationScope{})
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer
	return instructions
}

func (c *Compiler) pushLoop(continuePos int) {
	c.loops = append(c.loops, LoopContext{continuePos: continuePos})
}

func (c *Compiler) popLoop(breakTarget int) {
	loop := c.loops[len(c.loops)-1]
	for _, pos := range loop.breaks {
		c.changeOperand(pos, breakTarget)
	}
	c.loops = c.loops[:len(c.loops)-1]
}
