package object

import (
	"fmt"
	"sort"
	"strings"

	"yod/internal/console"
)

// BoundBuiltin — מתודה מובנית על רשימה/מילון (למשל אדם.הדפס_שמות)
type BoundBuiltin struct {
	Self Object
	Name string
	Fn   func(self Object, args ...Object) Object
}

func (b *BoundBuiltin) Type() Type { return BuiltinObj }
func (b *BoundBuiltin) Inspect() string {
	return "מתודה מובנית." + b.Name
}

func (b *BoundBuiltin) Call(args ...Object) Object {
	return b.Fn(b.Self, args...)
}

// ResolveValue — מתודה מובנית בלי () מופעלת אוטומטית (למשל פפ.אורך)
func ResolveValue(obj Object) Object {
	if bb, ok := obj.(*BoundBuiltin); ok {
		return bb.Call()
	}
	return obj
}

func hashKeys(h *Hash) []string {
	return h.Keys()
}

func hashKeysSorted(h *Hash) []string {
	return h.SortedKeys()
}

func errArity(name string, want string) Object {
	return &Error{Message: fmt.Sprintf("%s מצפה ל־%s", name, want)}
}

func hashPrintValues(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 0 {
		return errArity("הדפס_מידע", "0 ארגומנטים")
	}
	for _, k := range hashKeys(h) {
		console.Println(h.Pairs[k].Inspect())
	}
	return Nil
}

func hashKeysMethod(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 0 {
		return errArity("שמות", "0 ארגומנטים")
	}
	keys := hashKeys(h)
	arr := &Array{Elements: make([]Object, len(keys))}
	for i, k := range keys {
		arr.Elements[i] = &String{Value: k}
	}
	return arr
}

func arrayPrintAll(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("הדפס_הכל", "0 ארגומנטים")
	}
	for _, e := range a.Elements {
		console.Println(e.Inspect())
	}
	return Nil
}

func arrayPush(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) < 1 {
		return errArity("הוסף", "לפחות ערך אחד")
	}
	a.Elements = append(a.Elements, args...)
	return a
}

var hashMethods = map[string]func(self Object, args ...Object) Object{
	"הדפס_שמות": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("הדפס_שמות", "0 ארגומנטים")
		}
		for _, k := range hashKeys(h) {
			console.Println(k)
		}
		return Nil
	},
	"הדפס_מידע": hashPrintValues,
	"הדפס_ערכים": hashPrintValues,
	"הדפס_הכל": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("הדפס_הכל", "0 ארגומנטים")
		}
		for _, k := range hashKeys(h) {
			console.Println(k + ": " + h.Pairs[k].Inspect())
		}
		return Nil
	},
	"שמות": hashKeysMethod,
	"מפתחות": hashKeysMethod,
	"שמות_ממוינים": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("שמות_ממוינים", "0 ארגומנטים")
		}
		keys := hashKeysSorted(h)
		arr := &Array{Elements: make([]Object, len(keys))}
		for i, k := range keys {
			arr.Elements[i] = &String{Value: k}
		}
		return arr
	},
	"ערכים": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("ערכים", "0 ארגומנטים")
		}
		keys := hashKeys(h)
		arr := &Array{Elements: make([]Object, len(keys))}
		for i, k := range keys {
			arr.Elements[i] = h.Pairs[k]
		}
		return arr
	},
	"גודל": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("גודל", "0 ארגומנטים")
		}
		return &Number{Value: float64(len(h.Pairs))}
	},
	"ריק": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("ריק", "0 ארגומנטים")
		}
		return &Boolean{Value: len(h.Pairs) == 0}
	},
	"יש": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 1 {
			return errArity("יש", "מפתח אחד")
		}
		key, ok := args[0].(*String)
		if !ok {
			return &Error{Message: "יש מצפה למפתח מחרוזת"}
		}
		_, found := h.Pairs[key.Value]
		return &Boolean{Value: found}
	},
	"קבל": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) < 1 || len(args) > 2 {
			return errArity("קבל", "מפתח ואופציונלי ברירת_מחדל")
		}
		key, ok := args[0].(*String)
		if !ok {
			return &Error{Message: "קבל מצפה למפתח מחרוזת"}
		}
		if v, found := h.Pairs[key.Value]; found {
			return v
		}
		if len(args) == 2 {
			return args[1]
		}
		return Nil
	},
	"הגדר": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 2 {
			return errArity("הגדר", "מפתח וערך")
		}
		key, ok := args[0].(*String)
		if !ok {
			return &Error{Message: "הגדר מצפה למפתח מחרוזת"}
		}
		h.Pairs[key.Value] = args[1]
		return h
	},
	"מחק": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 1 {
			return errArity("מחק", "מפתח אחד")
		}
		key, ok := args[0].(*String)
		if !ok {
			return &Error{Message: "מחק מצפה למפתח מחרוזת"}
		}
		_, existed := h.Pairs[key.Value]
		delete(h.Pairs, key.Value)
		return &Boolean{Value: existed}
	},
	"נקה": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("נקה", "0 ארגומנטים")
		}
		h.Pairs = map[string]Object{}
		return h
	},
	"העתק": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 0 {
			return errArity("העתק", "0 ארגומנטים")
		}
		cp := &Hash{Pairs: make(map[string]Object, len(h.Pairs))}
		for k, v := range h.Pairs {
			cp.Pairs[k] = v
		}
		return cp
	},
	"העתק_עמוק": hashDeepCopy,
	"פריטים":    hashItems,
	"מפה":       hashMap,
	"סנן":       hashFilter,
	"מצא":       hashFind,
	"לכל":       hashForEach,
	"יש_ערך":    hashHasValue,
	"עדכן": func(self Object, args ...Object) Object {
		h := self.(*Hash)
		if len(args) != 1 {
			return errArity("עדכן", "מילון אחד")
		}
		other, ok := args[0].(*Hash)
		if !ok {
			return &Error{Message: "עדכן מצפה למילון"}
		}
		for k, v := range other.Pairs {
			h.Pairs[k] = v
		}
		return h
	},
}

var arrayMethods = map[string]func(self Object, args ...Object) Object{
	"הדפס_הכל": arrayPrintAll,
	"הדפס": arrayPrintAll,
	"גודל": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("גודל", "0 ארגומנטים")
		}
		return &Number{Value: float64(len(a.Elements))}
	},
	"ריק": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("ריק", "0 ארגומנטים")
		}
		return &Boolean{Value: len(a.Elements) == 0}
	},
	"הוסף": arrayPush,
	"דחוף": arrayPush,
	"משוך": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("משוך", "0 ארגומנטים")
		}
		if len(a.Elements) == 0 {
			return Nil
		}
		last := a.Elements[len(a.Elements)-1]
		a.Elements = a.Elements[:len(a.Elements)-1]
		return last
	},
	"ראשון": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("ראשון", "0 ארגומנטים")
		}
		if len(a.Elements) == 0 {
			return Nil
		}
		return a.Elements[0]
	},
	"אחרון": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("אחרון", "0 ארגומנטים")
		}
		if len(a.Elements) == 0 {
			return Nil
		}
		return a.Elements[len(a.Elements)-1]
	},
	"מכיל": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 1 {
			return errArity("מכיל", "ערך אחד")
		}
		want := args[0].Inspect()
		for _, e := range a.Elements {
			if e.Inspect() == want {
				return &Boolean{Value: true}
			}
		}
		return &Boolean{Value: false}
	},
	"אינדקס_של": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 1 {
			return errArity("אינדקס_של", "ערך אחד")
		}
		want := args[0].Inspect()
		for i, e := range a.Elements {
			if e.Inspect() == want {
				return &Number{Value: float64(i)}
			}
		}
		return &Number{Value: -1}
	},
	"אינדקס_אחרון": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 1 {
			return errArity("אינדקס_אחרון", "ערך אחד")
		}
		want := args[0].Inspect()
		for i := len(a.Elements) - 1; i >= 0; i-- {
			if a.Elements[i].Inspect() == want {
				return &Number{Value: float64(i)}
			}
		}
		return &Number{Value: -1}
	},
	"פצל_לפי": arrayPartition,
	"הסר": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 1 {
			return errArity("הסר", "אינדקס אחד")
		}
		idx, ok := args[0].(*Number)
		if !ok {
			return &Error{Message: "הסר מצפה לאינדקס מספרי"}
		}
		i := int(idx.Value)
		if i < 0 || i >= len(a.Elements) {
			return &Error{Message: fmt.Sprintf("אינדקס מחוץ לטווח: %d", i)}
		}
		removed := a.Elements[i]
		a.Elements = append(a.Elements[:i], a.Elements[i+1:]...)
		return removed
	},
	"מחק_ערך": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 1 {
			return errArity("מחק_ערך", "ערך אחד")
		}
		want := args[0].Inspect()
		for i, e := range a.Elements {
			if e.Inspect() == want {
				a.Elements = append(a.Elements[:i], a.Elements[i+1:]...)
				return &Boolean{Value: true}
			}
		}
		return &Boolean{Value: false}
	},
	"נקה": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("נקה", "0 ארגומנטים")
		}
		a.Elements = nil
		return a
	},
	"העתק": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("העתק", "0 ארגומנטים")
		}
		cp := &Array{Elements: make([]Object, len(a.Elements))}
		copy(cp.Elements, a.Elements)
		return cp
	},
	"העתק_עמוק": arrayDeepCopy,
	"הפוך": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) != 0 {
			return errArity("הפוך", "0 ארגומנטים")
		}
		for i, j := 0, len(a.Elements)-1; i < j; i, j = i+1, j-1 {
			a.Elements[i], a.Elements[j] = a.Elements[j], a.Elements[i]
		}
		return a
	},
	"מיין":         arraySort,
	"מיין_מספרים":  arraySortNumbers,
	"מיין_עם":      arraySortWith,
	"מפה":          arrayMap,
	"סנן":          arrayFilter,
	"מצא":          arrayFind,
	"מצא_אינדקס":   arrayFindIndex,
	"לכל":          arrayForEach,
	"צמצם":         arrayReduce,
	"כל":           arrayEvery,
	"יש_כל":        arraySome,
	"ספור":         arrayCount,
	"הכנס":         arrayInsert,
	"הסר_ראשון":    arrayShift,
	"הוסף_בהתחלה":  arrayUnshift,
	"מלא":          arrayFill,
	"חלק":          arrayChunk,
	"לקח":          arrayTake,
	"השמט":         arrayDrop,
	"ייחודי":       arrayUnique,
	"שטח":          arrayFlatten,
	"ערבב":         arrayShuffle,
	"חתוך": func(self Object, args ...Object) Object {
		a := self.(*Array)
		if len(args) < 1 || len(args) > 2 {
			return errArity("חתוך", "התחלה ואופציונלי סוף")
		}
		startN, ok := args[0].(*Number)
		if !ok {
			return &Error{Message: "חתוך מצפה למספרים"}
		}
		start := int(startN.Value)
		end := len(a.Elements)
		if len(args) == 2 {
			endN, ok := args[1].(*Number)
			if !ok {
				return &Error{Message: "חתוך מצפה למספרים"}
			}
			end = int(endN.Value)
		}
		if start < 0 {
			start = 0
		}
		if end > len(a.Elements) {
			end = len(a.Elements)
		}
		if start > end {
			return &Array{Elements: nil}
		}
		out := make([]Object, end-start)
		copy(out, a.Elements[start:end])
		return &Array{Elements: out}
	},
	"חבר":     arrayConcat,
	"הצטרף":   arrayJoin,
	"סכום":    arraySum,
	"ממוצע":   arrayAvg,
	"מקסימום": arrayMax,
	"מינימום": arrayMin,
}

func arrayConcat(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("חבר", "רשימה אחת")
	}
	other, ok := args[0].(*Array)
	if !ok {
		return &Error{Message: "חבר מצפה לרשימה"}
	}
	a.Elements = append(a.Elements, other.Elements...)
	return a
}

func arrayJoin(self Object, args ...Object) Object {
	a := self.(*Array)
	sep := ""
	if len(args) == 1 {
		if s, ok := args[0].(*String); ok {
			sep = s.Value
		} else {
			sep = args[0].Inspect()
		}
	} else if len(args) > 1 {
		return errArity("הצטרף", "מפריד אופציונלי")
	}
	parts := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		parts[i] = e.Inspect()
	}
	return &String{Value: strings.Join(parts, sep)}
}

func arraySum(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("סכום", "0 ארגומנטים")
	}
	sum := 0.0
	for _, e := range a.Elements {
		n, ok := e.(*Number)
		if !ok {
			return &Error{Message: "סכום עובד רק על רשימת מספרים"}
		}
		sum += n.Value
	}
	return &Number{Value: sum}
}

func arrayAvg(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("ממוצע", "0 ארגומנטים")
	}
	if len(a.Elements) == 0 {
		return &Error{Message: "ממוצע של רשימה ריקה"}
	}
	sumObj := arraySum(self)
	if _, ok := sumObj.(*Error); ok {
		return sumObj
	}
	return &Number{Value: sumObj.(*Number).Value / float64(len(a.Elements))}
}

func arrayMax(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("מקסימום", "0 ארגומנטים")
	}
	if len(a.Elements) == 0 {
		return Nil
	}
	best, ok := a.Elements[0].(*Number)
	if !ok {
		return &Error{Message: "מקסימום עובד רק על רשימת מספרים"}
	}
	for _, e := range a.Elements[1:] {
		n, ok := e.(*Number)
		if !ok {
			return &Error{Message: "מקסימום עובד רק על רשימת מספרים"}
		}
		if n.Value > best.Value {
			best = n
		}
	}
	return best
}

func arrayMin(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("מינימום", "0 ארגומנטים")
	}
	if len(a.Elements) == 0 {
		return Nil
	}
	best, ok := a.Elements[0].(*Number)
	if !ok {
		return &Error{Message: "מינימום עובד רק על רשימת מספרים"}
	}
	for _, e := range a.Elements[1:] {
		n, ok := e.(*Number)
		if !ok {
			return &Error{Message: "מינימום עובד רק על רשימת מספרים"}
		}
		if n.Value < best.Value {
			best = n
		}
	}
	return best
}

func hashDeepCopy(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("העתק_עמוק", "0 ארגומנטים")
	}
	return DeepCopy(self)
}

func hashItems(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 0 {
		return errArity("פריטים", "0 ארגומנטים")
	}
	keys := hashKeys(h)
	out := &Array{Elements: make([]Object, len(keys))}
	for i, k := range keys {
		out.Elements[i] = &Array{Elements: []Object{
			&String{Value: k},
			h.Pairs[k],
		}}
	}
	return out
}

func arrayDeepCopy(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("העתק_עמוק", "0 ארגומנטים")
	}
	return DeepCopy(self)
}

func allNumbers(a *Array) bool {
	for _, e := range a.Elements {
		if _, ok := e.(*Number); !ok {
			return false
		}
	}
	return true
}

func arraySort(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("מיין", "0 ארגומנטים")
	}
	if allNumbers(a) {
		return arraySortNumbers(self)
	}
	sort.SliceStable(a.Elements, func(i, j int) bool {
		return a.Elements[i].Inspect() < a.Elements[j].Inspect()
	})
	return a
}

func arraySortNumbers(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("מיין_מספרים", "0 ארגומנטים")
	}
	for _, e := range a.Elements {
		if _, ok := e.(*Number); !ok {
			return &Error{Message: "מיין_מספרים עובד רק על רשימת מספרים"}
		}
	}
	sort.SliceStable(a.Elements, func(i, j int) bool {
		return a.Elements[i].(*Number).Value < a.Elements[j].(*Number).Value
	})
	return a
}

func arraySortWith(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("מיין_עם", "פונקציית השוואה")
	}
	fn := args[0]
	sort.SliceStable(a.Elements, func(i, j int) bool {
		res := Call(fn, a.Elements[i], a.Elements[j])
		if _, ok := res.(*Error); ok {
			return false
		}
		return Truthy(res)
	})
	return a
}

func arrayMap(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("מפה", "פונקציה אחת")
	}
	fn := args[0]
	out := &Array{Elements: make([]Object, len(a.Elements))}
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		out.Elements[i] = res
	}
	return out
}

func arrayFilter(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("סנן", "פונקציה אחת")
	}
	fn := args[0]
	var out []Object
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			out = append(out, e)
		}
	}
	return &Array{Elements: out}
}

func arrayPartition(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("פצל_לפי", "פונקציה אחת")
	}
	fn := args[0]
	var yes, no []Object
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			yes = append(yes, e)
		} else {
			no = append(no, e)
		}
	}
	return &Array{Elements: []Object{
		&Array{Elements: yes},
		&Array{Elements: no},
	}}
}

func arrayFind(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("מצא", "פונקציה אחת")
	}
	fn := args[0]
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			return e
		}
	}
	return Nil
}

func arrayForEach(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("לכל", "פונקציה אחת")
	}
	fn := args[0]
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
	}
	return a
}

func arrayInsert(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 2 {
		return errArity("הכנס", "אינדקס וערך")
	}
	idx, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "הכנס מצפה לאינדקס מספרי"}
	}
	i := int(idx.Value)
	if i < 0 {
		i = 0
	}
	if i > len(a.Elements) {
		i = len(a.Elements)
	}
	a.Elements = append(a.Elements, nil)
	copy(a.Elements[i+1:], a.Elements[i:])
	a.Elements[i] = args[1]
	return a
}

func arrayUnique(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("ייחודי", "0 ארגומנטים")
	}
	var out []Object
	for _, e := range a.Elements {
		found := false
		for _, u := range out {
			if Equal(e, u) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, e)
		}
	}
	return &Array{Elements: out}
}

func arrayFlatten(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("שטח", "0 ארגומנטים")
	}
	var out []Object
	for _, e := range a.Elements {
		if inner, ok := e.(*Array); ok {
			out = append(out, inner.Elements...)
		} else {
			out = append(out, e)
		}
	}
	return &Array{Elements: out}
}

func arrayShuffle(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("ערבב", "0 ארגומנטים")
	}
	shuffleObjects(a.Elements)
	return a
}

func callHashItem(fn Object, value Object, key string) Object {
	switch fnArity(fn) {
	case 0:
		return Call(fn)
	case 2:
		return Call(fn, value, &String{Value: key})
	default:
		return Call(fn, value)
	}
}

func hashMap(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 1 {
		return errArity("מפה", "פונקציה אחת")
	}
	fn := args[0]
	out := &Hash{Pairs: make(map[string]Object, len(h.Pairs))}
	for _, k := range hashKeys(h) {
		res := callHashItem(fn, h.Pairs[k], k)
		if err, ok := res.(*Error); ok {
			return err
		}
		out.Pairs[k] = res
	}
	return out
}

func hashFilter(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 1 {
		return errArity("סנן", "פונקציה אחת")
	}
	fn := args[0]
	out := &Hash{Pairs: map[string]Object{}}
	for _, k := range hashKeys(h) {
		res := callHashItem(fn, h.Pairs[k], k)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			out.Pairs[k] = h.Pairs[k]
		}
	}
	return out
}

func hashFind(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 1 {
		return errArity("מצא", "פונקציה אחת")
	}
	fn := args[0]
	for _, k := range hashKeys(h) {
		res := callHashItem(fn, h.Pairs[k], k)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			return h.Pairs[k]
		}
	}
	return Nil
}

func hashForEach(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 1 {
		return errArity("לכל", "פונקציה אחת")
	}
	fn := args[0]
	for _, k := range hashKeys(h) {
		res := callHashItem(fn, h.Pairs[k], k)
		if err, ok := res.(*Error); ok {
			return err
		}
	}
	return h
}

func arrayReduce(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 2 {
		return errArity("צמצם", "ערך התחלתי ופונקציה")
	}
	acc := args[0]
	fn := args[1]
	for i, e := range a.Elements {
		var res Object
		switch fnArity(fn) {
		case 3:
			res = Call(fn, acc, e, &Number{Value: float64(i)})
		default:
			res = Call(fn, acc, e)
		}
		if err, ok := res.(*Error); ok {
			return err
		}
		acc = res
	}
	return acc
}

func arrayEvery(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("כל", "פונקציה אחת")
	}
	fn := args[0]
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if !Truthy(res) {
			return &Boolean{Value: false}
		}
	}
	return &Boolean{Value: true}
}

func arraySome(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("יש_כל", "פונקציה אחת")
	}
	fn := args[0]
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			return &Boolean{Value: true}
		}
	}
	return &Boolean{Value: false}
}

func arrayCount(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) == 0 {
		return &Number{Value: float64(len(a.Elements))}
	}
	if len(args) != 1 {
		return errArity("ספור", "פונקציה אופציונלית")
	}
	fn := args[0]
	n := 0
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			n++
		}
	}
	return &Number{Value: float64(n)}
}

func arrayFindIndex(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("מצא_אינדקס", "פונקציה אחת")
	}
	fn := args[0]
	for i, e := range a.Elements {
		res := CallItem(fn, e, i)
		if err, ok := res.(*Error); ok {
			return err
		}
		if Truthy(res) {
			return &Number{Value: float64(i)}
		}
	}
	return &Number{Value: -1}
}

func arrayShift(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 0 {
		return errArity("הסר_ראשון", "0 ארגומנטים")
	}
	if len(a.Elements) == 0 {
		return Nil
	}
	first := a.Elements[0]
	a.Elements = a.Elements[1:]
	return first
}

func arrayUnshift(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) < 1 {
		return errArity("הוסף_בהתחלה", "לפחות ערך אחד")
	}
	a.Elements = append(args, a.Elements...)
	return a
}

func arrayFill(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) < 1 || len(args) > 3 {
		return errArity("מלא", "ערך ואופציונלי התחלה/סוף")
	}
	val := args[0]
	start, end := 0, len(a.Elements)
	if len(args) >= 2 {
		n, ok := args[1].(*Number)
		if !ok {
			return &Error{Message: "מלא מצפה לאינדקסים מספריים"}
		}
		start = int(n.Value)
	}
	if len(args) == 3 {
		n, ok := args[2].(*Number)
		if !ok {
			return &Error{Message: "מלא מצפה לאינדקסים מספריים"}
		}
		end = int(n.Value)
	}
	if start < 0 {
		start = 0
	}
	if end > len(a.Elements) {
		end = len(a.Elements)
	}
	for i := start; i < end; i++ {
		a.Elements[i] = val
	}
	return a
}

func arrayChunk(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("חלק", "גודל חלק")
	}
	n, ok := args[0].(*Number)
	if !ok || int(n.Value) <= 0 {
		return &Error{Message: "חלק מצפה לגודל חיובי"}
	}
	size := int(n.Value)
	var out []Object
	for i := 0; i < len(a.Elements); i += size {
		end := i + size
		if end > len(a.Elements) {
			end = len(a.Elements)
		}
		chunk := make([]Object, end-i)
		copy(chunk, a.Elements[i:end])
		out = append(out, &Array{Elements: chunk})
	}
	return &Array{Elements: out}
}

func arrayTake(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("לקח", "כמות")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "לקח מצפה למספר"}
	}
	k := int(n.Value)
	if k < 0 {
		k = 0
	}
	if k > len(a.Elements) {
		k = len(a.Elements)
	}
	out := make([]Object, k)
	copy(out, a.Elements[:k])
	return &Array{Elements: out}
}

func arrayDrop(self Object, args ...Object) Object {
	a := self.(*Array)
	if len(args) != 1 {
		return errArity("השמט", "כמות")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "השמט מצפה למספר"}
	}
	k := int(n.Value)
	if k < 0 {
		k = 0
	}
	if k > len(a.Elements) {
		k = len(a.Elements)
	}
	out := make([]Object, len(a.Elements)-k)
	copy(out, a.Elements[k:])
	return &Array{Elements: out}
}

func hashHasValue(self Object, args ...Object) Object {
	h := self.(*Hash)
	if len(args) != 1 {
		return errArity("יש_ערך", "ערך אחד")
	}
	for _, v := range h.Pairs {
		if Equal(v, args[0]) {
			return &Boolean{Value: true}
		}
	}
	return &Boolean{Value: false}
}
