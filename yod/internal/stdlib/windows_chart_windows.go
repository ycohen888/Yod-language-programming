//go:build windows

package stdlib

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/object"
)

type chartSeries struct {
	name   string
	values []float64
	color  walk.Color
}

var defaultChartPalette = []walk.Color{
	walk.RGB(37, 99, 235),   // כחול
	walk.RGB(16, 185, 129),  // ירוק־עמירלד
	walk.RGB(249, 115, 22),  // כתום
	walk.RGB(168, 85, 247),  // סגול
	walk.RGB(236, 72, 153),  // ורוד
	walk.RGB(14, 165, 233),  // תכלת
	walk.RGB(234, 179, 8),   // זהב
	walk.RGB(100, 116, 139), // אפור־כחלחל
}

func winCreateChart(args ...object.Object) object.Object {
	kind := "עמודות"
	if len(args) >= 1 {
		s, ok := asString(args[0])
		if !ok {
			return errObj(`חלונות.גרף מצפה לסוג: "עמודות" | "קו" | "עוגה"`)
		}
		k, err := normalizeChartKind(s)
		if err != nil {
			return errObj(err.Error())
		}
		kind = k
	}
	if len(args) > 1 {
		return errObj("חלונות.גרף מצפה ל־0 או 1 ארגומנטים")
	}
	return newChartWidget(kind)
}

func newChartWidget(kind string) object.Object {
	st := &controlState{
		kind:            "גרף",
		chartKind:       kind,
		chartShowLegend: true,
		chartShowGrid:   true,
		listMinH:        320,
		chartLabels:     []string{},
		chartSeries:     nil,
		stretchFactor:   -1,
	}
	w := &object.GuiWidget{Kind: "גרף", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_סוג"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetKind(st, a...)
	}}
	w.Attrs["קבע_כותרת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetTitle(st, a...)
	}}
	w.Attrs["קבע_תוויות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetLabels(st, a...)
	}}
	w.Attrs["קבע_סדרות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetSeries(st, a...)
	}}
	w.Attrs["קבע_נתונים"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetData(st, a...)
	}}
	w.Attrs["קבע_ציר_X"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetAxis(st, true, a...)
	}}
	w.Attrs["קבע_ציר_Y"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetAxis(st, false, a...)
	}}
	w.Attrs["קבע_מקרא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetBool(st, &st.chartShowLegend, "קבע_מקרא", a...)
	}}
	w.Attrs["קבע_רשת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetBool(st, &st.chartShowGrid, "קבע_רשת", a...)
	}}
	w.Attrs["קבע_כהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetBool(st, &st.chartDark, "קבע_כהה", a...)
	}}
	w.Attrs["קבע_תת_כותרת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetSubtitle(st, a...)
	}}
	w.Attrs["קבע_טווח_Y"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetYRange(st, a...)
	}}
	w.Attrs["קבע_ערימה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetBool(st, &st.chartStacked, "קבע_ערימה", a...)
	}}
	w.Attrs["קבע_פורמט_ערכים"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return chartSetValueFormat(st, a...)
	}}
	w.Attrs["קבע_גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("גרף.קבע_גובה מצפה למספר")
		}
		n, ok := a[0].(*object.Number)
		if !ok || n.Value < 80 {
			return errObj("גרף.קבע_גובה מצפה למספר >= 80")
		}
		st.listMinH = int(n.Value)
		return object.Nil
	}}
	w.Attrs["רענן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("גרף.רענן מצפה ל־0 ארגומנטים")
		}
		invalidateChart(st)
		return object.Nil
	}}
	return w
}

func normalizeChartKind(s string) (string, error) {
	switch strings.TrimSpace(s) {
	case "עמודות", "עמודות_אנוכיות", "בר", "bar", "columns":
		return "עמודות", nil
	case "קו", "קווים", "line":
		return "קו", nil
	case "עוגה", "פאי", "pie":
		return "עוגה", nil
	default:
		return "", fmt.Errorf(`סוג גרף לא מוכר: %q (עמודות / קו / עוגה)`, s)
	}
}

func chartSetKind(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף.קבע_סוג מצפה למחרוזת")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("גרף.קבע_סוג מצפה למחרוזת")
	}
	k, err := normalizeChartKind(s)
	if err != nil {
		return errObj(err.Error())
	}
	st.chartKind = k
	invalidateChart(st)
	return object.Nil
}

func chartSetTitle(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף.קבע_כותרת מצפה למחרוזת")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("גרף.קבע_כותרת מצפה למחרוזת")
	}
	st.chartTitle = s
	invalidateChart(st)
	return object.Nil
}

func chartSetSubtitle(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף.קבע_תת_כותרת מצפה למחרוזת")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("גרף.קבע_תת_כותרת מצפה למחרוזת")
	}
	st.chartSubtitle = s
	invalidateChart(st)
	return object.Nil
}

func chartSetYRange(st *controlState, args ...object.Object) object.Object {
	switch len(args) {
	case 1:
		n, ok := args[0].(*object.Number)
		if !ok {
			return errObj("גרף.קבע_טווח_Y מצפה למספר מקס, או מינ ומקס")
		}
		st.chartYMin = 0
		st.chartYMax = n.Value
		st.chartYRangeSet = true
	case 2:
		a, ok1 := args[0].(*object.Number)
		b, ok2 := args[1].(*object.Number)
		if !ok1 || !ok2 {
			return errObj("גרף.קבע_טווח_Y מצפה למספרים")
		}
		st.chartYMin = a.Value
		st.chartYMax = b.Value
		st.chartYRangeSet = true
	default:
		return errObj("גרף.קבע_טווח_Y מצפה למקס, או מינ ומקס")
	}
	if st.chartYMax <= st.chartYMin {
		return errObj("גרף.קבע_טווח_Y: מקס חייב להיות גדול ממינ")
	}
	invalidateChart(st)
	return object.Nil
}

func chartSetAxis(st *controlState, isX bool, args ...object.Object) object.Object {
	name := "קבע_ציר_Y"
	if isX {
		name = "קבע_ציר_X"
	}
	if len(args) != 1 {
		return errObj("גרף." + name + " מצפה למחרוזת")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("גרף." + name + " מצפה למחרוזת")
	}
	if isX {
		st.chartXLabel = s
	} else {
		st.chartYLabel = s
	}
	invalidateChart(st)
	return object.Nil
}

func chartSetBool(st *controlState, dest *bool, method string, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף." + method + " מצפה לאמת/שקר")
	}
	b, ok := args[0].(*object.Boolean)
	if !ok {
		return errObj("גרף." + method + " מצפה לאמת/שקר")
	}
	*dest = b.Value
	invalidateChart(st)
	return object.Nil
}

func chartSetValueFormat(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj(`גרף.קבע_פורמט_ערכים מצפה ל־"" / "רגיל" / "בתים"`)
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj(`גרף.קבע_פורמט_ערכים מצפה למחרוזת`)
	}
	switch strings.TrimSpace(s) {
	case "", "רגיל", "מספר":
		st.chartValueFormat = ""
	case "בתים", "bytes":
		st.chartValueFormat = "בתים"
	default:
		return errObj(`גרף.קבע_פורמט_ערכים: השתמשו ב־"רגיל" או "בתים"`)
	}
	invalidateChart(st)
	return object.Nil
}

func chartSetLabels(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף.קבע_תוויות מצפה לרשימה")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("גרף.קבע_תוויות מצפה לרשימה")
	}
	labels := make([]string, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		if s, ok := asString(el); ok {
			labels = append(labels, s)
		} else {
			labels = append(labels, el.Inspect())
		}
	}
	st.chartLabels = labels
	invalidateChart(st)
	return object.Nil
}

func chartSetSeries(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף.קבע_סדרות מצפה לרשימת מילונים")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("גרף.קבע_סדרות מצפה לרשימה")
	}
	series := make([]chartSeries, 0, len(arr.Elements))
	for i, el := range arr.Elements {
		h, ok := el.(*object.Hash)
		if !ok {
			return errObj("גרף.קבע_סדרות: כל פריט חייב להיות מילון")
		}
		name := fmt.Sprintf("סדרה %d", i+1)
		if v, ok := h.Pairs["שם"]; ok {
			if s, ok := asString(v); ok {
				name = s
			}
		}
		valsObj, ok := h.Pairs["ערכים"]
		if !ok {
			return errObj(`גרף.קבע_סדרות: חסר מפתח "ערכים"`)
		}
		valsArr, ok := valsObj.(*object.Array)
		if !ok {
			return errObj(`גרף.קבע_סדרות: "ערכים" חייב להיות רשימה`)
		}
		vals := make([]float64, 0, len(valsArr.Elements))
		for _, ve := range valsArr.Elements {
			n, ok := ve.(*object.Number)
			if !ok {
				return errObj("גרף.קבע_סדרות: ערכים חייבים להיות מספרים")
			}
			vals = append(vals, n.Value)
		}
		col := defaultChartPalette[i%len(defaultChartPalette)]
		if cObj, ok := h.Pairs["צבע"]; ok {
			if s, ok := asString(cObj); ok {
				if c, err := parseChartColorName(s); err == nil {
					col = c
				}
			} else if arrC, ok := cObj.(*object.Array); ok && len(arrC.Elements) >= 3 {
				if c, err := parseWalkColor(arrC.Elements...); err == nil {
					col = c
				}
			}
		}
		series = append(series, chartSeries{name: name, values: vals, color: col})
	}
	st.chartSeries = series
	invalidateChart(st)
	return object.Nil
}

// קבע_נתונים — קיצור לסדרה אחת / עוגה: [{"שם","ערך"}, ...] או [מספרים]
func chartSetData(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גרף.קבע_נתונים מצפה לרשימה")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("גרף.קבע_נתונים מצפה לרשימה")
	}
	if len(arr.Elements) == 0 {
		st.chartSeries = nil
		st.chartLabels = nil
		invalidateChart(st)
		return object.Nil
	}

	// מילונים עם שם+ערך → עוגה / סדרה אחת עם תוויות
	if _, ok := arr.Elements[0].(*object.Hash); ok {
		labels := make([]string, 0, len(arr.Elements))
		vals := make([]float64, 0, len(arr.Elements))
		colors := make([]walk.Color, 0, len(arr.Elements))
		for i, el := range arr.Elements {
			h, ok := el.(*object.Hash)
			if !ok {
				return errObj("גרף.קבע_נתונים: ערבוב סוגי פריטים")
			}
			name := fmt.Sprintf("%d", i+1)
			if v, ok := h.Pairs["שם"]; ok {
				if s, ok := asString(v); ok {
					name = s
				}
			}
			valObj, ok := h.Pairs["ערך"]
			if !ok {
				return errObj(`גרף.קבע_נתונים: חסר "ערך"`)
			}
			n, ok := valObj.(*object.Number)
			if !ok {
				return errObj(`גרף.קבע_נתונים: "ערך" חייב להיות מספר`)
			}
			col := defaultChartPalette[i%len(defaultChartPalette)]
			if cObj, ok := h.Pairs["צבע"]; ok {
				if s, ok := asString(cObj); ok {
					if c, err := parseChartColorName(s); err == nil {
						col = c
					}
				}
			}
			labels = append(labels, name)
			vals = append(vals, n.Value)
			colors = append(colors, col)
		}
		st.chartLabels = labels
		if st.chartKind == "עוגה" {
			// לכל פלח סדרה נפרדת של ערך בודד — נוח לצביעה במקרא
			series := make([]chartSeries, len(vals))
			for i := range vals {
				series[i] = chartSeries{name: labels[i], values: []float64{vals[i]}, color: colors[i]}
			}
			st.chartSeries = series
		} else {
			st.chartSeries = []chartSeries{{name: "ערכים", values: vals, color: defaultChartPalette[0]}}
		}
		invalidateChart(st)
		return object.Nil
	}

	// רשימת מספרים פשוטה
	vals := make([]float64, 0, len(arr.Elements))
	labels := make([]string, 0, len(arr.Elements))
	for i, el := range arr.Elements {
		n, ok := el.(*object.Number)
		if !ok {
			return errObj("גרף.קבע_נתונים: מצפה למספרים או מילונים")
		}
		vals = append(vals, n.Value)
		labels = append(labels, strconv.Itoa(i+1))
	}
	if len(st.chartLabels) != len(vals) {
		st.chartLabels = labels
	}
	st.chartSeries = []chartSeries{{name: "ערכים", values: vals, color: defaultChartPalette[0]}}
	invalidateChart(st)
	return object.Nil
}

func parseChartColorName(s string) (walk.Color, error) {
	c, err := parseWalkColor(&object.String{Value: s})
	if err == nil {
		return c, nil
	}
	switch strings.TrimSpace(s) {
	case "צהוב", "yellow":
		return walk.RGB(234, 179, 8), nil
	case "סגול", "purple", "violet":
		return walk.RGB(168, 85, 247), nil
	case "ורוד", "pink":
		return walk.RGB(236, 72, 153), nil
	case "תכלת", "cyan", "sky":
		return walk.RGB(14, 165, 233), nil
	case "ירוק_בהיר", "lime":
		return walk.RGB(16, 185, 129), nil
	case "זהב", "gold":
		return walk.RGB(202, 138, 4), nil
	default:
		return 0, err
	}
}

func invalidateChart(st *controlState) {
	if st.chartWidget != nil {
		st.chartWidget.Invalidate()
	}
}

func buildChartWidget(ch *controlState) Widget {
	minH := ch.listMinH
	if minH < 80 {
		minH = 280
	}
	sf := stretchOr(ch.stretchFactor, 1)
	return CustomWidget{
		AssignTo:            &ch.chartWidget,
		MinSize:             Size{Width: 120, Height: minH},
		StretchFactor:       sf,
		InvalidatesOnResize: true,
		PaintMode:           PaintBuffered,
		Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
			return paintChart(ch, canvas, bounds)
		},
	}
}

func paintChart(st *controlState, canvas *walk.Canvas, bounds walk.Rectangle) error {
	if bounds.Width < 40 || bounds.Height < 40 {
		return nil
	}
	theme := chartThemeLight()
	if st.chartDark {
		theme = chartThemeDark()
	}
	bg, err := walk.NewSolidColorBrush(theme.bg)
	if err != nil {
		return err
	}
	defer bg.Dispose()
	if err := canvas.FillRectangle(bg, bounds); err != nil {
		return err
	}

	border, err := walk.NewCosmeticPen(walk.PenSolid, theme.border)
	if err == nil {
		defer border.Dispose()
		_ = canvas.DrawRectangle(border, bounds)
	}

	pad := 14
	titleH := 0
	if st.chartTitle != "" || st.chartSubtitle != "" {
		titleH = 28
		if st.chartSubtitle != "" {
			titleH = 46
		}
		y := bounds.Y + 6
		if st.chartTitle != "" {
			font, err := walk.NewFont("Segoe UI", 12, walk.FontBold)
			if err == nil {
				defer font.Dispose()
				tr := walk.Rectangle{X: bounds.X + pad, Y: y, Width: bounds.Width - pad*2, Height: 22}
				_ = canvas.DrawText(st.chartTitle, font, theme.title, tr,
					walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
				y += 22
			}
		}
		if st.chartSubtitle != "" {
			font, err := walk.NewFont("Segoe UI", 9, 0)
			if err == nil {
				defer font.Dispose()
				tr := walk.Rectangle{X: bounds.X + pad, Y: y, Width: bounds.Width - pad*2, Height: 18}
				_ = canvas.DrawText(st.chartSubtitle, font, theme.tick, tr,
					walk.TextCenter|walk.TextVCenter|walk.TextSingleLine|walk.TextEndEllipsis)
			}
		}
	}

	legendW := 0
	if st.chartShowLegend && len(st.chartSeries) > 0 {
		legendW = 120
		if st.chartKind == "עוגה" {
			legendW = 140
		}
	}

	leftAxis := 44
	if st.chartValueFormat == "בתים" {
		leftAxis = 86
	}

	plot := walk.Rectangle{
		X:      bounds.X + pad + leftAxis,
		Y:      bounds.Y + pad + titleH,
		Width:  bounds.Width - pad*2 - leftAxis - legendW,
		Height: bounds.Height - pad*2 - titleH - 36,
	}
	if plot.Width < 40 || plot.Height < 40 {
		return nil
	}

	switch st.chartKind {
	case "עוגה":
		return paintPieChart(st, canvas, bounds, plot, legendW, theme)
	case "קו":
		return paintXYChart(st, canvas, bounds, plot, legendW, true, theme)
	default:
		return paintXYChart(st, canvas, bounds, plot, legendW, false, theme)
	}
}

type chartTheme struct {
	bg, plotBG, border, title, tick, label, grid, axis, hole, holeRing walk.Color
}

func chartThemeLight() chartTheme {
	return chartTheme{
		bg:       walk.RGB(248, 250, 252),
		plotBG:   walk.RGB(255, 255, 255),
		border:   walk.RGB(226, 232, 240),
		title:    walk.RGB(15, 23, 42),
		tick:     walk.RGB(100, 116, 139),
		label:    walk.RGB(71, 85, 105),
		grid:     walk.RGB(241, 245, 249),
		axis:     walk.RGB(148, 163, 184),
		hole:     walk.RGB(248, 250, 252),
		holeRing: walk.RGB(226, 232, 240),
	}
}

func chartThemeDark() chartTheme {
	return chartTheme{
		bg:       walk.RGB(22, 27, 34),
		plotBG:   walk.RGB(30, 37, 46),
		border:   walk.RGB(48, 54, 61),
		title:    walk.RGB(230, 237, 243),
		tick:     walk.RGB(139, 148, 158),
		label:    walk.RGB(177, 186, 196),
		grid:     walk.RGB(48, 54, 61),
		axis:     walk.RGB(88, 96, 105),
		hole:     walk.RGB(22, 27, 34),
		holeRing: walk.RGB(48, 54, 61),
	}
}

func chartMaxValue(series []chartSeries) float64 {
	return chartMaxValueMode(series, false)
}

func chartMaxValueMode(series []chartSeries, stacked bool) float64 {
	max := 0.0
	if stacked && len(series) > 0 {
		nCats := 0
		for _, s := range series {
			if len(s.values) > nCats {
				nCats = len(s.values)
			}
		}
		for i := 0; i < nCats; i++ {
			sum := 0.0
			for _, s := range series {
				if i < len(s.values) && s.values[i] > 0 {
					sum += s.values[i]
				}
			}
			if sum > max {
				max = sum
			}
		}
	} else {
		for _, s := range series {
			for _, v := range s.values {
				if v > max {
					max = v
				}
			}
		}
	}
	if max <= 0 {
		return 1
	}
	return niceCeiling(max)
}

func niceCeiling(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(v))
	f := v / math.Pow(10, exp)
	var nf float64
	switch {
	case f <= 1:
		nf = 1
	case f <= 2:
		nf = 2
	case f <= 5:
		nf = 5
	default:
		nf = 10
	}
	return nf * math.Pow(10, exp)
}

func paintXYChart(st *controlState, canvas *walk.Canvas, bounds, plot walk.Rectangle, legendW int, isLine bool, theme chartTheme) error {
	stacked := st.chartStacked && !isLine && len(st.chartSeries) > 1
	maxV := chartMaxValueMode(st.chartSeries, stacked)
	minV := 0.0
	if st.chartYRangeSet {
		minV = st.chartYMin
		maxV = st.chartYMax
	}
	span := maxV - minV
	if span <= 0 {
		span = 1
	}
	nCats := len(st.chartLabels)
	for _, s := range st.chartSeries {
		if len(s.values) > nCats {
			nCats = len(s.values)
		}
	}
	if nCats == 0 {
		return drawChartEmpty(canvas, plot, theme)
	}

	plotBG, _ := walk.NewSolidColorBrush(theme.plotBG)
	if plotBG != nil {
		defer plotBG.Dispose()
		_ = canvas.FillRectangle(plotBG, plot)
	}

	axisPen, _ := walk.NewCosmeticPen(walk.PenSolid, theme.axis)
	if axisPen != nil {
		defer axisPen.Dispose()
	}
	gridPen, _ := walk.NewCosmeticPen(walk.PenSolid, theme.grid)
	if gridPen != nil {
		defer gridPen.Dispose()
	}

	tickFont, _ := walk.NewFont("Segoe UI", 8, 0)
	if tickFont != nil {
		defer tickFont.Dispose()
	}
	labelFont, _ := walk.NewFont("Segoe UI", 9, 0)
	if labelFont != nil {
		defer labelFont.Dispose()
	}
	valueFont, _ := walk.NewFont("Segoe UI", 8, 0)
	if valueFont != nil {
		defer valueFont.Dispose()
	}

	fmtVal := func(v float64) string {
		return formatChartNumberFmt(v, st.chartValueFormat)
	}

	// רשת + תוויות Y
	const yTicks = 5
	for i := 0; i <= yTicks; i++ {
		t := float64(i) / float64(yTicks)
		y := plot.Y + plot.Height - int(t*float64(plot.Height))
		if st.chartShowGrid && gridPen != nil && i > 0 && i < yTicks {
			_ = canvas.DrawLine(gridPen, walk.Point{X: plot.X, Y: y}, walk.Point{X: plot.X + plot.Width, Y: y})
		}
		if tickFont != nil {
			val := minV + span*t
			txt := fmtVal(val)
			tr := walk.Rectangle{X: bounds.X + 2, Y: y - 8, Width: plot.X - bounds.X - 6, Height: 16}
			_ = canvas.DrawText(txt, tickFont, theme.tick, tr,
				walk.TextRight|walk.TextVCenter|walk.TextSingleLine|walk.TextEndEllipsis)
		}
	}

	if axisPen != nil {
		_ = canvas.DrawLine(axisPen, walk.Point{X: plot.X, Y: plot.Y}, walk.Point{X: plot.X, Y: plot.Y + plot.Height})
		_ = canvas.DrawLine(axisPen, walk.Point{X: plot.X, Y: plot.Y + plot.Height}, walk.Point{X: plot.X + plot.Width, Y: plot.Y + plot.Height})
	}

	if st.chartYLabel != "" && labelFont != nil {
		tr := walk.Rectangle{X: bounds.X + 2, Y: plot.Y, Width: 48, Height: 18}
		_ = canvas.DrawText(st.chartYLabel, labelFont, theme.label, tr,
			walk.TextLeft|walk.TextSingleLine)
	}

	slotW := float64(plot.Width) / float64(nCats)
	nSeries := len(st.chartSeries)
	if nSeries == 0 {
		return drawChartEmpty(canvas, plot, theme)
	}

	if isLine {
		for si, s := range st.chartSeries {
			pts := make([]walk.Point, 0, len(s.values))
			for i, v := range s.values {
				x := plot.X + int((float64(i)+0.5)*slotW)
				ratio := (v - minV) / span
				if ratio < 0 {
					ratio = 0
				}
				if ratio > 1 {
					ratio = 1
				}
				y := plot.Y + plot.Height - int(ratio*float64(plot.Height))
				pts = append(pts, walk.Point{X: x, Y: y})
			}
			if len(pts) >= 2 {
				brush, _ := walk.NewSolidColorBrush(s.color)
				if brush != nil {
					pen, err := walk.NewGeometricPen(walk.PenSolid, 2, brush)
					if err == nil {
						for i := 0; i < len(pts)-1; i++ {
							_ = canvas.DrawLine(pen, pts[i], pts[i+1])
						}
						pen.Dispose()
					}
					brush.Dispose()
				}
			}
			dot, _ := walk.NewSolidColorBrush(s.color)
			if dot != nil {
				for _, p := range pts {
					r := walk.Rectangle{X: p.X - 4, Y: p.Y - 4, Width: 8, Height: 8}
					_ = canvas.FillEllipse(dot, r)
				}
				dot.Dispose()
			}
			_ = si
		}
	} else if stacked {
		barArea := slotW * 0.58
		barW := barArea
		if barW < 10 {
			barW = 10
		}
		sepPen, _ := walk.NewCosmeticPen(walk.PenSolid, theme.plotBG)
		if sepPen != nil {
			defer sepPen.Dispose()
		}
		for i := 0; i < nCats; i++ {
			base := 0.0
			x0 := plot.X + int(float64(i)*slotW+(slotW-barW)/2)
			topY := plot.Y + plot.Height
			total := 0.0
			segN := 0
			for _, s := range st.chartSeries {
				v := 0.0
				if i < len(s.values) {
					v = s.values[i]
				}
				if v < 0 {
					v = 0
				}
				total += v
				y0 := plot.Y + plot.Height - int(((base+v)-minV)/span*float64(plot.Height))
				y1 := plot.Y + plot.Height - int((base-minV)/span*float64(plot.Height))
				h := y1 - y0
				if h < 1 && v > 0 {
					h = 1
					y0 = y1 - 1
				}
				if h < 0 {
					h = 0
				}
				brush, err := walk.NewSolidColorBrush(s.color)
				if err == nil {
					rect := walk.Rectangle{X: x0, Y: y0, Width: int(barW), Height: h}
					_ = canvas.FillRectangle(brush, rect)
					brush.Dispose()
				}
				if sepPen != nil && segN > 0 && h > 2 {
					_ = canvas.DrawLine(sepPen, walk.Point{X: x0, Y: y1}, walk.Point{X: x0 + int(barW), Y: y1})
				}
				segN++
				base += v
				if y0 < topY {
					topY = y0
				}
			}
			if valueFont != nil && total > 0 {
				txt := fmtVal(total)
				tr := walk.Rectangle{X: x0 - 8, Y: topY - 16, Width: int(barW) + 16, Height: 14}
				_ = canvas.DrawText(txt, valueFont, theme.label, tr,
					walk.TextCenter|walk.TextSingleLine|walk.TextEndEllipsis)
			}
		}
	} else {
		groupGap := 0.18
		barArea := slotW * (1 - groupGap)
		barW := barArea / float64(nSeries)
		if barW < 3 {
			barW = 3
		}
		for si, s := range st.chartSeries {
			brush, err := walk.NewSolidColorBrush(s.color)
			if err != nil {
				continue
			}
			for i, v := range s.values {
				ratio := (v - minV) / span
				if ratio < 0 {
					ratio = 0
				}
				if ratio > 1 {
					ratio = 1
				}
				h := int(ratio * float64(plot.Height))
				if h < 1 && v > 0 {
					h = 1
				}
				x0 := plot.X + int(float64(i)*slotW+slotW*groupGap/2+float64(si)*barW)
				y0 := plot.Y + plot.Height - h
				rect := walk.Rectangle{X: x0, Y: y0, Width: int(barW) - 1, Height: h}
				if rect.Width < 1 {
					rect.Width = 1
				}
				_ = canvas.FillRectangle(brush, rect)
				if valueFont != nil && v > 0 && nSeries == 1 {
					txt := fmtVal(v)
					tr := walk.Rectangle{X: x0 - 6, Y: y0 - 16, Width: rect.Width + 12, Height: 14}
					_ = canvas.DrawText(txt, valueFont, theme.label, tr,
						walk.TextCenter|walk.TextSingleLine|walk.TextEndEllipsis)
				}
			}
			brush.Dispose()
		}
	}

	// תוויות X
	if labelFont != nil {
		for i := 0; i < nCats; i++ {
			label := strconv.Itoa(i + 1)
			if i < len(st.chartLabels) {
				label = st.chartLabels[i]
			}
			x := plot.X + int((float64(i)+0.5)*slotW)
			tr := walk.Rectangle{X: x - int(slotW/2), Y: plot.Y + plot.Height + 4, Width: int(slotW), Height: 20}
			_ = canvas.DrawText(label, labelFont, theme.label, tr,
				walk.TextCenter|walk.TextSingleLine)
		}
	}
	if st.chartXLabel != "" && labelFont != nil {
		tr := walk.Rectangle{X: plot.X, Y: bounds.Y + bounds.Height - 20, Width: plot.Width, Height: 16}
		_ = canvas.DrawText(st.chartXLabel, labelFont, theme.label, tr,
			walk.TextCenter|walk.TextSingleLine)
	}

	return paintChartLegend(st, canvas, bounds, plot, legendW, theme)
}

func paintPieChart(st *controlState, canvas *walk.Canvas, bounds, plot walk.Rectangle, legendW int, theme chartTheme) error {
	values := make([]float64, 0)
	colors := make([]walk.Color, 0)
	names := make([]string, 0)

	if len(st.chartSeries) > 0 && len(st.chartSeries[0].values) == 1 && len(st.chartSeries) > 1 {
		for _, s := range st.chartSeries {
			v := 0.0
			if len(s.values) > 0 {
				v = s.values[0]
			}
			if v < 0 {
				v = 0
			}
			values = append(values, v)
			colors = append(colors, s.color)
			names = append(names, s.name)
		}
	} else if len(st.chartSeries) > 0 {
		s := st.chartSeries[0]
		for i, v := range s.values {
			if v < 0 {
				v = 0
			}
			values = append(values, v)
			colors = append(colors, defaultChartPalette[i%len(defaultChartPalette)])
			name := strconv.Itoa(i + 1)
			if i < len(st.chartLabels) {
				name = st.chartLabels[i]
			}
			names = append(names, name)
		}
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	if sum <= 0 || len(values) == 0 {
		return drawChartEmpty(canvas, plot, theme)
	}

	size := plot.Width
	if plot.Height < size {
		size = plot.Height
	}
	size = int(float64(size) * 0.92)
	if size < 40 {
		size = 40
	}
	cx := size / 2
	cy := size / 2
	r := size/2 - 2

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// רקע שקוף יחסית — ימולא ע״י blitting על רקע הגרף
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	start := -math.Pi / 2
	for i, v := range values {
		sweep := (v / sum) * 2 * math.Pi
		if sweep <= 0 {
			continue
		}
		c := colors[i]
		fillWedgeRGBA(img, cx, cy, r, start, start+sweep, color.RGBA{uint8(c.R()), uint8(c.G()), uint8(c.B()), 255})
		start += sweep
	}
	// חור דונאט
	inner := int(float64(r) * 0.45)
	if inner > 8 {
		drawDisk(img, cx, cy, inner, color.RGBA{theme.hole.R(), theme.hole.G(), theme.hole.B(), 255})
	}
	// טבעת חיצונית עדינה
	drawCircleOutline(img, cx, cy, r, 2, color.RGBA{theme.holeRing.R(), theme.holeRing.G(), theme.holeRing.B(), 255})

	bmp, err := walk.NewBitmapFromImageForDPI(img, 96)
	if err != nil {
		return err
	}
	defer bmp.Dispose()
	ox := plot.X + (plot.Width-size)/2
	oy := plot.Y + (plot.Height-size)/2
	_ = canvas.DrawImageStretchedPixels(bmp, walk.Rectangle{X: ox, Y: oy, Width: size, Height: size})

	if st.chartShowLegend && legendW > 0 {
		font, _ := walk.NewFont("Segoe UI", 9, 0)
		if font != nil {
			defer font.Dispose()
		}
		lx := bounds.X + bounds.Width - legendW - 8
		ly := plot.Y
		for i := range values {
			pct := values[i] / sum * 100
			br, _ := walk.NewSolidColorBrush(colors[i])
			if br != nil {
				_ = canvas.FillRectangle(br, walk.Rectangle{X: lx, Y: ly + 3, Width: 12, Height: 12})
				br.Dispose()
			}
			if font != nil {
				txt := fmt.Sprintf("%s  %.0f%%", names[i], pct)
				tr := walk.Rectangle{X: lx + 18, Y: ly, Width: legendW - 22, Height: 18}
				_ = canvas.DrawText(txt, font, theme.label, tr, walk.TextLeft|walk.TextSingleLine)
			}
			ly += 22
		}
	}
	return nil
}

func fillWedgeRGBA(img *image.RGBA, cx, cy, r int, a0, a1 float64, c color.RGBA) {
	for {
		if a1-a0 >= 2*math.Pi {
			drawDisk(img, cx, cy, r, c)
			return
		}
		if a1 < a0 {
			a1 += 2 * math.Pi
		}
		break
	}
	r2 := float64(r * r)
	minY := cy - r
	maxY := cy + r
	minX := cx - r
	maxX := cx + r
	b := img.Bounds()
	if minY < b.Min.Y {
		minY = b.Min.Y
	}
	if maxY >= b.Max.Y {
		maxY = b.Max.Y - 1
	}
	if minX < b.Min.X {
		minX = b.Min.X
	}
	if maxX >= b.Max.X {
		maxX = b.Max.X - 1
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if dx*dx+dy*dy > r2 {
				continue
			}
			ang := math.Atan2(dy, dx)
			for ang < a0 {
				ang += 2 * math.Pi
			}
			for ang > a0+2*math.Pi {
				ang -= 2 * math.Pi
			}
			if ang >= a0 && ang <= a1 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func paintChartLegend(st *controlState, canvas *walk.Canvas, bounds, plot walk.Rectangle, legendW int, theme chartTheme) error {
	if !st.chartShowLegend || legendW <= 0 || len(st.chartSeries) == 0 {
		return nil
	}
	font, err := walk.NewFont("Segoe UI", 9, 0)
	if err != nil {
		return nil
	}
	defer font.Dispose()
	lx := bounds.X + bounds.Width - legendW - 6
	ly := plot.Y
	for _, s := range st.chartSeries {
		br, err := walk.NewSolidColorBrush(s.color)
		if err == nil {
			_ = canvas.FillRectangle(br, walk.Rectangle{X: lx, Y: ly + 3, Width: 12, Height: 12})
			br.Dispose()
		}
		tr := walk.Rectangle{X: lx + 18, Y: ly, Width: legendW - 22, Height: 18}
		_ = canvas.DrawText(s.name, font, theme.label, tr, walk.TextLeft|walk.TextSingleLine)
		ly += 22
	}
	return nil
}

func drawChartEmpty(canvas *walk.Canvas, plot walk.Rectangle, theme chartTheme) error {
	font, err := walk.NewFont("Segoe UI", 11, 0)
	if err != nil {
		return nil
	}
	defer font.Dispose()
	return canvas.DrawText("אין נתונים לגרף", font, theme.tick, plot,
		walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}

func formatChartNumber(v float64) string {
	return formatChartNumberFmt(v, "")
}

func formatChartNumberFmt(v float64, format string) string {
	if format == "בתים" {
		return formatBytes(v)
	}
	if math.Abs(v) >= 1000 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	if math.Abs(v-math.Round(v)) < 1e-9 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}
