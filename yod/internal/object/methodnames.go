package object

import "sort"

func mapKeys(m map[string]func(self Object, args ...Object) Object) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// StringMethodNames — שמות מתודות מחרוזת (להשלמה בעורך)
func StringMethodNames() []string { return mapKeys(stringMethods) }

// NumberMethodNames — שמות מתודות מספר
func NumberMethodNames() []string { return mapKeys(numberMethods) }

// BooleanMethodNames — שמות מתודות בוליאני
func BooleanMethodNames() []string { return mapKeys(booleanMethods) }

// NullMethodNames — שמות מתודות ריק
func NullMethodNames() []string { return mapKeys(nullMethods) }

// ArrayMethodNames — שמות מתודות רשימה
func ArrayMethodNames() []string { return mapKeys(arrayMethods) }

// HashMethodNames — שמות מתודות מילון
func HashMethodNames() []string { return mapKeys(hashMethods) }

// AllMethodNames — איחוד ייחודי של כל המתודות המובנות
func AllMethodNames() []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]string{
		StringMethodNames(),
		NumberMethodNames(),
		BooleanMethodNames(),
		NullMethodNames(),
		ArrayMethodNames(),
		HashMethodNames(),
	} {
		for _, n := range list {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}
