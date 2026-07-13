//go:build windows

package stdlib

import (
	"fmt"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/object"
)

type windowState struct {
	title    string
	width    int
	height   int
	children []*controlState
}

type controlState struct {
	kind      string // כפתור | תווית | שדה | נורית | דפדפן | שורה | משטח
	text      string
	textColor walk.Color
	onClick   object.Object
	label     *walk.Label
	edit      *walk.LineEdit
	ledWidget *walk.CustomWidget
	// דפדפן (WebView2)
	url     string
	html    string
	host    *walk.Composite
	browser *edge.Chromium
	// שורה — ילדים אופקיים
	children []*controlState
	// משטח ציור
	board       *drawBoard
	canvas      *walk.CustomWidget
	canvasW     int
	canvasH     int
	dragging    bool
	onMouseDown object.Object
	onMouseDrag object.Object
	onMouseUp   object.Object
}

func NewWindowsModule() *object.Module {
	m := &object.Module{Name: "חלונות", Attrs: map[string]object.Object{}}
	m.Attrs["חלון"] = &object.Builtin{Fn: winCreateWindow}
	m.Attrs["כפתור"] = &object.Builtin{Fn: winCreateButton}
	m.Attrs["תווית"] = &object.Builtin{Fn: winCreateLabel}
	m.Attrs["שדה"] = &object.Builtin{Fn: winCreateEdit}
	m.Attrs["נורית"] = &object.Builtin{Fn: winCreateLED}
	m.Attrs["דפדפן"] = &object.Builtin{Fn: winCreateBrowser}
	m.Attrs["שורה"] = &object.Builtin{Fn: winCreateRow}
	m.Attrs["משטח"] = &object.Builtin{Fn: winCreateCanvas}
	m.Attrs["הודעה"] = &object.Builtin{Fn: winMessage}
	return m
}

func winCreateWindow(args ...object.Object) object.Object {
	title := "יוד"
	if len(args) >= 1 {
		if s, ok := asString(args[0]); ok {
			title = s
		} else {
			return errObj("חלונות.חלון מצפה לכותרת מחרוזת")
		}
	}
	st := &windowState{title: title, width: 420, height: 320}
	w := &object.GuiWidget{Kind: "חלון", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["הוסף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winAdd(st, a...)
	}}
	w.Attrs["קבע_גודל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetSize(st, a...)
	}}
	w.Attrs["הצג"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winShow(st)
	}}
	return w
}

func winCreateButton(args ...object.Object) object.Object {
	text := "כפתור"
	if len(args) >= 1 {
		if s, ok := asString(args[0]); ok {
			text = s
		} else {
			return errObj("חלונות.כפתור מצפה לטקסט מחרוזת")
		}
	}
	st := &controlState{kind: "כפתור", text: text}
	w := &object.GuiWidget{Kind: "כפתור", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["בלחיצה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("כפתור.בלחיצה מצפה לפונקציה אחת")
		}
		switch a[0].(type) {
		case *object.Function, *object.Closure, *object.CompiledFunction:
			st.onClick = a[0]
			return &object.Null{}
		default:
			return errObj("כפתור.בלחיצה מצפה לפונקציה")
		}
	}}
	w.Attrs["קבע_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlText(st, a...)
	}}
	w.Attrs["קרא_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.String{Value: st.text}
	}}
	return w
}

func winCreateLabel(args ...object.Object) object.Object {
	text := ""
	if len(args) >= 1 {
		if s, ok := asString(args[0]); ok {
			text = s
		} else {
			text = args[0].Inspect()
		}
	}
	st := &controlState{kind: "תווית", text: text, textColor: walk.RGB(30, 30, 30)}
	w := &object.GuiWidget{Kind: "תווית", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlText(st, a...)
	}}
	w.Attrs["קרא_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if st.label != nil {
			return &object.String{Value: st.label.Text()}
		}
		return &object.String{Value: st.text}
	}}
	w.Attrs["קבע_צבע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlColor(st, a...)
	}}
	return w
}

// חלונות.נורית() — לד סטטוס מצויר: ירוק=פעיל, אדום=כבוי
func winCreateLED(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלונות.נורית מצפה ל־0 ארגומנטים")
	}
	st := &controlState{
		kind:      "נורית",
		text:      "כבוי",
		textColor: walk.RGB(239, 68, 68), // אדום
	}
	w := &object.GuiWidget{Kind: "נורית", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["הדלק"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("נורית.הדלק מצפה ל־0 ארגומנטים")
		}
		return setLED(st, true)
	}}
	w.Attrs["כבה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("נורית.כבה מצפה ל־0 ארגומנטים")
		}
		return setLED(st, false)
	}}
	w.Attrs["קבע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("נורית.קבע מצפה לאמת/שקר")
		}
		on := false
		switch v := a[0].(type) {
		case *object.Boolean:
			on = v.Value
		default:
			return errObj("נורית.קבע מצפה לאמת או שקר")
		}
		return setLED(st, on)
	}}
	w.Attrs["קבע_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlText(st, a...)
	}}
	return w
}

func setLED(st *controlState, on bool) object.Object {
	if on {
		st.text = "פעיל"
		st.textColor = walk.RGB(34, 197, 94) // ירוק
	} else {
		st.text = "כבוי"
		st.textColor = walk.RGB(239, 68, 68) // אדום
	}
	if st.ledWidget != nil {
		st.ledWidget.Invalidate()
	}
	return object.Nil
}

func paintLED(st *controlState, canvas *walk.Canvas, _ walk.Rectangle) error {
	if st.ledWidget == nil {
		return nil
	}
	bounds := st.ledWidget.ClientBounds()

	bg, err := walk.NewSolidColorBrush(walk.RGB(240, 240, 240))
	if err != nil {
		return err
	}
	defer bg.Dispose()
	if err := canvas.FillRectangle(bg, bounds); err != nil {
		return err
	}

	pad := 5
	d := bounds.Height - pad*2
	if d > 22 {
		d = 22
	}
	if d < 12 {
		d = 12
	}
	circle := walk.Rectangle{
		X:      bounds.X + pad,
		Y:      bounds.Y + (bounds.Height-d)/2,
		Width:  d,
		Height: d,
	}

	brush, err := walk.NewSolidColorBrush(st.textColor)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	if err := canvas.FillEllipse(brush, circle); err != nil {
		return err
	}

	ring, err := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(60, 60, 60))
	if err == nil {
		defer ring.Dispose()
		_ = canvas.DrawEllipse(ring, circle)
	}

	font, err := walk.NewFont("Segoe UI", 11, walk.FontBold)
	if err != nil {
		font = st.ledWidget.Font()
	} else {
		defer font.Dispose()
	}
	if font == nil {
		return nil
	}
	tr := walk.Rectangle{
		X:      circle.X + circle.Width + 10,
		Y:      bounds.Y,
		Width:  bounds.Width - (circle.X + circle.Width + 14),
		Height: bounds.Height,
	}
	return canvas.DrawText(st.text, font, walk.RGB(40, 40, 40), tr,
		walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
}

func setControlColor(st *controlState, args ...object.Object) object.Object {
	c, err := parseWalkColor(args...)
	if err != nil {
		return errObj(err.Error())
	}
	st.textColor = c
	if st.ledWidget != nil {
		st.ledWidget.Invalidate()
		return object.Nil
	}
	if st.label != nil {
		st.label.SetTextColor(c)
		st.label.Invalidate()
	}
	return object.Nil
}

func parseWalkColor(args ...object.Object) (walk.Color, error) {
	if len(args) == 1 {
		if s, ok := asString(args[0]); ok {
			switch s {
			case "ירוק", "green":
				return walk.RGB(22, 163, 74), nil
			case "אדום", "red":
				return walk.RGB(220, 38, 38), nil
			case "כתום", "orange":
				return walk.RGB(234, 88, 12), nil
			case "כחול", "blue":
				return walk.RGB(37, 99, 235), nil
			case "אפור", "gray", "grey":
				return walk.RGB(100, 116, 139), nil
			case "שחור", "black":
				return walk.RGB(30, 30, 30), nil
			case "לבן", "white":
				return walk.RGB(250, 250, 250), nil
			default:
				return 0, fmt.Errorf("צבע לא מוכר: %s (ירוק/אדום/כתום/כחול/אפור)", s)
			}
		}
		return 0, fmt.Errorf("קבע_צבע מצפה לשם צבע או ל־3 מספרים RGB")
	}
	if len(args) == 3 {
		r, ok1 := args[0].(*object.Number)
		g, ok2 := args[1].(*object.Number)
		b, ok3 := args[2].(*object.Number)
		if !ok1 || !ok2 || !ok3 {
			return 0, fmt.Errorf("קבע_צבע מצפה למספרים RGB")
		}
		return walk.RGB(byte(r.Value), byte(g.Value), byte(b.Value)), nil
	}
	return 0, fmt.Errorf("קבע_צבע מצפה לשם צבע, או ל־3 מספרים RGB")
}

// חלונות.דפדפן(כתובת?) — תצוגת אתר/HTML עם WebView2 בתוך החלון
func winCreateBrowser(args ...object.Object) object.Object {
	st := &controlState{kind: "דפדפן", url: "about:blank"}
	if len(args) >= 1 {
		if s, ok := asString(args[0]); ok {
			st.url = normalizeNavURL(s)
		} else {
			return errObj("חלונות.דפדפן מצפה לכתובת מחרוזת")
		}
	}
	if len(args) > 1 {
		return errObj("חלונות.דפדפן מצפה ל־0 או 1 ארגומנטים")
	}
	w := &object.GuiWidget{Kind: "דפדפן", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["נווט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("דפדפן.נווט מצפה לכתובת אחת")
		}
		s, ok := asString(a[0])
		if !ok {
			return errObj("דפדפן.נווט מצפה למחרוזת")
		}
		browserNavigate(st, s)
		return object.Nil
	}}
	w.Attrs["קבע_html"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("דפדפן.קבע_html מצפה למחרוזת HTML אחת")
		}
		s, ok := asString(a[0])
		if !ok {
			return errObj("דפדפן.קבע_html מצפה למחרוזת")
		}
		browserSetHTML(st, s)
		return object.Nil
	}}
	w.Attrs["רענן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("דפדפן.רענן מצפה ל־0 ארגומנטים")
		}
		browserRefresh(st)
		return object.Nil
	}}
	w.Attrs["קרא_כתובת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("דפדפן.קרא_כתובת מצפה ל־0 ארגומנטים")
		}
		return &object.String{Value: st.url}
	}}
	return w
}

func winCreateEdit(args ...object.Object) object.Object {
	text := ""
	if len(args) >= 1 {
		if s, ok := asString(args[0]); ok {
			text = s
		}
	}
	st := &controlState{kind: "שדה", text: text}
	w := &object.GuiWidget{Kind: "שדה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlText(st, a...)
	}}
	w.Attrs["קרא_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if st.edit != nil {
			return &object.String{Value: st.edit.Text()}
		}
		return &object.String{Value: st.text}
	}}
	return w
}

func setControlText(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("קבע_טקסט מצפה לערך אחד")
	}
	text := args[0].Inspect()
	if s, ok := args[0].(*object.String); ok {
		text = s.Value
	}
	st.text = text
	if st.label != nil {
		_ = st.label.SetText(text)
	}
	if st.edit != nil {
		_ = st.edit.SetText(text)
	}
	if st.ledWidget != nil {
		st.ledWidget.Invalidate()
	}
	return &object.Null{}
}

func winAdd(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חלון.הוסף מצפה לרכיב אחד")
	}
	gw, ok := args[0].(*object.GuiWidget)
	if !ok {
		return errObj("חלון.הוסף מצפה לרכיב ממשק (כפתור/תווית/שדה/דפדפן)")
	}
	cs, ok := gw.Data.(*controlState)
	if !ok {
		return errObj("לא ניתן להוסיף חלון לתוך חלון")
	}
	st.children = append(st.children, cs)
	return &object.Null{}
}

func winSetSize(st *windowState, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלון.קבע_גודל מצפה לרוחב וגובה")
	}
	w, ok1 := args[0].(*object.Number)
	h, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("קבע_גודל מצפה למספרים")
	}
	st.width = int(w.Value)
	st.height = int(h.Value)
	return &object.Null{}
}

func winShow(st *windowState) object.Object {
	var mw *walk.MainWindow
	children := make([]Widget, 0, len(st.children))
	for _, ch := range st.children {
		ch := ch
		children = append(children, buildControlWidget(ch))
	}

	if err := (MainWindow{
		AssignTo:  &mw,
		Title:     st.title,
		MinSize:   Size{Width: st.width, Height: st.height},
		Size:      Size{Width: st.width, Height: st.height},
		Layout:    VBox{},
		Children:  children,
	}).Create(); err != nil {
		return errObj("הצגת חלון נכשלה: " + err.Error())
	}

	for _, ch := range st.children {
		wireBrowsersRecursive(ch, mw)
	}

	mw.SizeChanged().Attach(func() {
		resizeBrowsersRecursive(st.children)
	})

	mw.Starting().Attach(func() {
		for _, ch := range st.children {
			startBrowsersRecursive(ch)
		}
	})

	mw.Run()
	return &object.Null{}
}

func wireBrowsersRecursive(ch *controlState, mw *walk.MainWindow) {
	if ch.kind == "דפדפן" && ch.host != nil {
		br, err := attachWebView2(ch.host, ch.url, ch.html)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		ch.browser = br
	}
	for _, c := range ch.children {
		wireBrowsersRecursive(c, mw)
	}
}

func resizeBrowsersRecursive(children []*controlState) {
	for _, ch := range children {
		if ch.browser != nil {
			ch.browser.Resize()
			_ = ch.browser.NotifyParentWindowPositionChanged()
		}
		resizeBrowsersRecursive(ch.children)
	}
}

func startBrowsersRecursive(ch *controlState) {
	if ch.kind == "דפדפן" {
		browserLoadInitial(ch)
	}
	for _, c := range ch.children {
		startBrowsersRecursive(c)
	}
}

func winMessage(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("חלונות.הודעה מצפה לטקסט, או כותרת וטקסט")
	}
	title := "יוד"
	msg := ""
	if len(args) == 1 {
		msg = args[0].Inspect()
		if s, ok := args[0].(*object.String); ok {
			msg = s.Value
		}
	} else {
		if s, ok := asString(args[0]); ok {
			title = s
		} else {
			title = args[0].Inspect()
		}
		if s, ok := asString(args[1]); ok {
			msg = s
		} else {
			msg = args[1].Inspect()
		}
	}
	walk.MsgBox(nil, title, msg, walk.MsgBoxIconInformation)
	return &object.Null{}
}
