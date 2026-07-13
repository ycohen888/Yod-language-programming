package stdlib

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"yod/internal/object"
)

// NewNumbersModule — עיצוב מספרים לתצוגה (פסיקים, כסף, זיכרון…).
func NewNumbersModule() *object.Module {
	m := &object.Module{Name: "מספרים", Attrs: map[string]object.Object{}}
	m.Attrs["מפריד"] = &object.Builtin{Fn: numThousands}
	m.Attrs["עשרוני"] = &object.Builtin{Fn: numDecimal}
	m.Attrs["כסף"] = &object.Builtin{Fn: numMoney}
	m.Attrs["בתים"] = &object.Builtin{Fn: numBytes}
	m.Attrs["אחוז"] = &object.Builtin{Fn: numPercent}
	m.Attrs["שלם"] = &object.Builtin{Fn: numWhole}
	return m
}

func asNumberArg(args []object.Object, name string) (float64, object.Object) {
	if len(args) < 1 {
		return 0, errObj(name + " מצפה למספר")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return 0, errObj(name + " מצפה למספר")
	}
	return n.Value, nil
}

// מספרים.מפריד(מספר) — 1234567 → "1,234,567"
func numThousands(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מספרים.מפריד מצפה למספר אחד")
	}
	n, err := asNumberArg(args, "מספרים.מפריד")
	if err != nil {
		return err
	}
	return &object.String{Value: formatThousands(n, 0)}
}

// מספרים.שלם(מספר) — עיגול + מפריד אלפים
func numWhole(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מספרים.שלם מצפה למספר אחד")
	}
	n, err := asNumberArg(args, "מספרים.שלם")
	if err != nil {
		return err
	}
	return &object.String{Value: formatThousands(math.Round(n), 0)}
}

// מספרים.עשרוני(מספר, [ספרות=2])
func numDecimal(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("מספרים.עשרוני מצפה למספר ואופציונלי ספרות אחרי הנקודה")
	}
	n, err := asNumberArg(args, "מספרים.עשרוני")
	if err != nil {
		return err
	}
	digits := 2
	if len(args) == 2 {
		d, ok := args[1].(*object.Number)
		if !ok || d.Value < 0 {
			return errObj("מספרים.עשרוני: ספרות חייבות להיות מספר לא־שלילי")
		}
		digits = int(d.Value)
	}
	return &object.String{Value: formatThousands(n, digits)}
}

// מספרים.כסף(מספר, [סמל="₪"], [ספרות=2]) — "₪1,234.50"
func numMoney(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 3 {
		return errObj("מספרים.כסף מצפה למספר, אופציונלי סמל מטבע, ואופציונלי ספרות")
	}
	n, err := asNumberArg(args, "מספרים.כסף")
	if err != nil {
		return err
	}
	symbol := "₪"
	digits := 2
	if len(args) >= 2 {
		if s, ok := asString(args[1]); ok {
			symbol = s
		} else if d, ok := args[1].(*object.Number); ok {
			digits = int(d.Value)
		} else {
			return errObj("מספרים.כסף: ארגומנט שני הוא סמל מטבע או מספר ספרות")
		}
	}
	if len(args) == 3 {
		d, ok := args[2].(*object.Number)
		if !ok || d.Value < 0 {
			return errObj("מספרים.כסף: ספרות חייבות להיות מספר לא־שלילי")
		}
		digits = int(d.Value)
	}
	return &object.String{Value: symbol + formatThousands(n, digits)}
}

// מספרים.בתים(בתים) — "5.3 מגה" / "1.2 גיגה" / "512 קילו"
func numBytes(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מספרים.בתים מצפה למספר בתים")
	}
	n, err := asNumberArg(args, "מספרים.בתים")
	if err != nil {
		return err
	}
	return &object.String{Value: formatBytes(n)}
}

// מספרים.אחוז(מספר, [ספרות=0]) — "45%" / "12.5%"
func numPercent(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("מספרים.אחוז מצפה למספר ואופציונלי ספרות")
	}
	n, err := asNumberArg(args, "מספרים.אחוז")
	if err != nil {
		return err
	}
	digits := 0
	if len(args) == 2 {
		d, ok := args[1].(*object.Number)
		if !ok || d.Value < 0 {
			return errObj("מספרים.אחוז: ספרות חייבות להיות מספר לא־שלילי")
		}
		digits = int(d.Value)
	}
	return &object.String{Value: formatThousands(n, digits) + "%"}
}

func formatThousands(n float64, digits int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	if digits < 0 {
		digits = 0
	}
	factor := math.Pow(10, float64(digits))
	n = math.Round(n*factor) / factor

	intPart := int64(math.Floor(n + 1e-9))
	frac := n - float64(intPart)
	if frac < 0 {
		frac = 0
	}

	s := strconv.FormatInt(intPart, 10)
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if digits > 0 {
		fracStr := fmt.Sprintf("%.*f", digits, frac)
		// "0.50" → ".50"
		if idx := strings.IndexByte(fracStr, '.'); idx >= 0 {
			b.WriteString(fracStr[idx:])
		}
	}
	return b.String()
}

func formatBytes(bytes float64) string {
	if bytes < 0 {
		bytes = 0
	}
	units := []string{"בתים", "קילו", "מגה", "גיגה", "טרה"}
	v := bytes
	u := 0
	for v >= 1024 && u < len(units)-1 {
		v /= 1024
		u++
	}
	if u == 0 {
		return formatThousands(math.Round(v), 0) + " " + units[u]
	}
	if v >= 100 {
		return formatThousands(math.Round(v), 0) + " " + units[u]
	}
	if v >= 10 {
		return formatThousands(v, 1) + " " + units[u]
	}
	return formatThousands(v, 2) + " " + units[u]
}
