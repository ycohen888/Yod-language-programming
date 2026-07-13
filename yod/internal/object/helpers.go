package object

import (
	"fmt"
	"math"
)

// Truthy — האם ערך נחשב אמת בתנאים / סנן
func Truthy(obj Object) bool {
	switch o := obj.(type) {
	case *Boolean:
		return o.Value
	case *Null:
		return false
	case *Number:
		return o.Value != 0
	case *String:
		return o.Value != ""
	case *Array:
		return len(o.Elements) != 0
	case *Hash:
		return len(o.Pairs) != 0
	default:
		return obj != nil
	}
}

// Call — קריאה לפונקציית משתמש / סגירה דרך המנוע הפעיל
func Call(fn Object, args ...Object) Object {
	if InvokeCallable == nil {
		return &Error{Message: "אין מנוע לקריאת פונקציה"}
	}
	return InvokeCallable(fn, args)
}

func fnArity(fn Object) int {
	switch f := fn.(type) {
	case *Function:
		return len(f.Parameters)
	case *CompiledFunction:
		return f.NumParameters
	case *Closure:
		return f.Fn.NumParameters
	case *BoundMethod:
		if f.Function != nil {
			return len(f.Function.Parameters)
		}
		if f.CompiledFn != nil {
			return f.CompiledFn.NumParameters
		}
	}
	return 1
}

// CallItem — קורא לפונקציה עם איבר (ואינדקס אם יש שני פרמטרים)
func CallItem(fn Object, item Object, index int) Object {
	switch fnArity(fn) {
	case 0:
		return Call(fn)
	case 2:
		return Call(fn, item, &Number{Value: float64(index)})
	default:
		return Call(fn, item)
	}
}

// DeepCopy — העתקה רקורסיבית לרשימה/מילון (טיפוסים אחרים מוחזרים כמו שהם)
func DeepCopy(obj Object) Object {
	switch o := obj.(type) {
	case *Array:
		out := &Array{Elements: make([]Object, len(o.Elements))}
		for i, e := range o.Elements {
			out.Elements[i] = DeepCopy(e)
		}
		return out
	case *Hash:
		out := &Hash{Pairs: make(map[string]Object, len(o.Pairs))}
		for k, v := range o.Pairs {
			out.Pairs[k] = DeepCopy(v)
		}
		return out
	default:
		return obj
	}
}

// Equal — השוואה מבנית פשוטה
func Equal(a, b Object) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Type() != b.Type() {
		return false
	}
	switch x := a.(type) {
	case *Number:
		y := b.(*Number)
		return x.Value == y.Value
	case *String:
		return x.Value == b.(*String).Value
	case *Boolean:
		return x.Value == b.(*Boolean).Value
	case *Null:
		return true
	case *Array:
		y := b.(*Array)
		if len(x.Elements) != len(y.Elements) {
			return false
		}
		for i := range x.Elements {
			if !Equal(x.Elements[i], y.Elements[i]) {
				return false
			}
		}
		return true
	case *Hash:
		y := b.(*Hash)
		if len(x.Pairs) != len(y.Pairs) {
			return false
		}
		for k, v := range x.Pairs {
			ov, ok := y.Pairs[k]
			if !ok || !Equal(v, ov) {
				return false
			}
		}
		return true
	default:
		return a.Inspect() == b.Inspect()
	}
}

// Range — בונה רשימת מספרים [start, end) עם קפיצה
func Range(args ...Object) Object {
	var start, end, step float64
	switch len(args) {
	case 1:
		n, ok := args[0].(*Number)
		if !ok {
			return &Error{Message: "טווח מצפה למספר"}
		}
		start, end, step = 0, n.Value, 1
	case 2:
		a, ok1 := args[0].(*Number)
		b, ok2 := args[1].(*Number)
		if !ok1 || !ok2 {
			return &Error{Message: "טווח מצפה למספרים"}
		}
		start, end, step = a.Value, b.Value, 1
	case 3:
		a, ok1 := args[0].(*Number)
		b, ok2 := args[1].(*Number)
		c, ok3 := args[2].(*Number)
		if !ok1 || !ok2 || !ok3 {
			return &Error{Message: "טווח מצפה למספרים"}
		}
		start, end, step = a.Value, b.Value, c.Value
	default:
		return &Error{Message: "טווח מצפה ל־1 עד 3 ארגומנטים"}
	}
	if step == 0 {
		return &Error{Message: "קפיצת טווח לא יכולה להיות 0"}
	}
	var elems []Object
	if step > 0 {
		for v := start; v < end; v += step {
			elems = append(elems, &Number{Value: v})
			if len(elems) > 1_000_000 {
				return &Error{Message: "טווח גדול מדי"}
			}
		}
	} else {
		for v := start; v > end; v += step {
			elems = append(elems, &Number{Value: v})
			if len(elems) > 1_000_000 {
				return &Error{Message: "טווח גדול מדי"}
			}
		}
	}
	return &Array{Elements: elems}
}

// Random01 — מספר אקראי ב־[0,1)
func Random01() Object {
	return &Number{Value: randFloat()}
}

// RandomBetween — מספר שלם אקראי בטווח כולל [a,b]
func RandomBetween(args ...Object) Object {
	if len(args) != 2 {
		return &Error{Message: "אקראי_בין מצפה לשני מספרים"}
	}
	a, ok1 := args[0].(*Number)
	b, ok2 := args[1].(*Number)
	if !ok1 || !ok2 {
		return &Error{Message: "אקראי_בין מצפה למספרים"}
	}
	lo, hi := a.Value, b.Value
	if lo > hi {
		lo, hi = hi, lo
	}
	loI := int64(math.Floor(lo))
	hiI := int64(math.Floor(hi))
	if loI == hiI {
		return &Number{Value: float64(loI)}
	}
	n := hiI - loI + 1
	return &Number{Value: float64(loI + randInt63n(n))}
}

func FormatString(template string, args []Object) Object {
	out := make([]rune, 0, len(template)+16)
	runes := []rune(template)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '{' && i+1 < len(runes) {
			j := i + 1
			if runes[j] == '{' {
				out = append(out, '{')
				i++
				continue
			}
			numStart := j
			for j < len(runes) && runes[j] >= '0' && runes[j] <= '9' {
				j++
			}
			if j > numStart && j < len(runes) && runes[j] == '}' {
				idx := 0
				for _, r := range runes[numStart:j] {
					idx = idx*10 + int(r-'0')
				}
				if idx < 0 || idx >= len(args) {
					return &Error{Message: fmt.Sprintf("פורמט: אין ארגומנט באינדקס %d", idx)}
				}
				out = append(out, []rune(args[idx].Inspect())...)
				i = j
				continue
			}
		}
		if runes[i] == '}' && i+1 < len(runes) && runes[i+1] == '}' {
			out = append(out, '}')
			i++
			continue
		}
		out = append(out, runes[i])
	}
	return &String{Value: string(out)}
}
