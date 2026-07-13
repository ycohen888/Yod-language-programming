package object

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"yod/internal/console"
)

var stringMethods = map[string]func(self Object, args ...Object) Object{
	"אורך":          stringLen,
	"גודל":          stringLen,
	"ריק":           stringEmpty,
	"הדפס":          stringPrint,
	"למחרוזת":       stringSelf,
	"למספר":         stringToNumber,
	"גזום":          stringTrim,
	"גזום_התחלה":    stringTrimStart,
	"גזום_סוף":      stringTrimEnd,
	"אותיות_גדולות": stringUpper,
	"אותיות_קטנות":  stringLower,
	"הפוך":          stringReverse,
	"חזור":          stringRepeat,
	"מכיל":          stringContains,
	"מתחיל_ב":       stringHasPrefix,
	"מסתיים_ב":      stringHasSuffix,
	"אינדקס_של":     stringIndexOf,
	"אינדקס_אחרון":  stringLastIndexOf,
	"החלף":          stringReplace,
	"פצל":           stringSplit,
	"חתוך":          stringSlice,
	"תו":            stringCharAt,
	"שורות":         stringLines,
	"פורמט":         stringFormat,
	"התאם":          stringMatch,
	"חפש":           stringSearch,
	"החלף_תבנית":    stringReplaceRegex,
	"פצל_תבנית":     stringSplitRegex,
	"שם_קובץ":       stringBaseName,
	"תיקייה":        stringDirName,
	"סיומת":         stringExt,
	"בלי_סיומת":     stringNoExt,
	"מלא_משמאל":     stringPadLeft,
	"מלא_מימין":     stringPadRight,
	"מלא_במרכז":     stringPadCenter,
	"ספור_הופעות":   stringCount,
	"קוד_תו":        stringCharCode,
	"מ_קוד":         stringFromCode,
	"אות_ראשונה_גדולה": stringCapitalize,
}

var numberMethods = map[string]func(self Object, args ...Object) Object{
	"הדפס":      numberPrint,
	"למחרוזת":   numberToString,
	"למספר":     numberSelf,
	"שלם":       numberInt,
	"עיגול":     numberRound,
	"רצפה":      numberFloor,
	"תקרה":      numberCeil,
	"מוחלט":     numberAbs,
	"ערך_מוחלט": numberAbs,
	"שורש":      numberSqrt,
	"חזקה":      numberPow,
	"זוגי":      numberEven,
	"אי_זוגי":   numberOdd,
	"סימן":      numberSign,
	"בין":       numberBetween,
	"הגבל":      numberClamp,
	"מקסימום":   numberMax,
	"מינימום":   numberMin,
	"סינוס":     numberSin,
	"קוסינוס":   numberCos,
	"טנגנס":     numberTan,
	"לוג":       numberLog,
	"לוג10":     numberLog10,
	"שארית":     numberMod,
	"עשרוניות":  numberDecimals,
	"לרדיאנים":  numberToRadians,
	"למעלות":    numberToDegrees,
}

var booleanMethods = map[string]func(self Object, args ...Object) Object{
	"הדפס":    boolPrint,
	"למחרוזת": boolToString,
	"למספר":   boolToNumber,
	"לא":      boolNot,
}

var nullMethods = map[string]func(self Object, args ...Object) Object{
	"הדפס":    nullPrint,
	"למחרוזת": nullToString,
	"ריק":     nullIsEmpty,
}

// LookupMethod — מתודה מובנית לפי טיפוס המקבל
func LookupMethod(recv Object, name string) Object {
	switch recv := recv.(type) {
	case *Hash:
		if fn, ok := hashMethods[name]; ok {
			return &BoundBuiltin{Self: recv, Name: name, Fn: fn}
		}
	case *Array:
		if fn, ok := arrayMethods[name]; ok {
			return &BoundBuiltin{Self: recv, Name: name, Fn: fn}
		}
	case *String:
		if fn, ok := stringMethods[name]; ok {
			return &BoundBuiltin{Self: recv, Name: name, Fn: fn}
		}
	case *Number:
		if fn, ok := numberMethods[name]; ok {
			return &BoundBuiltin{Self: recv, Name: name, Fn: fn}
		}
	case *Boolean:
		if fn, ok := booleanMethods[name]; ok {
			return &BoundBuiltin{Self: recv, Name: name, Fn: fn}
		}
	case *Null:
		if fn, ok := nullMethods[name]; ok {
			return &BoundBuiltin{Self: recv, Name: name, Fn: fn}
		}
	}
	return nil
}

// LookupCollectionMethod — תאימות לאחור
func LookupCollectionMethod(recv Object, name string) Object {
	return LookupMethod(recv, name)
}

func stringLen(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("אורך", "0 ארגומנטים")
	}
	s := self.(*String).Value
	return &Number{Value: float64(utf8.RuneCountInString(s))}
}

func stringEmpty(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("ריק", "0 ארגומנטים")
	}
	return &Boolean{Value: self.(*String).Value == ""}
}

func stringPrint(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("הדפס", "0 ארגומנטים")
	}
	console.Println(self.(*String).Value)
	return Nil
}

func stringSelf(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למחרוזת", "0 ארגומנטים")
	}
	return self
}

func stringToNumber(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למספר", "0 ארגומנטים")
	}
	s := strings.TrimSpace(self.(*String).Value)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return &Error{Message: fmt.Sprintf("לא הצלחתי להמיר %q למספר", s)}
	}
	return &Number{Value: n}
}

func stringTrim(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("גזום", "0 ארגומנטים")
	}
	return &String{Value: strings.TrimSpace(self.(*String).Value)}
}

func stringTrimStart(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("גזום_התחלה", "0 ארגומנטים")
	}
	return &String{Value: strings.TrimLeftFunc(self.(*String).Value, unicode.IsSpace)}
}

func stringTrimEnd(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("גזום_סוף", "0 ארגומנטים")
	}
	return &String{Value: strings.TrimRightFunc(self.(*String).Value, unicode.IsSpace)}
}

func stringUpper(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("אותיות_גדולות", "0 ארגומנטים")
	}
	return &String{Value: strings.ToUpper(self.(*String).Value)}
}

func stringLower(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("אותיות_קטנות", "0 ארגומנטים")
	}
	return &String{Value: strings.ToLower(self.(*String).Value)}
}

func stringReverse(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("הפוך", "0 ארגומנטים")
	}
	r := []rune(self.(*String).Value)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return &String{Value: string(r)}
}

func stringRepeat(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("חזור", "מספר אחד")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "חזור מצפה למספר"}
	}
	times := int(n.Value)
	if times < 0 {
		return &Error{Message: "חזור מצפה למספר לא־שלילי"}
	}
	return &String{Value: strings.Repeat(self.(*String).Value, times)}
}

func stringContains(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("מכיל", "מחרוזת אחת")
	}
	return &Boolean{Value: strings.Contains(self.(*String).Value, args[0].Inspect())}
}

func stringHasPrefix(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("מתחיל_ב", "מחרוזת אחת")
	}
	return &Boolean{Value: strings.HasPrefix(self.(*String).Value, args[0].Inspect())}
}

func stringHasSuffix(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("מסתיים_ב", "מחרוזת אחת")
	}
	return &Boolean{Value: strings.HasSuffix(self.(*String).Value, args[0].Inspect())}
}

func stringIndexOf(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("אינדקס_של", "מחרוזת אחת")
	}
	s := self.(*String).Value
	sub := args[0].Inspect()
	i := strings.Index(s, sub)
	if i < 0 {
		return &Number{Value: -1}
	}
	return &Number{Value: float64(utf8.RuneCountInString(s[:i]))}
}

func stringLastIndexOf(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("אינדקס_אחרון", "מחרוזת אחת")
	}
	s := self.(*String).Value
	sub := args[0].Inspect()
	i := strings.LastIndex(s, sub)
	if i < 0 {
		return &Number{Value: -1}
	}
	return &Number{Value: float64(utf8.RuneCountInString(s[:i]))}
}

func stringReplace(self Object, args ...Object) Object {
	if len(args) != 2 {
		return errArity("החלף", "ישן וחדש")
	}
	return &String{Value: strings.ReplaceAll(self.(*String).Value, args[0].Inspect(), args[1].Inspect())}
}

func stringSplit(self Object, args ...Object) Object {
	sep := ""
	if len(args) == 1 {
		sep = args[0].Inspect()
	} else if len(args) > 1 {
		return errArity("פצל", "מפריד אופציונלי")
	}
	var parts []string
	if sep == "" {
		for _, r := range self.(*String).Value {
			parts = append(parts, string(r))
		}
	} else {
		parts = strings.Split(self.(*String).Value, sep)
	}
	arr := &Array{Elements: make([]Object, len(parts))}
	for i, p := range parts {
		arr.Elements[i] = &String{Value: p}
	}
	return arr
}

func stringSlice(self Object, args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return errArity("חתוך", "התחלה ואופציונלי סוף")
	}
	startN, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "חתוך מצפה למספרים"}
	}
	runes := []rune(self.(*String).Value)
	start := int(startN.Value)
	end := len(runes)
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
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		return &String{Value: ""}
	}
	return &String{Value: string(runes[start:end])}
}

func stringCharAt(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("תו", "אינדקס אחד")
	}
	idx, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "תו מצפה לאינדקס מספרי"}
	}
	runes := []rune(self.(*String).Value)
	i := int(idx.Value)
	if i < 0 || i >= len(runes) {
		return &Error{Message: fmt.Sprintf("אינדקס מחוץ לטווח: %d", i)}
	}
	return &String{Value: string(runes[i])}
}

func stringLines(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("שורות", "0 ארגומנטים")
	}
	s := strings.ReplaceAll(self.(*String).Value, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	parts := strings.Split(s, "\n")
	arr := &Array{Elements: make([]Object, len(parts))}
	for i, p := range parts {
		arr.Elements[i] = &String{Value: p}
	}
	return arr
}

func stringFormat(self Object, args ...Object) Object {
	return FormatString(self.(*String).Value, args)
}

func compileRegex(name string, pattern string) (*regexp.Regexp, Object) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &Error{Message: fmt.Sprintf("%s: תבנית לא תקינה: %v", name, err)}
	}
	return re, nil
}

func stringMatch(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("התאם", "תבנית אחת")
	}
	re, errObj := compileRegex("התאם", args[0].Inspect())
	if errObj != nil {
		return errObj
	}
	return &Boolean{Value: re.MatchString(self.(*String).Value)}
}

func stringSearch(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("חפש", "תבנית אחת")
	}
	re, errObj := compileRegex("חפש", args[0].Inspect())
	if errObj != nil {
		return errObj
	}
	loc := re.FindStringIndex(self.(*String).Value)
	if loc == nil {
		return &Number{Value: -1}
	}
	return &Number{Value: float64(utf8.RuneCountInString(self.(*String).Value[:loc[0]]))}
}

func stringReplaceRegex(self Object, args ...Object) Object {
	if len(args) != 2 {
		return errArity("החלף_תבנית", "תבנית והחלפה")
	}
	re, errObj := compileRegex("החלף_תבנית", args[0].Inspect())
	if errObj != nil {
		return errObj
	}
	return &String{Value: re.ReplaceAllString(self.(*String).Value, args[1].Inspect())}
}

func stringSplitRegex(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("פצל_תבנית", "תבנית אחת")
	}
	re, errObj := compileRegex("פצל_תבנית", args[0].Inspect())
	if errObj != nil {
		return errObj
	}
	parts := re.Split(self.(*String).Value, -1)
	arr := &Array{Elements: make([]Object, len(parts))}
	for i, p := range parts {
		arr.Elements[i] = &String{Value: p}
	}
	return arr
}

func stringBaseName(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("שם_קובץ", "0 ארגומנטים")
	}
	return &String{Value: filepath.Base(self.(*String).Value)}
}

func stringDirName(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("תיקייה", "0 ארגומנטים")
	}
	return &String{Value: filepath.Dir(self.(*String).Value)}
}

func stringExt(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("סיומת", "0 ארגומנטים")
	}
	return &String{Value: filepath.Ext(self.(*String).Value)}
}

func stringNoExt(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("בלי_סיומת", "0 ארגומנטים")
	}
	base := filepath.Base(self.(*String).Value)
	ext := filepath.Ext(base)
	return &String{Value: strings.TrimSuffix(base, ext)}
}

func stringPadLeft(self Object, args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return errArity("מלא_משמאל", "אורך ואופציונלי תו")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "מלא_משמאל מצפה לאורך מספרי"}
	}
	pad := " "
	if len(args) == 2 {
		pad = args[1].Inspect()
		if pad == "" {
			pad = " "
		}
	}
	s := self.(*String).Value
	need := int(n.Value) - utf8.RuneCountInString(s)
	if need <= 0 {
		return self
	}
	var b strings.Builder
	pr := []rune(pad)
	for i := 0; i < need; i++ {
		b.WriteRune(pr[i%len(pr)])
	}
	b.WriteString(s)
	return &String{Value: b.String()}
}

func stringPadRight(self Object, args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return errArity("מלא_מימין", "אורך ואופציונלי תו")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "מלא_מימין מצפה לאורך מספרי"}
	}
	pad := " "
	if len(args) == 2 {
		pad = args[1].Inspect()
		if pad == "" {
			pad = " "
		}
	}
	s := self.(*String).Value
	need := int(n.Value) - utf8.RuneCountInString(s)
	if need <= 0 {
		return self
	}
	var b strings.Builder
	b.WriteString(s)
	pr := []rune(pad)
	for i := 0; i < need; i++ {
		b.WriteRune(pr[i%len(pr)])
	}
	return &String{Value: b.String()}
}

func stringPadCenter(self Object, args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return errArity("מלא_במרכז", "אורך ואופציונלי תו")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "מלא_במרכז מצפה לאורך מספרי"}
	}
	pad := " "
	if len(args) == 2 {
		pad = args[1].Inspect()
		if pad == "" {
			pad = " "
		}
	}
	s := self.(*String).Value
	need := int(n.Value) - utf8.RuneCountInString(s)
	if need <= 0 {
		return self
	}
	left := need / 2
	right := need - left
	pr := []rune(pad)
	var b strings.Builder
	for i := 0; i < left; i++ {
		b.WriteRune(pr[i%len(pr)])
	}
	b.WriteString(s)
	for i := 0; i < right; i++ {
		b.WriteRune(pr[i%len(pr)])
	}
	return &String{Value: b.String()}
}

func stringCount(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("ספור_הופעות", "מחרוזת אחת")
	}
	sub := args[0].Inspect()
	if sub == "" {
		return &Number{Value: float64(utf8.RuneCountInString(self.(*String).Value) + 1)}
	}
	return &Number{Value: float64(strings.Count(self.(*String).Value, sub))}
}

func stringCharCode(self Object, args ...Object) Object {
	if len(args) > 1 {
		return errArity("קוד_תו", "אינדקס אופציונלי")
	}
	runes := []rune(self.(*String).Value)
	if len(runes) == 0 {
		return &Error{Message: "קוד_תו על מחרוזת ריקה"}
	}
	i := 0
	if len(args) == 1 {
		n, ok := args[0].(*Number)
		if !ok {
			return &Error{Message: "קוד_תו מצפה לאינדקס"}
		}
		i = int(n.Value)
	}
	if i < 0 || i >= len(runes) {
		return &Error{Message: fmt.Sprintf("אינדקס מחוץ לטווח: %d", i)}
	}
	return &Number{Value: float64(runes[i])}
}

func stringFromCode(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("מ_קוד", "קוד אחד")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "מ_קוד מצפה למספר"}
	}
	return &String{Value: string(rune(int(n.Value)))}
}

func stringCapitalize(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("אות_ראשונה_גדולה", "0 ארגומנטים")
	}
	runes := []rune(self.(*String).Value)
	if len(runes) == 0 {
		return self
	}
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return &String{Value: string(runes)}
}

func numberPrint(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("הדפס", "0 ארגומנטים")
	}
	console.Println(self.Inspect())
	return Nil
}

func numberToString(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למחרוזת", "0 ארגומנטים")
	}
	return &String{Value: self.Inspect()}
}

func numberSelf(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למספר", "0 ארגומנטים")
	}
	return self
}

func numberInt(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("שלם", "0 ארגומנטים")
	}
	return &Number{Value: float64(int64(self.(*Number).Value))}
}

func numberRound(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("עיגול", "0 ארגומנטים")
	}
	return &Number{Value: math.Round(self.(*Number).Value)}
}

func numberFloor(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("רצפה", "0 ארגומנטים")
	}
	return &Number{Value: math.Floor(self.(*Number).Value)}
}

func numberCeil(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("תקרה", "0 ארגומנטים")
	}
	return &Number{Value: math.Ceil(self.(*Number).Value)}
}

func numberAbs(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("מוחלט", "0 ארגומנטים")
	}
	return &Number{Value: math.Abs(self.(*Number).Value)}
}

func numberSqrt(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("שורש", "0 ארגומנטים")
	}
	v := self.(*Number).Value
	if v < 0 {
		return &Error{Message: "שורש של מספר שלילי"}
	}
	return &Number{Value: math.Sqrt(v)}
}

func numberPow(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("חזקה", "מעריך אחד")
	}
	exp, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "חזקה מצפה למספר"}
	}
	return &Number{Value: math.Pow(self.(*Number).Value, exp.Value)}
}

func numberEven(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("זוגי", "0 ארגומנטים")
	}
	v := self.(*Number).Value
	return &Boolean{Value: v == float64(int64(v)) && int64(v)%2 == 0}
}

func numberOdd(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("אי_זוגי", "0 ארגומנטים")
	}
	v := self.(*Number).Value
	return &Boolean{Value: v == float64(int64(v)) && int64(v)%2 != 0}
}

func numberSign(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("סימן", "0 ארגומנטים")
	}
	v := self.(*Number).Value
	switch {
	case v > 0:
		return &Number{Value: 1}
	case v < 0:
		return &Number{Value: -1}
	default:
		return &Number{Value: 0}
	}
}

func numberBetween(self Object, args ...Object) Object {
	if len(args) != 2 {
		return errArity("בין", "שני מספרים")
	}
	a, ok1 := args[0].(*Number)
	b, ok2 := args[1].(*Number)
	if !ok1 || !ok2 {
		return &Error{Message: "בין מצפה למספרים"}
	}
	v := self.(*Number).Value
	lo, hi := a.Value, b.Value
	if lo > hi {
		lo, hi = hi, lo
	}
	return &Boolean{Value: v >= lo && v <= hi}
}

func numberMax(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("מקסימום", "מספר אחד")
	}
	other, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "מקסימום מצפה למספר"}
	}
	if self.(*Number).Value >= other.Value {
		return self
	}
	return other
}

func numberMin(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("מינימום", "מספר אחד")
	}
	other, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "מינימום מצפה למספר"}
	}
	if self.(*Number).Value <= other.Value {
		return self
	}
	return other
}

func numberClamp(self Object, args ...Object) Object {
	if len(args) != 2 {
		return errArity("הגבל", "מינימום ומקסימום")
	}
	minN, ok1 := args[0].(*Number)
	maxN, ok2 := args[1].(*Number)
	if !ok1 || !ok2 {
		return &Error{Message: "הגבל מצפה לשני מספרים"}
	}
	lo, hi := minN.Value, maxN.Value
	if lo > hi {
		lo, hi = hi, lo
	}
	v := self.(*Number).Value
	if v < lo {
		v = lo
	} else if v > hi {
		v = hi
	}
	return &Number{Value: v}
}

func numberToRadians(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("לרדיאנים", "0 ארגומנטים")
	}
	return &Number{Value: self.(*Number).Value * math.Pi / 180}
}

func numberToDegrees(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למעלות", "0 ארגומנטים")
	}
	return &Number{Value: self.(*Number).Value * 180 / math.Pi}
}

func numberSin(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("סינוס", "0 ארגומנטים")
	}
	return &Number{Value: math.Sin(self.(*Number).Value)}
}

func numberCos(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("קוסינוס", "0 ארגומנטים")
	}
	return &Number{Value: math.Cos(self.(*Number).Value)}
}

func numberTan(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("טנגנס", "0 ארגומנטים")
	}
	return &Number{Value: math.Tan(self.(*Number).Value)}
}

func numberLog(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("לוג", "0 ארגומנטים")
	}
	v := self.(*Number).Value
	if v <= 0 {
		return &Error{Message: "לוג מוגדר רק למספר חיובי"}
	}
	return &Number{Value: math.Log(v)}
}

func numberLog10(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("לוג10", "0 ארגומנטים")
	}
	v := self.(*Number).Value
	if v <= 0 {
		return &Error{Message: "לוג10 מוגדר רק למספר חיובי"}
	}
	return &Number{Value: math.Log10(v)}
}

func numberMod(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("שארית", "מחלק אחד")
	}
	other, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "שארית מצפה למספר"}
	}
	if other.Value == 0 {
		return &Error{Message: "חילוק באפס"}
	}
	return &Number{Value: math.Mod(self.(*Number).Value, other.Value)}
}

func numberDecimals(self Object, args ...Object) Object {
	if len(args) != 1 {
		return errArity("עשרוניות", "מספר ספרות")
	}
	n, ok := args[0].(*Number)
	if !ok {
		return &Error{Message: "עשרוניות מצפה למספר"}
	}
	digits := int(n.Value)
	if digits < 0 {
		digits = 0
	}
	pow := math.Pow(10, float64(digits))
	return &Number{Value: math.Round(self.(*Number).Value*pow) / pow}
}

func boolPrint(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("הדפס", "0 ארגומנטים")
	}
	console.Println(self.Inspect())
	return Nil
}

func boolToString(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למחרוזת", "0 ארגומנטים")
	}
	return &String{Value: self.Inspect()}
}

func boolToNumber(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למספר", "0 ארגומנטים")
	}
	if self.(*Boolean).Value {
		return &Number{Value: 1}
	}
	return &Number{Value: 0}
}

func boolNot(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("לא", "0 ארגומנטים")
	}
	return &Boolean{Value: !self.(*Boolean).Value}
}

func nullPrint(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("הדפס", "0 ארגומנטים")
	}
	console.Println("ריק")
	return Nil
}

func nullToString(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("למחרוזת", "0 ארגומנטים")
	}
	return &String{Value: "ריק"}
}

func nullIsEmpty(self Object, args ...Object) Object {
	if len(args) != 0 {
		return errArity("ריק", "0 ארגומנטים")
	}
	return &Boolean{Value: true}
}
