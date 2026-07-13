package stdlib

import (
	"sort"

	"yod/internal/object"
)

// מערכת.תהליכים() — רשימת תהליכים פעילים (ממוינים לפי זיכרון, מהגדול)
func sysProcesses(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.תהליכים מצפה ל־0 ארגומנטים")
	}
	list, err := listOSProcesses()
	if err != nil {
		return errObj("מערכת.תהליכים נכשל: " + err.Error())
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].WorkingSet > list[j].WorkingSet
	})
	out := make([]object.Object, 0, len(list))
	const mb = 1024 * 1024
	for _, p := range list {
		out = append(out, &object.Hash{Pairs: map[string]object.Object{
			"מזהה":        &object.Number{Value: float64(p.PID)},
			"שם":          &object.String{Value: p.Name},
			"נתיב":        &object.String{Value: p.Path},
			"זיכרון_בתים": &object.Number{Value: float64(p.WorkingSet)},
			"זיכרון_מגה":  &object.Number{Value: float64(p.WorkingSet / mb)},
		}})
	}
	return &object.Array{Elements: out}
}

// מערכת.סיים_תהליך(מזהה) — מסיים תהליך; מחזיר מילון הצלחה/שגיאה (בלי לזרוק)
func sysKillProcess(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מערכת.סיים_תהליך מצפה למזהה מספרי")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("מערכת.סיים_תהליך מצפה למספר")
	}
	pid := uint32(n.Value)
	if pid == 0 {
		return &object.Hash{Pairs: map[string]object.Object{
			"הצלחה": &object.Boolean{Value: false},
			"שגיאה": &object.String{Value: "לא ניתן לסיים מזהה 0"},
		}}
	}
	if err := killOSProcess(pid); err != nil {
		return &object.Hash{Pairs: map[string]object.Object{
			"הצלחה": &object.Boolean{Value: false},
			"שגיאה":  &object.String{Value: err.Error()},
		}}
	}
	return &object.Hash{Pairs: map[string]object.Object{
		"הצלחה": &object.Boolean{Value: true},
		"שגיאה":  &object.String{Value: ""},
	}}
}

// מערכת.שימוש_מעבד() — אחוז שימוש כולל במעבד (0–100)
func sysCPUUsage(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.שימוש_מעבד מצפה ל־0 ארגומנטים")
	}
	pct, ok := systemCPUPercent()
	if !ok {
		return errObj("לא הצלחתי לקרוא שימוש מעבד")
	}
	return &object.Number{Value: pct}
}

// מערכת.זמן_פעיל() — שניות מאז עליית המערכת
func sysUptime(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.זמן_פעיל מצפה ל־0 ארגומנטים")
	}
	sec, ok := systemUptimeSeconds()
	if !ok {
		return errObj("לא הצלחתי לקרוא זמן פעיל")
	}
	return &object.Number{Value: float64(sec)}
}

// מערכת.כוננים() — רשימת כוננים עם נפח
func sysDrives(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.כוננים מצפה ל־0 ארגומנטים")
	}
	list, err := listOSDrives()
	if err != nil {
		return errObj("מערכת.כוננים נכשל: " + err.Error())
	}
	const mb = 1024 * 1024
	out := make([]object.Object, 0, len(list))
	for _, d := range list {
		used := d.Total - d.Free
		if d.Total < d.Free {
			used = 0
		}
		out = append(out, &object.Hash{Pairs: map[string]object.Object{
			"אות":        &object.String{Value: d.Letter},
			"סהכ_בתים":   &object.Number{Value: float64(d.Total)},
			"פנוי_בתים":  &object.Number{Value: float64(d.Free)},
			"בשימוש_בתים": &object.Number{Value: float64(used)},
			"סהכ_מגה":    &object.Number{Value: float64(d.Total / mb)},
			"פנוי_מגה":   &object.Number{Value: float64(d.Free / mb)},
			"בשימוש_מגה": &object.Number{Value: float64(used / mb)},
		}})
	}
	return &object.Array{Elements: out}
}

type osProcess struct {
	PID        uint32
	Name       string
	WorkingSet uint64
	Path       string
}

type osDrive struct {
	Letter string
	Total  uint64
	Free   uint64
}
