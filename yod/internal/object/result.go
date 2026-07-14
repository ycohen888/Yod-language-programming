package object

// ResultOk בונה מילון תוצאה מוצלחת.
func ResultOk(value Object) *Hash {
	h := NewHash()
	h.Set("הצלחה", &Boolean{Value: true})
	h.Set("ערך", value)
	h.Set("הודעה", &String{Value: ""})
	h.Set("קוד", &Number{Value: 0})
	return h
}

// ResultErr בונה מילון תוצאה כושלת.
func ResultErr(msg string, code float64) *Hash {
	h := NewHash()
	h.Set("הצלחה", &Boolean{Value: false})
	h.Set("ערך", Nil)
	h.Set("הודעה", &String{Value: msg})
	h.Set("קוד", &Number{Value: code})
	return h
}

// MatchesType בודק אם ערך תואם להערת טיפוס (מספר/מחרוזת/…/שם מחלקה).
func MatchesType(obj Object, typeName string) bool {
	if typeName == "" {
		return true
	}
	switch typeName {
	case "מספר":
		_, ok := obj.(*Number)
		return ok
	case "מחרוזת":
		_, ok := obj.(*String)
		return ok
	case "בוליאני":
		_, ok := obj.(*Boolean)
		return ok
	case "רשימה":
		_, ok := obj.(*Array)
		return ok
	case "מילון":
		_, ok := obj.(*Hash)
		return ok
	case "ריק":
		_, ok := obj.(*Null)
		return ok
	case "תוצאה":
		h, ok := obj.(*Hash)
		if !ok {
			return false
		}
		_, has := h.Get("הצלחה")
		return has
	case "משימה":
		_, ok := obj.(*Task)
		return ok
	default:
		inst, ok := obj.(*Instance)
		if !ok {
			return false
		}
		for c := inst.Class; c != nil; c = c.Parent {
			if c.Name == typeName {
				return true
			}
		}
		return false
	}
}
