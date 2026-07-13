//go:build windows

package stdlib

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/object"
)

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// yodTableRow — שורת תצוגה + ערכי מיון ומקור לקרא_שורה.
type yodTableRow struct {
	display []string
	sortKey []interface{} // float64 או string
	source  map[string]object.Object
}

type yodTableModel struct {
	walk.TableModelBase
	walk.SorterBase
	fields      []string
	items       []yodTableRow
	hasUserSort bool
}

func (m *yodTableModel) RowCount() int { return len(m.items) }

func (m *yodTableModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.items) || col < 0 || col >= len(m.items[row].display) {
		return ""
	}
	return m.items[row].display[col]
}

func (m *yodTableModel) SortedColumn() int {
	if !m.hasUserSort {
		return -1
	}
	return m.SorterBase.SortedColumn()
}

func (m *yodTableModel) Sort(col int, order walk.SortOrder) error {
	if col < 0 || col >= len(m.fields) {
		return m.SorterBase.Sort(col, order)
	}
	sort.SliceStable(m.items, func(i, j int) bool {
		less := cellLess(m.items[i].sortKey[col], m.items[j].sortKey[col])
		if order == walk.SortAscending {
			return less
		}
		return !less
	})
	// רק אחרי מיון עם נתונים — אחרת Create/SetModel קורא Sort(0) על טבלה ריקה
	// ואז SortChanged מצייר מחדש בלי LVM_SETITEMCOUNT → טבלה נשארת ריקה.
	if len(m.items) > 0 {
		m.hasUserSort = true
	}
	return m.SorterBase.Sort(col, order)
}

func (m *yodTableModel) sortInPlace(col int, order walk.SortOrder) {
	if col < 0 || col >= len(m.fields) {
		return
	}
	sort.SliceStable(m.items, func(i, j int) bool {
		less := cellLess(m.items[i].sortKey[col], m.items[j].sortKey[col])
		if order == walk.SortAscending {
			return less
		}
		return !less
	})
}

func cellLess(a, b interface{}) bool {
	af, aOK := a.(float64)
	bf, bOK := b.(float64)
	if aOK && bOK {
		return af < bf
	}
	as, _ := a.(string)
	bs, _ := b.(string)
	if !aOK {
		as = fmt.Sprint(a)
	}
	if !bOK {
		bs = fmt.Sprint(b)
	}
	return strings.ToLower(as) < strings.ToLower(bs)
}

func winCreateTable(args ...object.Object) object.Object {
	if len(args) > 0 {
		return errObj("חלונות.טבלה מצפה ל־0 ארגומנטים")
	}
	model := &yodTableModel{fields: []string{}, items: []yodTableRow{}}
	st := &controlState{
		kind:       "טבלה",
		listMinH:   280,
		tableModel: model,
		tableCols:  []int{},
	}
	w := &object.GuiWidget{Kind: "טבלה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_שדות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return tableSetFields(st, a...)
	}}
	w.Attrs["קבע_שורות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return tableSetRows(st, a...)
	}}
	w.Attrs["מיין"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return tableSort(st, a...)
	}}
	w.Attrs["קרא_אינדקס"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("טבלה.קרא_אינדקס מצפה ל־0 ארגומנטים")
		}
		if st.tableView == nil {
			return &object.Number{Value: -1}
		}
		return &object.Number{Value: float64(st.tableView.CurrentIndex())}
	}}
	w.Attrs["קרא_שורה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("טבלה.קרא_שורה מצפה ל־0 ארגומנטים")
		}
		return selectedTableRow(st)
	}}
	w.Attrs["קרא_שדה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("טבלה.קרא_שדה מצפה לשם שדה")
		}
		name, ok := asString(a[0])
		if !ok {
			return errObj("טבלה.קרא_שדה מצפה למחרוזת")
		}
		h := selectedTableRow(st)
		if v, ok := h.Pairs[name]; ok {
			return v
		}
		return object.Nil
	}}
	w.Attrs["קבע_גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("טבלה.קבע_גובה מצפה למספר")
		}
		n, ok := a[0].(*object.Number)
		if !ok || n.Value < 40 {
			return errObj("טבלה.קבע_גובה מצפה למספר >= 40")
		}
		st.listMinH = int(n.Value)
		return object.Nil
	}}
	w.Attrs["קבע_רוחב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return tableSetWidths(st, a...)
	}}
	w.Attrs["קבע_כהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("טבלה.קבע_כהה מצפה לאמת/שקר")
		}
		b, ok := a[0].(*object.Boolean)
		if !ok {
			return errObj("טבלה.קבע_כהה מצפה לאמת/שקר")
		}
		st.tableDark = b.Value
		if st.tableView != nil {
			applyTableDarkColors(st)
			st.tableView.Invalidate()
		}
		return object.Nil
	}}
	w.Attrs["בבחירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("טבלה.בבחירה מצפה לפונקציה")
		}
		st.onSelect = a[0]
		return object.Nil
	}}
	w.Attrs["סנן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return tableFilter(st, a...)
	}}
	w.Attrs["ייצא_csv"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return tableExportCSV(st, a...)
	}}
	return w
}

func selectedTableRow(st *controlState) *object.Hash {
	idx := -1
	if st.tableView != nil {
		idx = st.tableView.CurrentIndex()
	}
	if st.tableModel == nil || idx < 0 || idx >= len(st.tableModel.items) {
		return &object.Hash{Pairs: map[string]object.Object{}}
	}
	src := st.tableModel.items[idx].source
	pairs := make(map[string]object.Object, len(src))
	for k, v := range src {
		if k == "מיון" {
			continue
		}
		pairs[k] = v
	}
	return &object.Hash{Pairs: pairs}
}

func tableSetFields(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טבלה.קבע_שדות מצפה לרשימת שמות")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("טבלה.קבע_שדות מצפה לרשימה")
	}
	fields := make([]string, 0, len(arr.Elements))
	widths := make([]int, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		switch v := el.(type) {
		case *object.String:
			fields = append(fields, v.Value)
			widths = append(widths, 0)
		case *object.Hash:
			name := ""
			w := 0
			if s, ok := v.Pairs["שם"]; ok {
				if ss, ok := asString(s); ok {
					name = ss
				}
			}
			if name == "" {
				if s, ok := v.Pairs["שדה"]; ok {
					if ss, ok := asString(s); ok {
						name = ss
					}
				}
			}
			if n, ok := v.Pairs["רוחב"].(*object.Number); ok && n.Value > 0 {
				w = int(n.Value)
			}
			if name == "" {
				return errObj("טבלה.קבע_שדות: מילון שדה חייב שם")
			}
			fields = append(fields, name)
			widths = append(widths, w)
		default:
			return errObj("טבלה.קבע_שדות מצפה למחרוזות או מילונים")
		}
	}
	if len(fields) == 0 {
		return errObj("טבלה.קבע_שדות מצפה לפחות שדה אחד")
	}
	st.tableModel.fields = fields
	st.tableCols = widths
	st.tableModel.items = nil
	if st.tableView != nil {
		rebuildTableColumns(st)
		_ = st.tableView.SetModel(st.tableModel)
		st.tableModel.PublishRowsReset()
	}
	return object.Nil
}

func tableSetWidths(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טבלה.קבע_רוחב מצפה לרשימת מספרים")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("טבלה.קבע_רוחב מצפה לרשימה")
	}
	if st.tableCols == nil || len(st.tableCols) != len(st.tableModel.fields) {
		st.tableCols = make([]int, len(st.tableModel.fields))
	}
	for i := 0; i < len(arr.Elements) && i < len(st.tableCols); i++ {
		if n, ok := arr.Elements[i].(*object.Number); ok && n.Value > 0 {
			st.tableCols[i] = int(n.Value)
		}
	}
	if st.tableView != nil {
		rebuildTableColumns(st)
	}
	return object.Nil
}

func tableSetRows(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טבלה.קבע_שורות מצפה לרשימה")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("טבלה.קבע_שורות מצפה לרשימה")
	}
	if len(st.tableModel.fields) == 0 {
		return errObj("טבלה.קבע_שורות: קודם קבע_שדות")
	}
	rows := make([]yodTableRow, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		row, err := parseTableRow(st.tableModel.fields, el)
		if err != nil {
			return errObj(err.Error())
		}
		rows = append(rows, row)
	}
	st.tableAllRows = rows
	applyTableFilter(st)
	return object.Nil
}

func tableFilter(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טבלה.סנן מצפה למחרוזת")
	}
	s, ok := asString(args[0])
	if !ok {
		if n, ok := args[0].(*object.Number); ok {
			s = strconv.FormatFloat(n.Value, 'f', -1, 64)
		} else {
			return errObj("טבלה.סנן מצפה למחרוזת")
		}
	}
	st.tableFilter = strings.TrimSpace(s)
	applyTableFilter(st)
	return object.Nil
}

func applyTableFilter(st *controlState) {
	if st.tableModel == nil {
		return
	}
	src := st.tableAllRows
	if src == nil {
		src = st.tableModel.items
	}
	q := strings.ToLower(st.tableFilter)
	var rows []yodTableRow
	if q == "" {
		rows = append([]yodTableRow(nil), src...)
	} else {
		rows = make([]yodTableRow, 0, len(src))
		for _, r := range src {
			match := false
			for _, cell := range r.display {
				if strings.Contains(strings.ToLower(cell), q) {
					match = true
					break
				}
			}
			if match {
				rows = append(rows, r)
			}
		}
	}
	st.tableModel.items = rows
	if st.tableModel.hasUserSort {
		col := st.tableModel.SorterBase.SortedColumn()
		order := st.tableModel.SortOrder()
		st.tableModel.sortInPlace(col, order)
	}
	st.tableModel.PublishRowsReset()
	if st.tableView != nil {
		_ = st.tableView.SetCurrentIndex(-1)
	}
}

func tableExportCSV(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טבלה.ייצא_csv מצפה לנתיב קובץ")
	}
	path, ok := asString(args[0])
	if !ok || strings.TrimSpace(path) == "" {
		return errObj("טבלה.ייצא_csv מצפה לנתיב מחרוזת")
	}
	if st.tableModel == nil || len(st.tableModel.fields) == 0 {
		return errObj("טבלה.ייצא_csv: אין שדות")
	}
	var b strings.Builder
	b.WriteString("\ufeff") // BOM ל־Excel
	for i, f := range st.tableModel.fields {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(csvEscape(f))
	}
	b.WriteByte('\n')
	for _, row := range st.tableModel.items {
		for i, cell := range row.display {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(csvEscape(cell))
		}
		b.WriteByte('\n')
	}
	if err := writeFileUTF8(path, b.String()); err != nil {
		return errObj("טבלה.ייצא_csv נכשל: " + err.Error())
	}
	return object.Nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func writeFileUTF8(path, content string) error {
	return osWriteFile(path, []byte(content))
}

func parseTableRow(fields []string, el object.Object) (yodTableRow, error) {
	display := make([]string, len(fields))
	sortKey := make([]interface{}, len(fields))
	source := make(map[string]object.Object, len(fields)+2)

	switch v := el.(type) {
	case *object.Hash:
		for k, val := range v.Pairs {
			source[k] = val
		}
		for i, f := range fields {
			val, ok := v.Pairs[f]
			if !ok {
				display[i] = ""
				sortKey[i] = ""
				continue
			}
			d, s := cellFromObject(val)
			display[i] = d
			sortKey[i] = s
		}
		// מפתחות מיון אופציונליים: שורה["מיון"] = {"זיכרון": 12345}
		if sortMap, ok := v.Pairs["מיון"].(*object.Hash); ok {
			for i, f := range fields {
				if sk, ok := sortMap.Pairs[f]; ok {
					if n, ok := sk.(*object.Number); ok {
						sortKey[i] = n.Value
					} else if s, ok := asString(sk); ok {
						sortKey[i] = s
					}
				}
			}
		}
	case *object.Array:
		for i := 0; i < len(fields); i++ {
			if i >= len(v.Elements) {
				display[i] = ""
				sortKey[i] = ""
				continue
			}
			d, s := cellFromObject(v.Elements[i])
			display[i] = d
			sortKey[i] = s
			source[fields[i]] = v.Elements[i]
		}
	default:
		return yodTableRow{}, fmt.Errorf("טבלה.קבע_שורות מצפה למילונים או רשימות")
	}
	return yodTableRow{display: display, sortKey: sortKey, source: source}, nil
}

func cellFromObject(o object.Object) (display string, sortKey interface{}) {
	switch v := o.(type) {
	case *object.Number:
		if v.Value == math.Trunc(v.Value) && !math.IsInf(v.Value, 0) && !math.IsNaN(v.Value) {
			display = strconv.FormatInt(int64(math.Round(v.Value)), 10)
		} else {
			display = strconv.FormatFloat(v.Value, 'f', -1, 64)
		}
		return display, v.Value
	case *object.String:
		return v.Value, v.Value
	case *object.Boolean:
		if v.Value {
			return "אמת", "אמת"
		}
		return "שקר", "שקר"
	case *object.Null:
		return "", ""
	default:
		s := o.Inspect()
		return s, s
	}
}

func tableSort(st *controlState, args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj(`טבלה.מיין מצפה לשם שדה, ואופציונלי "עולה"/"יורד"`)
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("טבלה.מיין מצפה לשם שדה")
	}
	col := -1
	for i, f := range st.tableModel.fields {
		if f == name {
			col = i
			break
		}
	}
	if col < 0 {
		return errObj("טבלה.מיין: שדה לא נמצא: " + name)
	}
	order := walk.SortAscending
	if len(args) == 2 {
		dir, ok := asString(args[1])
		if !ok {
			return errObj(`טבלה.מיין: כיוון חייב להיות "עולה" או "יורד"`)
		}
		switch strings.TrimSpace(dir) {
		case "עולה", "asc", "ASC":
			order = walk.SortAscending
		case "יורד", "desc", "DESC":
			order = walk.SortDescending
		default:
			return errObj(`טבלה.מיין: כיוון חייב להיות "עולה" או "יורד"`)
		}
	}
	if err := st.tableModel.Sort(col, order); err != nil {
		return errObj("טבלה.מיין נכשל: " + err.Error())
	}
	if st.tableView != nil {
		_ = st.tableView.SetCurrentIndex(-1)
	}
	return object.Nil
}

func rebuildTableColumns(st *controlState) {
	if st.tableView == nil {
		return
	}
	cols := st.tableView.Columns()
	for cols.Len() > 0 {
		_ = cols.Remove(cols.At(0))
	}
	for i, title := range st.tableModel.fields {
		c := walk.NewTableViewColumn()
		_ = c.SetTitle(title)
		w := 120
		if i < len(st.tableCols) && st.tableCols[i] > 0 {
			w = st.tableCols[i]
		} else if i == 0 {
			w = 220
		}
		_ = c.SetWidth(w)
		_ = cols.Add(c)
	}
}

func buildTableWidget(ch *controlState) Widget {
	minH := ch.listMinH
	if minH < 40 {
		minH = 200
	}
	if ch.tableModel == nil {
		ch.tableModel = &yodTableModel{}
	}
	cols := make([]TableViewColumn, 0, len(ch.tableModel.fields))
	for i, title := range ch.tableModel.fields {
		w := 120
		if i < len(ch.tableCols) && ch.tableCols[i] > 0 {
			w = ch.tableCols[i]
		} else if i == 0 {
			w = 220
		}
		cols = append(cols, TableViewColumn{Title: title, Width: w})
	}
	if len(cols) == 0 {
		cols = []TableViewColumn{{Title: " ", Width: 100}}
	}
	tv := TableView{
		AssignTo:            &ch.tableView,
		AlternatingRowBG:    true,
		ColumnsOrderable:    true,
		LastColumnStretched: true,
		Columns:             cols,
		Model:               ch.tableModel,
		StretchFactor:       1,
		MinSize:             Size{Width: 200, Height: minH},
		Font:                Font{Family: "Segoe UI", PointSize: 10},
		OnCurrentIndexChanged: func() {
			if ch.onSelect != nil {
				invokeYod(ch.onSelect, nil)
			}
		},
		StyleCell: func(style *walk.CellStyle) {
			styleTableCell(ch, style)
		},
	}
	if ch.tableDark {
		tv.Background = SolidColorBrush{Color: darkPanelBG()}
		tv.CustomHeaderHeight = 30
	}
	return tv
}

func styleTableCell(ch *controlState, style *walk.CellStyle) {
	if ch == nil || !ch.tableDark {
		return
	}
	darkBG := darkCtlBG()
	darkAlt := darkCtlAlt()
	darkText := darkCtlText()
	darkMuted := darkCtlMuted()
	darkSel := darkSelBG()
	darkSelText := walk.RGB(255, 255, 255)
	headerBG := darkHeaderBG()

	row := style.Row()
	if row == -1 {
		style.BackgroundColor = headerBG
		style.TextColor = darkText
		if canvas := style.Canvas(); canvas != nil {
			bounds := style.Bounds()
			brush, err := walk.NewSolidColorBrush(headerBG)
			if err == nil {
				defer brush.Dispose()
				_ = canvas.FillRectangle(brush, bounds)
			}
			title := ""
			if ch.tableModel != nil && style.Col() >= 0 && style.Col() < len(ch.tableModel.fields) {
				title = ch.tableModel.fields[style.Col()]
			}
			if title != "" {
				font, err := walk.NewFont("Segoe UI", 9, walk.FontBold)
				if err == nil {
					defer font.Dispose()
					pad := bounds
					pad.X += 8
					pad.Width -= 10
					_ = canvas.DrawText(title, font, darkText, pad,
						walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
				}
			}
		}
		return
	}

	selected := false
	if ch.tableView != nil && ch.tableView.CurrentIndex() == row {
		selected = true
	}
	if selected {
		style.BackgroundColor = darkSel
		style.TextColor = darkSelText
		return
	}
	if row%2 == 1 {
		style.BackgroundColor = darkAlt
	} else {
		style.BackgroundColor = darkBG
	}
	if style.Col() >= 1 {
		style.TextColor = darkMuted
	} else {
		style.TextColor = darkText
	}
}

func applyTableDarkColors(st *controlState) {
	if st == nil || st.tableView == nil || !st.tableDark {
		return
	}
	brush, err := walk.NewSolidColorBrush(darkPanelBG())
	if err == nil {
		st.tableView.SetBackground(brush)
	}
	st.tableView.SetAlternatingRowBG(true)
	applyDarkThemeToWidget(st.tableView)
	// כותרות עמודות + סקרולים
	if hwnd := st.tableView.Handle(); hwnd != 0 {
		applyDarkChrome(hwnd)
	}
}
