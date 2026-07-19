package stdlib

import "yod/internal/object"

// NewChartsModule — גרפים מקצועיים לחלון (עמודות / קו / עוגה).
// הרכיבים הם GuiWidget ומוסיפים עם חלונות.חלון.הוסף.
// חלונות.גרף הוא אותו יישום (winCreateChart) — ספרייה זו היא ה-API המומלץ.
func NewChartsModule() *object.Module {
	m := &object.Module{Name: "גרפים", Attrs: map[string]object.Object{}}
	m.Attrs["גרף"] = &object.Builtin{Fn: chartsCreate}
	m.Attrs["עמודות"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return errObj("גרפים.עמודות מצפה ל־0 ארגומנטים")
		}
		return newChartWidgetSafe("עמודות")
	}}
	m.Attrs["קו"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return errObj("גרפים.קו מצפה ל־0 ארגומנטים")
		}
		return newChartWidgetSafe("קו")
	}}
	m.Attrs["עוגה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return errObj("גרפים.עוגה מצפה ל־0 ארגומנטים")
		}
		return newChartWidgetSafe("עוגה")
	}}
	return m
}

func chartsCreate(args ...object.Object) object.Object {
	return winCreateChartBridge(args...)
}
