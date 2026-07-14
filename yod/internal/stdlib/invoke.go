package stdlib

import "yod/internal/object"

func isCallable(o object.Object) bool {
	switch o.(type) {
	case *object.Function, *object.Closure, *object.CompiledFunction, *object.Builtin:
		return true
	default:
		return false
	}
}

func invokeYod(fn object.Object, args []object.Object) {
	if fn == nil {
		return
	}
	var res object.Object
	if object.InvokeCallable != nil {
		res = object.InvokeCallable(fn, args)
	} else if f, ok := fn.(*object.Function); ok && object.InvokeFunction != nil {
		res = object.InvokeFunction(f, args)
	} else if b, ok := fn.(*object.Builtin); ok && b.Fn != nil {
		res = b.Fn(args...)
	}
	_ = res
}
