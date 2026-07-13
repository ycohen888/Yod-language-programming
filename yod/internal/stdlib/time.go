package stdlib

import (
	"time"

	"yod/internal/object"
)

func NewTimeModule() *object.Module {
	m := &object.Module{Name: "זמן", Attrs: map[string]object.Object{}}
	m.Attrs["עכשיו"] = &object.Builtin{Fn: timeNow}
	m.Attrs["חותמת"] = &object.Builtin{Fn: timeStamp}
	m.Attrs["תאריך"] = &object.Builtin{Fn: timeDate}
	m.Attrs["שעה"] = &object.Builtin{Fn: timeClock}
	m.Attrs["המתן"] = &object.Builtin{Fn: timeSleep}
	m.Attrs["פורמט"] = &object.Builtin{Fn: timeFormat}
	m.Attrs["פרסר"] = &object.Builtin{Fn: timeParse}
	m.Attrs["הוסף"] = &object.Builtin{Fn: timeAdd}
	m.Attrs["הפרש"] = &object.Builtin{Fn: timeDiff}
	m.Attrs["יום_בשבוע"] = &object.Builtin{Fn: timeWeekday}
	m.Attrs["שנה"] = &object.Builtin{Fn: timeYear}
	m.Attrs["חודש"] = &object.Builtin{Fn: timeMonth}
	m.Attrs["יום"] = &object.Builtin{Fn: timeDay}
	m.Attrs["שעה_מספר"] = &object.Builtin{Fn: timeHour}
	m.Attrs["דקה"] = &object.Builtin{Fn: timeMinute}
	m.Attrs["שנייה"] = &object.Builtin{Fn: timeSecond}
	return m
}

func timeNow(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("זמן.עכשיו מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: time.Now().Format("2006-01-02 15:04:05")}
}

func timeStamp(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("זמן.חותמת מצפה ל־0 ארגומנטים")
	}
	return &object.Number{Value: float64(time.Now().Unix())}
}

func timeDate(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("זמן.תאריך מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: time.Now().Format("2006-01-02")}
}

func timeClock(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("זמן.שעה מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: time.Now().Format("15:04:05")}
}

func timeSleep(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("זמן.המתן מצפה למספר שניות")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("זמן.המתן מצפה למספר")
	}
	if n.Value < 0 {
		return errObj("זמן.המתן מצפה למספר לא־שלילי")
	}
	time.Sleep(time.Duration(n.Value * float64(time.Second)))
	return object.Nil
}

func timeFormat(args ...object.Object) object.Object {
	switch len(args) {
	case 1:
		layout, ok := asString(args[0])
		if !ok {
			return errObj("זמן.פורמט מצפה לתבנית מחרוזת")
		}
		return &object.String{Value: time.Now().Format(layout)}
	case 2:
		ts, ok1 := args[0].(*object.Number)
		layout, ok2 := asString(args[1])
		if !ok1 || !ok2 {
			return errObj("זמן.פורמט מצפה לחותמת ותבנית")
		}
		t := time.Unix(int64(ts.Value), 0)
		return &object.String{Value: t.Format(layout)}
	default:
		return errObj("זמן.פורמט מצפה לתבנית או לחותמת+תבנית")
	}
}

func timeParse(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("זמן.פרסר מצפה למחרוזת ואופציונלי פורמט")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("זמן.פרסר מצפה למחרוזת")
	}
	layout := "2006-01-02 15:04:05"
	if len(args) == 2 {
		l, ok := asString(args[1])
		if !ok {
			return errObj("זמן.פרסר: פורמט חייב להיות מחרוזת")
		}
		layout = l
	}
	t, err := time.ParseInLocation(layout, s, time.Local)
	if err != nil && len(args) == 1 {
		// ניסיונות נפוצים כשלא סופק פורמט
		for _, alt := range []string{"2006-01-02", "02/01/2006", "02/01/2006 15:04:05", time.RFC3339} {
			if t2, e2 := time.ParseInLocation(alt, s, time.Local); e2 == nil {
				t, err = t2, nil
				break
			}
		}
	}
	if err != nil {
		return errObj("זמן.פרסר נכשל: " + err.Error())
	}
	return &object.Number{Value: float64(t.Unix())}
}

func timeAdd(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("זמן.הוסף מצפה לחותמת ולמספר שניות")
	}
	ts, ok1 := args[0].(*object.Number)
	sec, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("זמן.הוסף מצפה לשני מספרים")
	}
	return &object.Number{Value: ts.Value + sec.Value}
}

func timeDiff(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("זמן.הפרש מצפה לשתי חותמות")
	}
	a, ok1 := args[0].(*object.Number)
	b, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("זמן.הפרש מצפה לשני מספרים")
	}
	return &object.Number{Value: a.Value - b.Value}
}

var hebrewWeekdays = []string{"ראשון", "שני", "שלישי", "רביעי", "חמישי", "שישי", "שבת"}

func timeFromStamp(args []object.Object, name string) (time.Time, *object.Error) {
	if len(args) != 1 {
		return time.Time{}, errObj(name + " מצפה לחותמת אחת")
	}
	ts, ok := args[0].(*object.Number)
	if !ok {
		return time.Time{}, errObj(name + " מצפה למספר")
	}
	return time.Unix(int64(ts.Value), 0).Local(), nil
}

func timeWeekday(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.יום_בשבוע")
	if err != nil {
		return err
	}
	return &object.String{Value: hebrewWeekdays[t.Weekday()]}
}

func timeYear(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.שנה")
	if err != nil {
		return err
	}
	return &object.Number{Value: float64(t.Year())}
}

func timeMonth(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.חודש")
	if err != nil {
		return err
	}
	return &object.Number{Value: float64(t.Month())}
}

func timeDay(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.יום")
	if err != nil {
		return err
	}
	return &object.Number{Value: float64(t.Day())}
}

func timeHour(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.שעה_מספר")
	if err != nil {
		return err
	}
	return &object.Number{Value: float64(t.Hour())}
}

func timeMinute(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.דקה")
	if err != nil {
		return err
	}
	return &object.Number{Value: float64(t.Minute())}
}

func timeSecond(args ...object.Object) object.Object {
	t, err := timeFromStamp(args, "זמן.שנייה")
	if err != nil {
		return err
	}
	return &object.Number{Value: float64(t.Second())}
}
