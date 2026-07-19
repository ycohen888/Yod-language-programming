package stdlib

import (
	"math"

	"yod/internal/object"
)

func NewMathModule() *object.Module {
	m := &object.Module{Name: "מתמטיקה", Attrs: map[string]object.Object{}}
	m.Attrs["פי"] = &object.Number{Value: math.Pi}
	m.Attrs["אי"] = &object.Number{Value: math.E}
	m.Attrs["סינוס"] = &object.Builtin{Fn: mathSin}
	m.Attrs["קוסינוס"] = &object.Builtin{Fn: mathCos}
	m.Attrs["טנגנס"] = &object.Builtin{Fn: mathTan}
	m.Attrs["שורש"] = &object.Builtin{Fn: mathSqrt}
	m.Attrs["חזקה"] = &object.Builtin{Fn: mathPow}
	m.Attrs["לוג"] = &object.Builtin{Fn: mathLog}
	m.Attrs["לוג10"] = &object.Builtin{Fn: mathLog10}
	m.Attrs["מוחלט"] = &object.Builtin{Fn: mathAbs}
	m.Attrs["רצפה"] = &object.Builtin{Fn: mathFloor}
	m.Attrs["תקרה"] = &object.Builtin{Fn: mathCeil}
	m.Attrs["עיגול"] = &object.Builtin{Fn: mathRound}
	m.Attrs["מקסימום"] = &object.Builtin{Fn: mathMax}
	m.Attrs["מינימום"] = &object.Builtin{Fn: mathMin}
	m.Attrs["atan2"] = &object.Builtin{Fn: mathAtan2}
	m.Attrs["זווית"] = &object.Builtin{Fn: mathAngleDeg}
	return m
}

func mathOne(name string, args []object.Object, fn func(float64) float64) object.Object {
	if err := expectArgs("מתמטיקה."+name, 1, args); err != nil {
		return err
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("מתמטיקה." + name + " מצפה למספר")
	}
	return &object.Number{Value: fn(n.Value)}
}

func mathSin(args ...object.Object) object.Object {
	return mathOne("סינוס", args, math.Sin)
}
func mathCos(args ...object.Object) object.Object {
	return mathOne("קוסינוס", args, math.Cos)
}
func mathTan(args ...object.Object) object.Object {
	return mathOne("טנגנס", args, math.Tan)
}
func mathSqrt(args ...object.Object) object.Object {
	if err := expectArgs("מתמטיקה.שורש", 1, args); err != nil {
		return err
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("מתמטיקה.שורש מצפה למספר")
	}
	if n.Value < 0 {
		return errObj("שורש של מספר שלילי")
	}
	return &object.Number{Value: math.Sqrt(n.Value)}
}
func mathLog(args ...object.Object) object.Object {
	return mathOne("לוג", args, math.Log)
}
func mathLog10(args ...object.Object) object.Object {
	return mathOne("לוג10", args, math.Log10)
}
func mathAbs(args ...object.Object) object.Object {
	return mathOne("מוחלט", args, math.Abs)
}
func mathFloor(args ...object.Object) object.Object {
	return mathOne("רצפה", args, math.Floor)
}
func mathCeil(args ...object.Object) object.Object {
	return mathOne("תקרה", args, math.Ceil)
}
func mathRound(args ...object.Object) object.Object {
	return mathOne("עיגול", args, math.Round)
}

func mathPow(args ...object.Object) object.Object {
	if err := expectArgs("מתמטיקה.חזקה", 2, args); err != nil {
		return err
	}
	a, ok1 := args[0].(*object.Number)
	b, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("מתמטיקה.חזקה מצפה למספרים")
	}
	return &object.Number{Value: math.Pow(a.Value, b.Value)}
}

func mathMax(args ...object.Object) object.Object {
	if len(args) < 1 {
		return errObj("מתמטיקה.מקסימום מצפה לפחות למספר אחד")
	}
	best, ok := args[0].(*object.Number)
	if !ok {
		return errObj("מתמטיקה.מקסימום מצפה למספרים")
	}
	for _, a := range args[1:] {
		n, ok := a.(*object.Number)
		if !ok {
			return errObj("מתמטיקה.מקסימום מצפה למספרים")
		}
		if n.Value > best.Value {
			best = n
		}
	}
	return best
}

func mathMin(args ...object.Object) object.Object {
	if len(args) < 1 {
		return errObj("מתמטיקה.מינימום מצפה לפחות למספר אחד")
	}
	best, ok := args[0].(*object.Number)
	if !ok {
		return errObj("מתמטיקה.מינימום מצפה למספרים")
	}
	for _, a := range args[1:] {
		n, ok := a.(*object.Number)
		if !ok {
			return errObj("מתמטיקה.מינימום מצפה למספרים")
		}
		if n.Value < best.Value {
			best = n
		}
	}
	return best
}

// mathAtan2 — atan2(y, x) ברדיאנים (כמו בשפות אחרות).
func mathAtan2(args ...object.Object) object.Object {
	if err := expectArgs("מתמטיקה.atan2", 2, args); err != nil {
		return err
	}
	y, ok1 := args[0].(*object.Number)
	x, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("מתמטיקה.atan2 מצפה לשני מספרים (y, x)")
	}
	return &object.Number{Value: math.Atan2(y.Value, x.Value)}
}

// mathAngleDeg — זווית במעלות מ־(0,0) ל־(x,y): זווית(y, x).
func mathAngleDeg(args ...object.Object) object.Object {
	if err := expectArgs("מתמטיקה.זווית", 2, args); err != nil {
		return err
	}
	y, ok1 := args[0].(*object.Number)
	x, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("מתמטיקה.זווית מצפה לשני מספרים (y, x)")
	}
	deg := math.Atan2(y.Value, x.Value) * 180 / math.Pi
	return &object.Number{Value: deg}
}
