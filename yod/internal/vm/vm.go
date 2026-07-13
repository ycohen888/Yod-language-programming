package vm

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"yod/internal/code"
	"yod/internal/compiler"
	"yod/internal/console"
	"yod/internal/object"
)

const StackSize = 2048
const GlobalsSize = 65536
const MaxFrames = 1024

var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
	Null  = &object.Null{}
)

type Frame struct {
	fn           *object.CompiledFunction
	ip           int
	basePointer  int
	afterReturn  object.Object // אם מוגדר — מוחזר במקום ערך החזרה (לבנאי)
	currentClass *object.Class // מחלקת המתודה הרצה (לפרטי/ציבורי)
	free         []*object.Cell
}

func NewFrame(fn *object.CompiledFunction, basePointer int) *Frame {
	return &Frame{fn: fn, ip: -1, basePointer: basePointer}
}

func (f *Frame) Instructions() code.Instructions {
	return f.fn.Instructions
}

type ExceptionHandler struct {
	CatchIP      int
	FrameIndex   int
	StackPointer int
}

type VM struct {
	constants   []object.Object
	globals     []object.Object
	stack       []object.Object
	sp          int
	frames      []*Frame
	framesIndex int
	handlers    []ExceptionHandler
	runBase     int // לקריאות מסונכרנות (GUI): עצור כש־framesIndex <= runBase
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn, 0)
	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame
	return &VM{
		constants:   bytecode.Constants,
		stack:       make([]object.Object, StackSize),
		globals:     make([]object.Object, GlobalsSize),
		frames:      frames,
		framesIndex: 1,
		handlers:    []ExceptionHandler{},
	}
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

// handleError — אם יש נסה/תפוס פעיל, קופץ לתפיסה ומחזיר true
func (vm *VM) handleError(msg string) bool {
	if len(vm.handlers) == 0 {
		return false
	}
	h := vm.handlers[len(vm.handlers)-1]
	vm.handlers = vm.handlers[:len(vm.handlers)-1]
	for vm.framesIndex > h.FrameIndex {
		vm.popFrame()
	}
	vm.sp = h.StackPointer
	_ = vm.push(&object.String{Value: msg})
	vm.currentFrame().ip = h.CatchIP - 1
	return true
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for {
		if vm.runBase > 0 && vm.framesIndex <= vm.runBase {
			return nil
		}
		vm.currentFrame().ip++
		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		if ip >= len(ins) {
			if vm.framesIndex > 1 {
				frame := vm.popFrame()
				vm.sp = frame.basePointer - 1
				if err := vm.push(Null); err != nil {
					return err
				}
				continue
			}
			break
		}
		op = code.Opcode(ins[ip])

		switch op {
		case code.OpConstant:
			constIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			if err := vm.push(vm.constants[constIndex]); err != nil {
				return err
			}
		case code.OpPop:
			top := vm.pop()
			if bb, ok := top.(*object.BoundBuiltin); ok {
				result := bb.Call()
				if errObj, ok := result.(*object.Error); ok {
					if !vm.handleError(errObj.Message) {
						return fmt.Errorf("%s", errObj.Message)
					}
				}
			}
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow:
			if err := vm.executeBinary(op); err != nil {
				if !vm.handleError(err.Error()) {
					return err
				}
			}
		case code.OpTrue:
			if err := vm.push(True); err != nil {
				return err
			}
		case code.OpFalse:
			if err := vm.push(False); err != nil {
				return err
			}
		case code.OpNull:
			if err := vm.push(Null); err != nil {
				return err
			}
		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterEqual:
			if err := vm.executeComparison(op); err != nil {
				if !vm.handleError(err.Error()) {
					return err
				}
			}
		case code.OpBang:
			if err := vm.push(nativeBool(!isTruthy(vm.pop()))); err != nil {
				return err
			}
		case code.OpMinus:
			operand := vm.pop()
			n, ok := operand.(*object.Number)
			if !ok {
				if !vm.handleError("האופרטור - עובד רק על מספרים") {
					return fmt.Errorf("האופרטור - עובד רק על מספרים")
				}
			} else if err := vm.push(&object.Number{Value: -n.Value}); err != nil {
				return err
			}
		case code.OpJump:
			pos := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip = pos - 1
		case code.OpJumpNotTruthy:
			pos := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}
		case code.OpJumpFalsyKeep:
			pos := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			condition := vm.stack[vm.sp-1]
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			} else {
				vm.pop()
			}
		case code.OpJumpTruthyKeep:
			pos := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			condition := vm.stack[vm.sp-1]
			if isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			} else {
				vm.pop()
			}
		case code.OpDup:
			if vm.sp == 0 {
				return fmt.Errorf("מחסנית ריקה ל־Dup")
			}
			if err := vm.push(vm.stack[vm.sp-1]); err != nil {
				return err
			}
		case code.OpJumpTruthy:
			pos := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			condition := vm.pop()
			if isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}
		case code.OpJumpNotNullKeep:
			pos := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			top := vm.stack[vm.sp-1]
			if _, isNull := top.(*object.Null); !isNull {
				vm.currentFrame().ip = pos - 1
			} else {
				vm.pop()
			}
		case code.OpSetGlobal:
			idx := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			vm.globals[idx] = vm.pop()
		case code.OpGetGlobal:
			idx := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			if err := vm.push(vm.globals[idx]); err != nil {
				return err
			}
		case code.OpSetLocal:
			localIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			frame := vm.currentFrame()
			val := vm.pop()
			slot := frame.basePointer + localIndex
			if cell, ok := vm.stack[slot].(*object.Cell); ok {
				cell.Value = val
			} else {
				vm.stack[slot] = val
			}
		case code.OpGetLocal:
			localIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			frame := vm.currentFrame()
			val := vm.stack[frame.basePointer+localIndex]
			if cell, ok := val.(*object.Cell); ok {
				val = cell.Value
			}
			if err := vm.push(val); err != nil {
				return err
			}
		case code.OpGetLocalCell:
			localIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			frame := vm.currentFrame()
			slot := frame.basePointer + localIndex
			val := vm.stack[slot]
			if cell, ok := val.(*object.Cell); ok {
				if err := vm.push(cell); err != nil {
					return err
				}
			} else {
				cell := &object.Cell{Value: val}
				vm.stack[slot] = cell
				if err := vm.push(cell); err != nil {
					return err
				}
			}
		case code.OpGetFreeCell:
			freeIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			frame := vm.currentFrame()
			if freeIndex < 0 || freeIndex >= len(frame.free) {
				if !vm.handleError("תא חופשי מחוץ לטווח") {
					return fmt.Errorf("תא חופשי מחוץ לטווח")
				}
			} else if err := vm.push(frame.free[freeIndex]); err != nil {
				return err
			}
		case code.OpGetFree:
			freeIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			frame := vm.currentFrame()
			if freeIndex < 0 || freeIndex >= len(frame.free) {
				if !vm.handleError("משתנה חופשי מחוץ לטווח") {
					return fmt.Errorf("משתנה חופשי מחוץ לטווח")
				}
			} else if err := vm.push(frame.free[freeIndex].Value); err != nil {
				return err
			}
		case code.OpSetFree:
			freeIndex := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			frame := vm.currentFrame()
			if freeIndex < 0 || freeIndex >= len(frame.free) {
				if !vm.handleError("השמה למשתנה חופשי מחוץ לטווח") {
					return fmt.Errorf("השמה למשתנה חופשי מחוץ לטווח")
				}
			} else {
				frame.free[freeIndex].Value = vm.pop()
			}
		case code.OpClosure:
			constIndex := int(readUint16(ins[ip+1:]))
			numFree := int(ins[ip+3])
			vm.currentFrame().ip += 3
			fn, ok := vm.constants[constIndex].(*object.CompiledFunction)
			if !ok {
				if !vm.handleError("OpClosure מצפה לפונקציה מקומפלת") {
					return fmt.Errorf("OpClosure מצפה לפונקציה מקומפלת")
				}
				continue
			}
			free := make([]*object.Cell, numFree)
			for i := numFree - 1; i >= 0; i-- {
				obj := vm.pop()
				if cell, ok := obj.(*object.Cell); ok {
					free[i] = cell
				} else {
					free[i] = &object.Cell{Value: obj}
				}
			}
			if err := vm.push(&object.Closure{Fn: fn, Free: free}); err != nil {
				return err
			}
		case code.OpCall:
			numArgs := int(ins[ip+1])
			vm.currentFrame().ip++
			if err := vm.callFunction(numArgs); err != nil {
				if !vm.handleError(err.Error()) {
					return err
				}
			}
		case code.OpReturnValue:
			returnValue := vm.pop()
			frame := vm.popFrame()
			vm.sp = frame.basePointer - 1
			if frame.afterReturn != nil {
				returnValue = frame.afterReturn
			}
			if err := vm.push(returnValue); err != nil {
				return err
			}
		case code.OpReturn:
			frame := vm.popFrame()
			vm.sp = frame.basePointer - 1
			ret := object.Object(Null)
			if frame.afterReturn != nil {
				ret = frame.afterReturn
			}
			if err := vm.push(ret); err != nil {
				return err
			}
		case code.OpCallBuiltin:
			builtinIdx := int(ins[ip+1])
			argc := int(ins[ip+2])
			vm.currentFrame().ip += 2
			args := make([]object.Object, argc)
			for i := argc - 1; i >= 0; i-- {
				args[i] = object.ResolveValue(vm.pop())
			}
			result := callBuiltin(builtinIdx, args)
			if errObj, ok := result.(*object.Error); ok {
				if !vm.handleError(errObj.Message) {
					return fmt.Errorf("%s", errObj.Message)
				}
			} else if err := vm.push(result); err != nil {
				return err
			}
		case code.OpArray:
			n := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			elements := make([]object.Object, n)
			for i := n - 1; i >= 0; i-- {
				elements[i] = vm.pop()
			}
			if err := vm.push(&object.Array{Elements: elements}); err != nil {
				return err
			}
		case code.OpHash:
			n := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			pairs := map[string]object.Object{}
			var hashErr string
			for i := 0; i < n; i++ {
				value := vm.pop()
				keyObj := vm.pop()
				key, ok := keyObj.(*object.String)
				if !ok {
					hashErr = "מפתח במילון חייב להיות מחרוזת"
					break
				}
				pairs[key.Value] = value
			}
			if hashErr != "" {
				if !vm.handleError(hashErr) {
					return fmt.Errorf("%s", hashErr)
				}
			} else if err := vm.push(&object.Hash{Pairs: pairs}); err != nil {
				return err
			}
		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()
			if err := vm.executeIndex(left, index); err != nil {
				if !vm.handleError(err.Error()) {
					return err
				}
			}
		case code.OpSetIndex:
			val := vm.pop()
			index := vm.pop()
			left := vm.pop()
			if err := vm.executeSetIndex(left, index, val); err != nil {
				if !vm.handleError(err.Error()) {
					return err
				}
			}
		case code.OpSetupTry:
			catchIP := int(readUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			vm.handlers = append(vm.handlers, ExceptionHandler{
				CatchIP:      catchIP,
				FrameIndex:   vm.framesIndex,
				StackPointer: vm.sp,
			})
		case code.OpEndTry:
			if len(vm.handlers) > 0 {
				vm.handlers = vm.handlers[:len(vm.handlers)-1]
			}
		case code.OpThrow:
			val := vm.pop()
			msg := val.Inspect()
			if s, ok := val.(*object.String); ok {
				msg = s.Value
			}
			if !vm.handleError(msg) {
				return fmt.Errorf("%s", msg)
			}
		case code.OpMakeIter:
			coll := vm.pop()
			iter, err := makeIterator(coll)
			if err != nil {
				if !vm.handleError(err.Error()) {
					return err
				}
			} else if err := vm.push(iter); err != nil {
				return err
			}
		case code.OpHasIter:
			iter, ok := vm.stack[vm.sp-1].(*object.Iterator)
			if !ok {
				if !vm.handleError("HasIter על ערך שאינו איטרטור") {
					return fmt.Errorf("HasIter על ערך שאינו איטרטור")
				}
			} else if err := vm.push(nativeBool(iter.Index < len(iter.Values))); err != nil {
				return err
			}
		case code.OpIterValue:
			iter, ok := vm.stack[vm.sp-1].(*object.Iterator)
			if !ok {
				if !vm.handleError("IterValue על ערך שאינו איטרטור") {
					return fmt.Errorf("IterValue על ערך שאינו איטרטור")
				}
			} else if iter.Index >= len(iter.Values) {
				if !vm.handleError("איטרטור הסתיים") {
					return fmt.Errorf("איטרטור הסתיים")
				}
			} else {
				val := iter.Values[iter.Index]
				iter.Index++
				if err := vm.push(val); err != nil {
					return err
				}
			}
		case code.OpParentRef:
			inst, ok := vm.pop().(*object.Instance)
			if !ok {
				if !vm.handleError("'הורה' דורש מופע") {
					return fmt.Errorf("'הורה' דורש מופע")
				}
			} else if err := vm.push(&object.ParentRef{Instance: inst}); err != nil {
				return err
			}
		case code.OpNew:
			numArgs := int(ins[ip+1])
			vm.currentFrame().ip++
			args := make([]object.Object, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = vm.pop()
			}
			classObj := vm.pop()
			class, ok := classObj.(*object.Class)
			if !ok {
				if !vm.handleError("חדש דורש מחלקה") {
					return fmt.Errorf("חדש דורש מחלקה")
				}
				continue
			}
			inst := &object.Instance{
				Class:  class,
				Fields: map[string]object.Object{},
			}
			for _, fi := range collectFieldInits(class) {
				if fi.init != nil {
					val := vm.CallCallable(fi.init, nil)
					if errObj, ok := val.(*object.Error); ok {
						if !vm.handleError(errObj.Message) {
							return fmt.Errorf("%s", errObj.Message)
						}
						continue
					}
					inst.Fields[fi.name] = val
				} else if fi.def != nil {
					inst.Fields[fi.name] = fi.def
				} else {
					inst.Fields[fi.name] = Null
				}
			}
			ctor, hasCtor := class.Methods["בנאי"]
			if !hasCtor {
				ctor, hasCtor = class.FindMethod("בנאי")
			}
			if hasCtor && ctor.CompiledFn != nil {
				if err := vm.push(&object.BoundMethod{
					Instance:   inst,
					CompiledFn: ctor.CompiledFn,
					Owner:      ctor.Owner,
				}); err != nil {
					return err
				}
				for _, a := range args {
					if err := vm.push(a); err != nil {
						return err
					}
				}
				if err := vm.callFunction(numArgs); err != nil {
					if !vm.handleError(err.Error()) {
						return err
					}
				} else {
					vm.currentFrame().afterReturn = inst
				}
			} else if numArgs > 0 {
				if !vm.handleError("למחלקה אין בנאי, אבל הועברו ארגומנטים") {
					return fmt.Errorf("למחלקה אין בנאי, אבל הועברו ארגומנטים")
				}
			} else if err := vm.push(inst); err != nil {
				return err
			}
		default:
			return fmt.Errorf("אופקוד לא ממומש: %d", op)
		}
	}
	return nil
}

// CallCallable — קריאה מסונכרנת לסגירה/פונקציה מקומפלת (למשל בלחיצה בזמן הצג)
func (vm *VM) CallCallable(fn object.Object, args []object.Object) object.Object {
	base := vm.framesIndex
	switch f := fn.(type) {
	case *object.Closure:
		if err := vm.push(f); err != nil {
			return &object.Error{Message: err.Error()}
		}
	case *object.CompiledFunction:
		if err := vm.push(f); err != nil {
			return &object.Error{Message: err.Error()}
		}
	default:
		return &object.Error{Message: "בלחיצה מצפה לפונקציה"}
	}
	for _, a := range args {
		if err := vm.push(a); err != nil {
			return &object.Error{Message: err.Error()}
		}
	}
	if err := vm.callFunction(len(args)); err != nil {
		return &object.Error{Message: err.Error()}
	}
	prev := vm.runBase
	vm.runBase = base
	err := vm.Run()
	vm.runBase = prev
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	if vm.sp <= 0 {
		return Null
	}
	return vm.pop()
}

func (vm *VM) callFunction(numArgs int) error {
	callee := vm.stack[vm.sp-1-numArgs]
	switch fn := callee.(type) {
	case *object.Closure:
		return vm.pushCallFrame(fn.Fn, numArgs, fn.Free)
	case *object.CompiledFunction:
		return vm.pushCallFrame(fn, numArgs, nil)
	case *object.BoundMethod:
		if fn.CompiledFn == nil {
			return fmt.Errorf("מתודה לא מקומפלת למכונה")
		}
		owner := fn.Owner
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		vm.pop() // BoundMethod
		if err := vm.push(fn.CompiledFn); err != nil {
			return err
		}
		if err := vm.push(fn.Instance); err != nil {
			return err
		}
		for _, a := range args {
			if err := vm.push(a); err != nil {
				return err
			}
		}
		if err := vm.callFunction(numArgs + 1); err != nil {
			return err
		}
		vm.currentFrame().currentClass = owner
		return nil
	case *object.ParentRef:
		parent := fn.Instance.Class.Parent
		if parent == nil {
			return fmt.Errorf("אין מחלקת הורה")
		}
		ctor, ok := parent.FindMethod("בנאי")
		if !ok || ctor.CompiledFn == nil {
			return fmt.Errorf("להורה אין בנאי")
		}
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		vm.pop() // ParentRef
		if err := vm.push(&object.BoundMethod{
			Instance:   fn.Instance,
			CompiledFn: ctor.CompiledFn,
			Owner:      ctor.Owner,
		}); err != nil {
			return err
		}
		for _, a := range args {
			if err := vm.push(a); err != nil {
				return err
			}
		}
		return vm.callFunction(numArgs)
	case *object.Builtin:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = object.ResolveValue(vm.pop())
		}
		vm.pop() // Builtin
		result := fn.Fn(args...)
		if errObj, ok := result.(*object.Error); ok {
			return fmt.Errorf("%s", errObj.Message)
		}
		return vm.push(result)
	case *object.BoundBuiltin:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = object.ResolveValue(vm.pop())
		}
		vm.pop() // BoundBuiltin
		result := fn.Call(args...)
		if errObj, ok := result.(*object.Error); ok {
			return fmt.Errorf("%s", errObj.Message)
		}
		return vm.push(result)
	default:
		return fmt.Errorf("קריאה לערך שאינו פונקציה")
	}
}

func (vm *VM) pushCallFrame(fn *object.CompiledFunction, numArgs int, free []*object.Cell) error {
	nReq := fn.NumRequired
	if nReq == 0 && len(fn.Defaults) == 0 {
		nReq = fn.NumParameters
	}
	if numArgs < nReq || numArgs > fn.NumParameters {
		if nReq == fn.NumParameters {
			return fmt.Errorf("מספר ארגומנטים שגוי: ציפיתי ל־%d קיבלתי %d", fn.NumParameters, numArgs)
		}
		return fmt.Errorf("מספר ארגומנטים שגוי: ציפיתי ל־%d…%d קיבלתי %d", nReq, fn.NumParameters, numArgs)
	}
	// השלמת ברירות מחדל על המחסנית לפני הכניסה לפריים
	for i := numArgs; i < fn.NumParameters; i++ {
		var def object.Object = &object.Null{}
		if i < len(fn.Defaults) && fn.Defaults[i] != nil {
			def = fn.Defaults[i]
		}
		if err := vm.push(def); err != nil {
			return err
		}
		numArgs++
	}
	frame := NewFrame(fn, vm.sp-numArgs)
	frame.free = free
	vm.pushFrame(frame)
	vm.sp = frame.basePointer + fn.NumLocals
	return nil
}

type fieldInitVM struct {
	name string
	def  object.Object
	init *object.CompiledFunction
}

func collectFieldInits(class *object.Class) []fieldInitVM {
	var chain []*object.Class
	for c := class; c != nil; c = c.Parent {
		chain = append(chain, c)
	}
	out := []fieldInitVM{}
	seen := map[string]bool{}
	for i := len(chain) - 1; i >= 0; i-- {
		c := chain[i]
		for name, fd := range c.FieldDefs {
			item := fieldInitVM{name: name, def: fd.Default, init: fd.Init}
			if seen[name] {
				for j := range out {
					if out[j].name == name {
						out[j] = item
						break
					}
				}
			} else {
				seen[name] = true
				out = append(out, item)
			}
		}
	}
	return out
}

func makeIterator(coll object.Object) (*object.Iterator, error) {
	switch c := coll.(type) {
	case *object.Array:
		vals := make([]object.Object, len(c.Elements))
		copy(vals, c.Elements)
		return &object.Iterator{Values: vals}, nil
	case *object.Hash:
		vals := make([]object.Object, 0, len(c.Pairs))
		for k := range c.Pairs {
			vals = append(vals, &object.String{Value: k})
		}
		return &object.Iterator{Values: vals}, nil
	case *object.String:
		runes := []rune(c.Value)
		vals := make([]object.Object, len(runes))
		for i, r := range runes {
			vals[i] = &object.String{Value: string(r)}
		}
		return &object.Iterator{Values: vals}, nil
	default:
		return nil, fmt.Errorf("עבור...בתוך עובד על רשימה, מילון או מחרוזת")
	}
}

func readUint16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("גלישת מחסנית")
	}
	vm.stack[vm.sp] = o
	vm.sp++
	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) executeBinary(op code.Opcode) error {
	right := object.ResolveValue(vm.pop())
	left := object.ResolveValue(vm.pop())
	if left.Type() == object.StringObj || right.Type() == object.StringObj {
		if op != code.OpAdd {
			return fmt.Errorf("אופרטור לא חוקי בין מחרוזות")
		}
		return vm.push(&object.String{Value: left.Inspect() + right.Inspect()})
	}
	ln, ok1 := left.(*object.Number)
	rn, ok2 := right.(*object.Number)
	if !ok1 || !ok2 {
		return fmt.Errorf("אופרטור לא נתמך בין %s ל־%s", left.Type(), right.Type())
	}
	var result float64
	switch op {
	case code.OpAdd:
		result = ln.Value + rn.Value
	case code.OpSub:
		result = ln.Value - rn.Value
	case code.OpMul:
		result = ln.Value * rn.Value
	case code.OpDiv:
		if rn.Value == 0 {
			return fmt.Errorf("חילוק באפס")
		}
		result = ln.Value / rn.Value
	case code.OpMod:
		if rn.Value == 0 {
			return fmt.Errorf("מודולו באפס")
		}
		result = float64(int64(ln.Value) % int64(rn.Value))
	case code.OpPow:
		result = math.Pow(ln.Value, rn.Value)
	}
	return vm.push(&object.Number{Value: result})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := object.ResolveValue(vm.pop())
	left := object.ResolveValue(vm.pop())
	if ln, ok1 := left.(*object.Number); ok1 {
		if rn, ok2 := right.(*object.Number); ok2 {
			return vm.push(nativeBool(compareNumbers(op, ln.Value, rn.Value)))
		}
	}
	switch op {
	case code.OpEqual:
		return vm.push(nativeBool(objectsEqual(left, right)))
	case code.OpNotEqual:
		return vm.push(nativeBool(!objectsEqual(left, right)))
	default:
		return fmt.Errorf("השוואה לא נתמכת בין %s ל־%s", left.Type(), right.Type())
	}
}

func compareNumbers(op code.Opcode, l, r float64) bool {
	switch op {
	case code.OpEqual:
		return l == r
	case code.OpNotEqual:
		return l != r
	case code.OpGreaterThan:
		return l > r
	case code.OpGreaterEqual:
		return l >= r
	}
	return false
}

func objectsEqual(a, b object.Object) bool {
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

func (vm *VM) executeIndex(left, index object.Object) error {
	switch left := left.(type) {
	case *object.Hash:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("מפתח מילון חייב להיות מחרוזת")
		}
		if m := object.LookupMethod(left, key.Value); m != nil {
			return vm.push(m)
		}
		if val, ok := left.Pairs[key.Value]; ok {
			return vm.push(val)
		}
		return vm.push(Null)
	case *object.Array:
		// רשימה[מספר] או רשימה.מתודה
		if idx, ok := index.(*object.Number); ok {
			i := int(idx.Value)
			if i < 0 || i >= len(left.Elements) {
				return fmt.Errorf("אינדקס מחוץ לטווח: %d", i)
			}
			return vm.push(left.Elements[i])
		}
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("אינדקס של רשימה חייב להיות מספר או שם מתודה")
		}
		if m := object.LookupMethod(left, key.Value); m != nil {
			return vm.push(m)
		}
		return fmt.Errorf("לרשימה אין מתודה בשם %q", key.Value)
	case *object.String:
		if key, ok := index.(*object.String); ok {
			if m := object.LookupMethod(left, key.Value); m != nil {
				return vm.push(m)
			}
			return fmt.Errorf("למחרוזת אין מתודה בשם %q", key.Value)
		}
		idx, ok := index.(*object.Number)
		if !ok {
			return fmt.Errorf("אינדקס של מחרוזת חייב להיות מספר או שם מתודה")
		}
		runes := []rune(left.Value)
		i := int(idx.Value)
		if i < 0 || i >= len(runes) {
			return fmt.Errorf("אינדקס מחוץ לטווח: %d", i)
		}
		return vm.push(&object.String{Value: string(runes[i])})
	case *object.Number, *object.Boolean, *object.Null:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("גישה לנקודה מצפה לשם מתודה")
		}
		if m := object.LookupMethod(left, key.Value); m != nil {
			return vm.push(m)
		}
		return fmt.Errorf("ל־%s אין מתודה בשם %q", left.Type(), key.Value)
	case *object.Instance:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("שם שדה/מתודה חייב להיות מחרוזת")
		}
		name := key.Value
		caller := vm.currentFrame().currentClass
		if val, ok := left.Fields[name]; ok {
			if fd, ok := left.Class.FindField(name); ok {
				if !object.CanAccess(fd.Public, fd.Owner, caller) {
					return fmt.Errorf("השדה %q הוא פרטי", name)
				}
			}
			return vm.push(val)
		}
		if method, ok := left.Class.FindMethod(name); ok {
			if !object.CanAccess(method.Public, method.Owner, caller) {
				return fmt.Errorf("המתודה %q היא פרטית", name)
			}
			if method.CompiledFn == nil {
				return fmt.Errorf("מתודה %q לא זמינה במכונה", name)
			}
			return vm.push(&object.BoundMethod{
				Instance:   left,
				CompiledFn: method.CompiledFn,
				Owner:      method.Owner,
			})
		}
		return fmt.Errorf("למופע אין שדה/מתודה בשם %q", name)
	case *object.ParentRef:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("שם מתודת הורה חייב להיות מחרוזת")
		}
		if left.Instance.Class.Parent == nil {
			return fmt.Errorf("אין מחלקת הורה")
		}
		method, ok := left.Instance.Class.Parent.FindMethod(key.Value)
		if !ok || method.CompiledFn == nil {
			return fmt.Errorf("להורה אין מתודה בשם %q", key.Value)
		}
		caller := vm.currentFrame().currentClass
		if !object.CanAccess(method.Public, method.Owner, caller) {
			return fmt.Errorf("המתודה %q היא פרטית", key.Value)
		}
		return vm.push(&object.BoundMethod{
			Instance:   left.Instance,
			CompiledFn: method.CompiledFn,
			Owner:      method.Owner,
		})
	case *object.Module:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("שם במודול חייב להיות מחרוזת")
		}
		val, ok := left.Get(key.Value)
		if !ok {
			return fmt.Errorf("במודול אין %q", key.Value)
		}
		return vm.push(val)
	case *object.GuiWidget:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("שם ברכיב חייב להיות מחרוזת")
		}
		val, ok := left.Get(key.Value)
		if !ok {
			return fmt.Errorf("לרכיב אין %q", key.Value)
		}
		return vm.push(val)
	default:
		return fmt.Errorf("אינדקס לא נתמך על %s", left.Type())
	}
}

func (vm *VM) executeSetIndex(left, index, val object.Object) error {
	switch left := left.(type) {
	case *object.Array:
		idx, ok := index.(*object.Number)
		if !ok {
			return fmt.Errorf("אינדקס של רשימה חייב להיות מספר")
		}
		i := int(idx.Value)
		if i < 0 || i >= len(left.Elements) {
			return fmt.Errorf("אינדקס מחוץ לטווח: %d", i)
		}
		left.Elements[i] = val
		return vm.push(val)
	case *object.Hash:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("מפתח מילון חייב להיות מחרוזת")
		}
		left.Pairs[key.Value] = val
		return vm.push(val)
	case *object.Instance:
		key, ok := index.(*object.String)
		if !ok {
			return fmt.Errorf("שם שדה חייב להיות מחרוזת")
		}
		if fd, ok := left.Class.FindField(key.Value); ok {
			if !object.CanAccess(fd.Public, fd.Owner, vm.currentFrame().currentClass) {
				return fmt.Errorf("השדה %q הוא פרטי", key.Value)
			}
		}
		left.Fields[key.Value] = val
		return vm.push(val)
	default:
		return fmt.Errorf("השמה באינדקס אפשרית רק על רשימה, מילון או מופע")
	}
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case Null, False:
		return false
	case True:
		return true
	default:
		if b, ok := obj.(*object.Boolean); ok {
			return b.Value
		}
		if n, ok := obj.(*object.Number); ok {
			return n.Value != 0
		}
		if s, ok := obj.(*object.String); ok {
			return s.Value != ""
		}
		return true
	}
}

func nativeBool(v bool) *object.Boolean {
	if v {
		return True
	}
	return False
}

func callBuiltin(idx int, args []object.Object) object.Object {
	switch idx {
	case 0:
		parts := make([]any, 0, len(args))
		for _, a := range args {
			parts = append(parts, a.Inspect())
		}
		if len(parts) == 0 {
			console.Println()
		} else {
			console.Println(parts...)
		}
		return Null
	case 1:
		if len(args) != 1 {
			return &object.Error{Message: "אורך מצפה לארגומנט אחד"}
		}
		switch a := args[0].(type) {
		case *object.Array:
			return &object.Number{Value: float64(len(a.Elements))}
		case *object.String:
			return &object.Number{Value: float64(len([]rune(a.Value)))}
		case *object.Hash:
			return &object.Number{Value: float64(len(a.Pairs))}
		default:
			return &object.Error{Message: "אורך עובד על רשימה, מילון או מחרוזת"}
		}
	case 2:
		if len(args) != 1 {
			return &object.Error{Message: "למספר מצפה לארגומנט אחד"}
		}
		switch a := args[0].(type) {
		case *object.Number:
			return a
		case *object.String:
			n, err := strconv.ParseFloat(strings.TrimSpace(a.Value), 64)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("לא הצלחתי להמיר %q למספר", a.Value)}
			}
			return &object.Number{Value: n}
		case *object.Boolean:
			if a.Value {
				return &object.Number{Value: 1}
			}
			return &object.Number{Value: 0}
		default:
			return &object.Error{Message: "למספר לא תומך בטיפוס זה"}
		}
	case 3:
		if len(args) != 1 {
			return &object.Error{Message: "למחרוזת מצפה לארגומנט אחד"}
		}
		return &object.String{Value: args[0].Inspect()}
	case 4:
		if len(args) != 1 {
			return &object.Error{Message: "סוג מצפה לארגומנט אחד"}
		}
		return &object.String{Value: string(args[0].Type())}
	case 5:
		if len(args) > 0 {
			console.Print(args[0].Inspect())
		}
		var s string
		fmt.Scanln(&s)
		return &object.String{Value: s}
	case 6:
		if len(args) < 2 {
			return &object.Error{Message: "הוסף מצפה לרשימה וערך"}
		}
		arr, ok := args[0].(*object.Array)
		if !ok {
			return &object.Error{Message: "הארגומנט הראשון של הוסף חייב להיות רשימה"}
		}
		for _, v := range args[1:] {
			arr.Elements = append(arr.Elements, v)
		}
		return arr
	case 7:
		return object.Range(args...)
	case 8:
		if len(args) != 0 {
			return &object.Error{Message: "אקראי מצפה ל־0 ארגומנטים"}
		}
		return object.Random01()
	case 9:
		return object.RandomBetween(args...)
	default:
		return &object.Error{Message: "פונקציה מובנית לא ידועה"}
	}
}
