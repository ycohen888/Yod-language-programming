package stdlib

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"

	"yod/internal/object"
)

func NewSpreadsheetModule() *object.Module {
	m := &object.Module{Name: "גיליון", Attrs: map[string]object.Object{}}
	m.Attrs["חדש"] = &object.Builtin{Fn: spreadsheetNew}
	m.Attrs["פתח"] = &object.Builtin{Fn: spreadsheetOpen}
	return m
}

type spreadsheetBook struct {
	mu      sync.Mutex
	file    *excelize.File
	sheet   string
	path    string
	closed  bool
}

func spreadsheetNew(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("גיליון.חדש מצפה ל־0 ארגומנטים")
	}
	f := excelize.NewFile()
	sheets := f.GetSheetList()
	sheet := "Sheet1"
	if len(sheets) > 0 {
		sheet = sheets[0]
	}
	return newSpreadsheetHandle(&spreadsheetBook{file: f, sheet: sheet})
}

func spreadsheetOpen(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("גיליון.פתח מצפה לנתיב אחד")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("גיליון.פתח מצפה למחרוזת נתיב")
	}
	path = resolveAppPath(path)
	f, err := excelize.OpenFile(path)
	if err != nil {
		return errObj("פתיחת גיליון נכשלה: " + err.Error())
	}
	sheets := f.GetSheetList()
	sheet := "Sheet1"
	if len(sheets) > 0 {
		sheet = sheets[0]
	}
	return newSpreadsheetHandle(&spreadsheetBook{file: f, sheet: sheet, path: path})
}

func newSpreadsheetHandle(b *spreadsheetBook) *object.Module {
	m := &object.Module{Name: "קובץ_גיליון", Attrs: map[string]object.Object{}}
	m.Attrs["בחר"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetSelect(b, args...)
	}}
	m.Attrs["שמות_גיליונות"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetSheetNames(b, args...)
	}}
	m.Attrs["קרא_תא"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetGetCell(b, args...)
	}}
	m.Attrs["קבע_תא"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetSetCell(b, args...)
	}}
	m.Attrs["קבע_סגנון"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetSetStyle(b, args...)
	}}
	m.Attrs["גרף"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetAddChart(b, args...)
	}}
	m.Attrs["שמור"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetSave(b, args...)
	}}
	m.Attrs["שמור_כ"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetSaveAs(b, args...)
	}}
	m.Attrs["סגור"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return spreadsheetClose(b, args...)
	}}
	return m
}

func spreadsheetSelect(b *spreadsheetBook, args ...object.Object) object.Object {
	if err := expectArgs("קובץ_גיליון.בחר", 1, args); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	if n, ok := args[0].(*object.Number); ok {
		idx := int(n.Value)
		list := b.file.GetSheetList()
		if idx < 0 || idx >= len(list) {
			return errObj(fmt.Sprintf("אינדקס גיליון לא תקין: %d", idx))
		}
		b.sheet = list[idx]
		return &object.Null{}
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("בחר מצפה לשם גיליון (מחרוזת) או אינדקס (מספר)")
	}
	idx, err := b.file.GetSheetIndex(name)
	if err != nil || idx < 0 {
		return errObj("גיליון לא נמצא: " + name)
	}
	b.sheet = name
	return &object.Null{}
}

func spreadsheetSheetNames(b *spreadsheetBook, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שמות_גיליונות מצפה ל־0 ארגומנטים")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	list := b.file.GetSheetList()
	els := make([]object.Object, len(list))
	for i, s := range list {
		els[i] = &object.String{Value: s}
	}
	return &object.Array{Elements: els}
}

func spreadsheetGetCell(b *spreadsheetBook, args ...object.Object) object.Object {
	if err := expectArgs("קרא_תא", 1, args); err != nil {
		return err
	}
	cell, ok := asString(args[0])
	if !ok {
		return errObj("קרא_תא מצפה לכתובת תא (למשל \"A1\")")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	val, err := b.file.GetCellValue(b.sheet, cell)
	if err != nil {
		return errObj("קריאת תא נכשלה: " + err.Error())
	}
	if val == "" {
		return &object.String{Value: ""}
	}
	if n, err := strconv.ParseFloat(val, 64); err == nil {
		return &object.Number{Value: n}
	}
	return &object.String{Value: val}
}

func spreadsheetSetCell(b *spreadsheetBook, args ...object.Object) object.Object {
	if err := expectArgs("קבע_תא", 2, args); err != nil {
		return err
	}
	cell, ok := asString(args[0])
	if !ok {
		return errObj("קבע_תא מצפה לכתובת תא (למשל \"A1\")")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	var err error
	switch v := args[1].(type) {
	case *object.Number:
		err = b.file.SetCellValue(b.sheet, cell, v.Value)
	case *object.Boolean:
		err = b.file.SetCellValue(b.sheet, cell, v.Value)
	case *object.Null:
		err = b.file.SetCellValue(b.sheet, cell, "")
	case *object.String:
		err = b.file.SetCellValue(b.sheet, cell, v.Value)
	default:
		err = b.file.SetCellValue(b.sheet, cell, args[1].Inspect())
	}
	if err != nil {
		return errObj("כתיבת תא נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func spreadsheetSetStyle(b *spreadsheetBook, args ...object.Object) object.Object {
	if err := expectArgs("קבע_סגנון", 2, args); err != nil {
		return err
	}
	cell, ok := asString(args[0])
	if !ok {
		return errObj("קבע_סגנון מצפה לכתובת תא")
	}
	h, ok := args[1].(*object.Hash)
	if !ok {
		return errObj("קבע_סגנון מצפה למילון סגנון")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	st := &excelize.Style{}
	font := &excelize.Font{}
	hasFont := false
	if hashBool(h, "מודגש", false) {
		font.Bold = true
		hasFont = true
	}
	if n := hashInt(h, "גודל", 0); n > 0 {
		font.Size = float64(n)
		hasFont = true
	}
	if c := strings.TrimPrefix(hashStr(h, "צבע"), "#"); c != "" {
		font.Color = c
		hasFont = true
	}
	if hasFont {
		st.Font = font
	}
	if bg := strings.TrimPrefix(hashStr(h, "רקע"), "#"); bg != "" {
		st.Fill = excelize.Fill{Type: "pattern", Color: []string{bg}, Pattern: 1}
	}
	if align := hashStr(h, "יישור"); align != "" {
		hAlign := "left"
		switch align {
		case "מרכז", "center":
			hAlign = "center"
		case "ימין", "right":
			hAlign = "right"
		case "שמאל", "left":
			hAlign = "left"
		}
		st.Alignment = &excelize.Alignment{Horizontal: hAlign, Vertical: "center"}
	}
	styleID, err := b.file.NewStyle(st)
	if err != nil {
		return errObj("יצירת סגנון נכשלה: " + err.Error())
	}
	if err := b.file.SetCellStyle(b.sheet, cell, cell, styleID); err != nil {
		return errObj("החלת סגנון נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func spreadsheetAddChart(b *spreadsheetBook, args ...object.Object) object.Object {
	if err := expectArgs("גרף", 1, args); err != nil {
		return err
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("גרף מצפה למילון (סוג, נתונים, מיקום, …)")
	}
	kind := hashStr(h, "סוג")
	if kind == "" {
		kind = "עמודות"
	}
	dataRange := hashStr(h, "נתונים")
	if dataRange == "" {
		return errObj("גרף: חסר מפתח \"נתונים\" (למשל \"A1:B12\")")
	}
	pos := hashStr(h, "מיקום")
	if pos == "" {
		pos = "E2"
	}
	title := hashStr(h, "כותרת")

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}

	chartType, errMsg := spreadsheetChartType(kind)
	if errMsg != "" {
		return errObj(errMsg)
	}

	cats, vals, seriesName, errMsg := spreadsheetSplitDataRange(b.sheet, dataRange)
	if errMsg != "" {
		return errObj(errMsg)
	}

	ch := &excelize.Chart{
		Type: chartType,
		Series: []excelize.ChartSeries{{
			Name:       seriesName,
			Categories: cats,
			Values:     vals,
		}},
	}
	if title != "" {
		ch.Title = excelize.ChartTitle{Paragraph: []excelize.RichTextRun{{Text: title}}}
	}
	if err := b.file.AddChart(b.sheet, pos, ch); err != nil {
		return errObj("הוספת גרף נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func spreadsheetChartType(kind string) (excelize.ChartType, string) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "עמודות", "col", "column", "עמוד":
		return excelize.Col, ""
	case "קו", "line":
		return excelize.Line, ""
	case "עוגה", "pie":
		return excelize.Pie, ""
	default:
		return 0, "סוג גרף לא נתמך: " + kind + " (עמודות/קו/עוגה)"
	}
}

// spreadsheetSplitDataRange — טווח דו־עמודתי A1:B12 → קטגוריות בעמודה A, ערכים ב־B.
func spreadsheetSplitDataRange(sheet, rng string) (cats, vals, seriesName, errMsg string) {
	parts := strings.Split(rng, ":")
	if len(parts) != 2 {
		return "", "", "", "טווח נתונים חייב להיות בצורה A1:B12"
	}
	startCol, startRow, err1 := excelize.CellNameToCoordinates(parts[0])
	endCol, endRow, err2 := excelize.CellNameToCoordinates(parts[1])
	if err1 != nil || err2 != nil {
		return "", "", "", "טווח נתונים לא תקין"
	}
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}
	if startCol > endCol {
		startCol, endCol = endCol, startCol
	}
	dataStart := startRow
	if endRow > startRow {
		dataStart = startRow + 1
	}
	catCol := startCol
	valCol := endCol
	if valCol == catCol {
		return "", "", "", "נדרשות לפחות שתי עמודות (קטגוריה וערך)"
	}
	c1, _ := excelize.CoordinatesToCellName(catCol, dataStart)
	c2, _ := excelize.CoordinatesToCellName(catCol, endRow)
	v1, _ := excelize.CoordinatesToCellName(valCol, dataStart)
	v2, _ := excelize.CoordinatesToCellName(valCol, endRow)
	hdr, _ := excelize.CoordinatesToCellName(valCol, startRow)
	cats = fmt.Sprintf("%s!%s", sheet, absRange(c1, c2))
	vals = fmt.Sprintf("%s!%s", sheet, absRange(v1, v2))
	seriesName = fmt.Sprintf("%s!%s", sheet, absCell(hdr))
	return cats, vals, seriesName, ""
}

func absCell(cell string) string {
	col, row, err := excelize.CellNameToCoordinates(cell)
	if err != nil {
		return cell
	}
	name, _ := excelize.ColumnNumberToName(col)
	return fmt.Sprintf("$%s$%d", name, row)
}

func absRange(a, b string) string {
	return absCell(a) + ":" + absCell(b)
}

func spreadsheetSave(b *spreadsheetBook, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שמור מצפה ל־0 ארגומנטים (או השתמשו ב־שמור_כ)")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	if b.path == "" {
		return errObj("אין נתיב שמירה — השתמשו ב־שמור_כ(נתיב)")
	}
	if err := b.file.Save(); err != nil {
		return errObj("שמירה נכשלה: " + err.Error())
	}
	return &object.Null{}
}

func spreadsheetSaveAs(b *spreadsheetBook, args ...object.Object) object.Object {
	if err := expectArgs("שמור_כ", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("שמור_כ מצפה לנתיב מחרוזת")
	}
	path = resolveAppPath(path)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errObj("הגיליון כבר סגור")
	}
	if err := b.file.SaveAs(path); err != nil {
		return errObj("שמירה נכשלה: " + err.Error())
	}
	b.path = path
	return &object.Null{}
}

func spreadsheetClose(b *spreadsheetBook, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("סגור מצפה ל־0 ארגומנטים")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return &object.Null{}
	}
	if err := b.file.Close(); err != nil {
		return errObj("סגירה נכשלה: " + err.Error())
	}
	b.closed = true
	return &object.Null{}
}
