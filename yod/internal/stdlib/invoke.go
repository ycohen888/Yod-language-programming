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
	_ = invokeYodResult(fn, args)
}

func invokeYodResult(fn object.Object, args []object.Object) object.Object {
	if fn == nil {
		return object.Nil
	}
	// כל callback של UI/טיימר הוא כניסה חיצונית להרצת קוד יוד — נועלים את מנעול הריצה
	// (re-entrant) כדי לא לרוץ במקביל למשימת רקע. עלות זניחה כשאין משימות.
	object.LockYod()
	defer object.UnlockYod()
	if object.InvokeCallable != nil {
		return object.InvokeCallable(fn, args)
	}
	if f, ok := fn.(*object.Function); ok && object.InvokeFunction != nil {
		return object.InvokeFunction(f, args)
	}
	if b, ok := fn.(*object.Builtin); ok && b.Fn != nil {
		return b.Fn(args...)
	}
	return object.Nil
}
