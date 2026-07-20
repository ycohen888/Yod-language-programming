//go:build windows

package stdlib

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"yod/internal/console"
	"yod/internal/gpu"
	"yod/internal/object"
)

type windowState struct {
	title     string
	width     int
	height    int
	fixedSize bool // קבע_גודל_קבוע — בלי גרירת מסגרת / מקסם
	fixedSizeBusy bool
	borderless  bool // קבע_בלי_מסגרת — בלי כותרת Windows
	transparent bool // קבע_שקוף — WebView שקוף + DWM (בלי color-key)
	windowShape string // קבע_צורת_חלון — עיגול / מעוגל / מלבן (SetWindowRgn)
	topMost     bool // קבע_תמיד_מעל
	children  []*controlState
	timers    []windowTimer
	onStart   object.Object
	onClosing object.Object
	mw        *walk.MainWindow
	closed    bool
	forceClose bool
	iconPath  string
	icon      *walk.Icon
	bgColor   walk.Color
	hasBg     bool
	dark      bool
	menuItems []MenuItem
	// deferShow — חלון מוסתר עד שמסך הטעינה של עיצוב מוכן (בלי הבזק כהה/לבן)
	deferShow bool
	revealed  bool
	// מיקום: מרכז או קואורדינטות מפורשות (מסך)
	center bool
	posX   int
	posY   int
	hasPos bool
	// מגש מערכת
	tray            *walk.NotifyIcon
	trayHost        *walk.MainWindow // חלון נסתר — NotifyIcon של walk דורס USERDATA
	trayIcon        *walk.Icon
	trayTip         string
	trayIconPath    string
	trayItems       []trayMenuItem
	trayConfigured  bool
	onTrayMenu      object.Object
	trayLastLeftClick time.Time
}

type trayMenuItem struct {
	text      string
	value     string
	separator bool
}

type windowTimer struct {
	seconds float64
	fn      object.Object
}

type controlState struct {
	kind      string // כפתור | תווית | שדה | נורית | דפדפן | שורה | עמודה | מסגרת | משטח | דגם | סמל | רשימה | טבלה
	text      string
	textColor walk.Color
	onClick   object.Object
	label     *walk.Label
	edit      *walk.LineEdit
	ledWidget *walk.CustomWidget
	// דפדפן (WebView2)
	url            string
	html           string
	host           *walk.Composite
	browser        *edge.Chromium
	onBrowserMsg   object.Object // בהודעה(טקסט) — מ־JS דרך postMessage
	onBrowserReady object.Object // מוכן() — אחרי NavigationCompleted
	browserReady   bool
	parentWin      *windowState // לחשיפת חלון מושהית אחרי splash
	// וידאו (WebView2 + HTML5)
	videoPath         string
	videoLoop         bool
	videoVol          int
	videoOverlays     []textOverlay
	videoShowOverlays bool
	videoSelected     int // אינדקס שכבת טקסט נבחרת (-1 = אין)
	// שורה / עמודה / מסגרת
	children  []*controlState
	frameDir  string // אופקי | אנכי (למסגרת)
	panel  *walk.Composite // מסגרת — AssignTo + קבע_נראה
	hidden bool            // קבע_נראה:שקר
	// משטח ציור
	board       *drawBoard
	canvas      *walk.CustomWidget
	canvasW     int // גודל לוגי (DIP) שנחשף ליוד
	canvasH     int
	canvasDipW  int // גודל מקורי/נעול ב־DIP (ל־MinSize/MaxSize)
	canvasDipH  int
	dragging    bool
	dragButton  string
	onMouseDown  object.Object
	onMouseDrag  object.Object
	onMouseUp    object.Object
	onMouseHover object.Object // תנועת עכבר בלי לחיצה (רמזים וכו')
	onMouseWheel object.Object // בעכבר_גלגל(דלתא) — גלגל עכבר
	onKeyChar    object.Object // בעת_תו — תו מוקלד (כולל עברית)
	onKeyCmd     object.Object // בעת_מקש — מחיקה / אנטר / …
	onSizeChange object.Object // בשינוי_גודל(רוחב, גובה)
	stretchFactor int          // -1 = ברירת מחדל לפי סוג רכיב; מסגרת/טבלה/גרף
	canvasLockH   bool         // משטח: נעילת גובה (סרגלים) — רוחב גמיש
	canvasLockW   bool         // משטח: נעילת רוחב (סרגל צד) — גובה גמיש
	canvasKeysWired bool
	wheelWired      bool
	canvasBatch     bool // אחרי נקה — בלי Invalidate עד רענן
	canvasDirty     bool
	canvasKeyKeep   uintptr // מונע GC מ־NewCallback לחצי מקלדת
	sizeWired     bool
	sizeBusy      bool
	undoStack   []*image.RGBA
	backup      *image.RGBA
	// דגם צבע / סמל כלי
	swatchColor color.RGBA
	iconKind    string
	toolWidget  *walk.CustomWidget
	// רשימה
	listBox     *walk.ListBox
	listItems   []string
	listMinH    int
	onSelect    object.Object
	// טבלה
	tableView   *walk.TableView
	tableModel  *yodTableModel
	tableCols   []int
	tableDark   bool
	tableAllRows []yodTableRow // מקור מלא לסינון
	tableFilter string
	// עיצוב כהה + מצב רכיב
	ctrlDark     bool
	ctrlDisabled bool
	ctrlHint     string
	passwordMode bool // שדה: הצגת • במקום אותיות
	button       *walk.PushButton
	// גרף
	chartWidget     *walk.CustomWidget
	chartKind       string // עמודות | קו | עוגה
	chartTitle      string
	chartSubtitle   string
	chartLabels     []string
	chartSeries     []chartSeries
	chartXLabel     string
	chartYLabel     string
	chartShowLegend bool
	chartShowGrid   bool
	chartDark       bool
	chartYMin       float64
	chartYMax       float64
	chartYRangeSet  bool
	chartStacked      bool
	chartValueFormat  string // "" | "בתים"
	chartLegendPos    string // ""|"צד" | "מעל" | "תחת"
	// רקע אופציונלי למסגרת
	bgColor walk.Color
	hasBg   bool
	// משטח_GPU (OpenGL מקומי)
	gpuSurface *gpu.Surface
}

func NewWindowsModule() *object.Module {
	m := &object.Module{Name: "חלונות", Attrs: map[string]object.Object{}}
	m.Attrs["חלון"] = &object.Builtin{Fn: winCreateWindow}
	m.Attrs["גודל_מסך"] = &object.Builtin{Fn: winScreenSize}
	m.Attrs["כפתור"] = &object.Builtin{Fn: winCreateButton}
	m.Attrs["תווית"] = &object.Builtin{Fn: winCreateLabel}
	m.Attrs["שדה"] = &object.Builtin{Fn: winCreateEdit}
	m.Attrs["נורית"] = &object.Builtin{Fn: winCreateLED}
	m.Attrs["דפדפן"] = &object.Builtin{Fn: winCreateBrowser}
	m.Attrs["שורה"] = &object.Builtin{Fn: winCreateRow}
	m.Attrs["עמודה"] = &object.Builtin{Fn: winCreateColumn}
	m.Attrs["מסגרת"] = &object.Builtin{Fn: winCreateFrame}
	m.Attrs["משטח"] = &object.Builtin{Fn: winCreateCanvas}
	m.Attrs["משטח_GPU"] = &object.Builtin{Fn: winCreateGPUSurface}
	m.Attrs["דגם"] = &object.Builtin{Fn: winCreateSwatch}
	m.Attrs["סמל"] = &object.Builtin{Fn: winCreateIcon}
	m.Attrs["הודעה"] = &object.Builtin{Fn: winMessage}
	m.Attrs["בחר_שמירה"] = &object.Builtin{Fn: winFileSave}
	m.Attrs["בחר_פתיחה"] = &object.Builtin{Fn: winFileOpen}
	m.Attrs["בחר_תיקייה"] = &object.Builtin{Fn: winBrowseFolder}
	m.Attrs["רשימה"] = &object.Builtin{Fn: winCreateList}
	m.Attrs["טבלה"] = &object.Builtin{Fn: winCreateTable}
	// קיצור תאימות — אותו רכיב כמו גרפים.גרף (מומלץ: כלול "גרפים")
	m.Attrs["גרף"] = &object.Builtin{Fn: winCreateChart}
	m.Attrs["שאל"] = &object.Builtin{Fn: winAsk}
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
	w.Attrs["קבע_גודל_קבוע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetFixedSize(st, a...)
	}}
	w.Attrs["קבע_בלי_מסגרת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetBorderless(st, a...)
	}}
	w.Attrs["קבע_שקוף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetTransparent(st, a...)
	}}
	w.Attrs["קבע_צורת_חלון"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetWindowShape(st, a...)
	}}
	w.Attrs["קבע_תמיד_מעל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetTopMost(st, a...)
	}}
	w.Attrs["הזז"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winMoveBy(st, a...)
	}}
	w.Attrs["התחל_גרירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winStartDrag(st, a...)
	}}
	w.Attrs["קבע_מיקום"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetPosition(st, a...)
	}}
	w.Attrs["מרכז"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winCenter(st, a...)
	}}
	w.Attrs["כל_כמה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winEvery(st, a...)
	}}
	w.Attrs["בהתחלה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("חלון.בהתחלה מצפה לפונקציה")
		}
		st.onStart = a[0]
		return object.Nil
	}}
	w.Attrs["קבע_איקון"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetIcon(st, a...)
	}}
	w.Attrs["קבע_רקע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetBackground(st, a...)
	}}
	w.Attrs["קבע_כהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetDark(st, a...)
	}}
	w.Attrs["קבע_תפריט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winSetMenu(st, a...)
	}}
	w.Attrs["הצג"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winShow(st)
	}}
	w.Attrs["הסתר"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winHide(st, a...)
	}}
	w.Attrs["הצג_שוב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winShowAgain(st, a...)
	}}
	w.Attrs["בסגירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winOnClosing(st, a...)
	}}
	w.Attrs["סגור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winForceClose(st, a...)
	}}
	w.Attrs["מגש"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winTray(st, a...)
	}}
	w.Attrs["במגש"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winOnTrayMenu(st, a...)
	}}
	w.Attrs["מגש_התראה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winTrayNotify(st, a...)
	}}
	w.Attrs["הסר_מגש"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return winRemoveTray(st, a...)
	}}
	return w
}

func winSetDark(st *windowState, args ...object.Object) object.Object {
	on, errObjVal := parseDarkBool("חלון.קבע_כהה", args...)
	if errObjVal != nil {
		return errObjVal
	}
	st.dark = on
	if on && !st.hasBg {
		st.bgColor = darkWinBG()
		st.hasBg = true
	}
	if on {
		markDarkControlsRecursive(st.children)
	}
	if st.mw != nil {
		if on {
			preferAppDarkMode()
			applyDarkThemeToWidget(st.mw)
			if st.hasBg {
				brush, err := walk.NewSolidColorBrush(st.bgColor)
				if err == nil {
					st.mw.SetBackground(brush)
				}
			}
			applyDarkControlsRecursive(st.children, true)
			st.mw.Invalidate()
		}
	}
	return object.Nil
}

func markDarkControlsRecursive(children []*controlState) {
	for _, ch := range children {
		if ch == nil {
			continue
		}
		switch ch.kind {
		case "טבלה":
			ch.tableDark = true
		case "גרף":
			ch.chartDark = true
		case "כפתור", "שדה", "רשימה":
			ch.ctrlDark = true
		case "תווית":
			if ch.textColor == walk.RGB(30, 30, 30) {
				ch.textColor = darkCtlText()
			}
		}
		markDarkControlsRecursive(ch.children)
	}
}

func winSetBackground(st *windowState, args ...object.Object) object.Object {
	c, err := parseWalkColor(args...)
	if err != nil {
		return errObj("חלון.קבע_רקע: " + err.Error())
	}
	st.bgColor = c
	st.hasBg = true
	if st.mw != nil {
		brush, err := walk.NewSolidColorBrush(c)
		if err != nil {
			return errObj("חלון.קבע_רקע נכשל: " + err.Error())
		}
		st.mw.SetBackground(brush)
		st.mw.Invalidate()
	}
	return object.Nil
}

func winSetIcon(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חלון.קבע_איקון מצפה לנתיב קובץ")
	}
	s, ok := asString(args[0])
	if !ok || strings.TrimSpace(s) == "" {
		return errObj("חלון.קבע_איקון מצפה למחרוזת נתיב")
	}
	path := s
	if !filepath.IsAbs(path) {
		base := AppBaseDir()
		if base == "" {
			base, _ = os.Getwd()
		}
		path = filepath.Join(base, path)
	}
	if _, err := os.Stat(path); err != nil {
		return errObj("חלון.קבע_איקון: הקובץ לא נמצא: " + path)
	}
	st.iconPath = path
	if st.mw != nil {
		if err := applyWindowIcon(st, path); err != nil {
			return errObj("חלון.קבע_איקון נכשל: " + err.Error())
		}
	}
	return object.Nil
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
	w.Attrs["קבע_כהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		on, errV := parseDarkBool("כפתור.קבע_כהה", a...)
		if errV != nil {
			return errV
		}
		st.ctrlDark = on
		if st.button != nil {
			applyButtonDark(st)
		}
		return object.Nil
	}}
	w.Attrs["קבע_מושבת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlDisabled(st, "כפתור.קבע_מושבת", a...)
	}}
	w.Attrs["קבע_רמז"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlHint(st, "כפתור.קבע_רמז", a...)
	}}
	attachVisible(w, st, "כפתור")
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
	attachVisible(w, st, "תווית")
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
			case "צהוב", "yellow":
				return walk.RGB(234, 179, 8), nil
			case "סגול", "purple", "violet":
				return walk.RGB(168, 85, 247), nil
			case "ורוד", "pink":
				return walk.RGB(236, 72, 153), nil
			case "תכלת", "cyan", "sky":
				return walk.RGB(14, 165, 233), nil
			case "זהב", "gold":
				return walk.RGB(202, 138, 4), nil
			default:
				return 0, fmt.Errorf("צבע לא מוכר: %s", s)
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
	st := &controlState{kind: "דפדפן", url: "about:blank", stretchFactor: -1}
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
	w.Attrs["הרץ_js"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("דפדפן.הרץ_js מצפה למחרוזת JavaScript אחת")
		}
		s, ok := asString(a[0])
		if !ok {
			return errObj("דפדפן.הרץ_js מצפה למחרוזת")
		}
		return browserEval(st, s)
	}}
	w.Attrs["שלח"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("דפדפן.שלח מצפה למחרוזת אחת (טקסט או JSON)")
		}
		s, ok := asString(a[0])
		if !ok {
			return errObj("דפדפן.שלח מצפה למחרוזת")
		}
		return browserSend(st, s)
	}}
	w.Attrs["בהודעה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("דפדפן.בהודעה מצפה לפונקציה אחת")
		}
		if !isCallable(a[0]) {
			return errObj("דפדפן.בהודעה מצפה לפונקציה")
		}
		st.onBrowserMsg = a[0]
		return object.Nil
	}}
	w.Attrs["מוכן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("דפדפן.מוכן מצפה לפונקציה אחת")
		}
		if !isCallable(a[0]) {
			return errObj("דפדפן.מוכן מצפה לפונקציה")
		}
		st.onBrowserReady = a[0]
		if st.browserReady {
			runOnUI(func() { invokeYod(st.onBrowserReady, nil) })
		}
		return object.Nil
	}}
	w.Attrs["קבע_מתיחה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setStretchFactor(st, "דפדפן.קבע_מתיחה", a...)
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
	w.Attrs["קבע_כהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		on, errV := parseDarkBool("שדה.קבע_כהה", a...)
		if errV != nil {
			return errV
		}
		st.ctrlDark = on
		if st.edit != nil {
			applyEditDark(st)
		}
		return object.Nil
	}}
	w.Attrs["קבע_מושבת"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlDisabled(st, "שדה.קבע_מושבת", a...)
	}}
	w.Attrs["קבע_רמז"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setControlHint(st, "שדה.קבע_רמז", a...)
	}}
	w.Attrs["קבע_סיסמה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		on, errV := parseDarkBool("שדה.קבע_סיסמה", a...)
		if errV != nil {
			return errV
		}
		st.passwordMode = on
		if st.edit != nil {
			st.edit.SetPasswordMode(on)
		}
		return object.Nil
	}}
	attachVisible(w, st, "שדה")
	return w
}

func attachVisible(w *object.GuiWidget, st *controlState, kind string) {
	w.Attrs["קבע_נראה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj(kind + ".קבע_נראה מצפה לערך בוליאני")
		}
		on, errV := parseDarkBool(kind+".קבע_נראה", a...)
		if errV != nil {
			return errV
		}
		st.hidden = !on
		applyControlVisible(st)
		return object.Nil
	}}
}

func applyControlVisible(st *controlState) {
	if st == nil {
		return
	}
	vis := !st.hidden
	if st.button != nil {
		st.button.SetVisible(vis)
	}
	if st.label != nil {
		st.label.SetVisible(vis)
	}
	if st.edit != nil {
		st.edit.SetVisible(vis)
	}
	if st.canvas != nil {
		st.canvas.SetVisible(vis)
	}
	if st.panel != nil {
		st.panel.SetVisible(vis)
	}
	if st.host != nil {
		st.host.SetVisible(vis)
	}
}

func setControlDisabled(st *controlState, name string, a ...object.Object) object.Object {
	on, errV := parseDarkBool(name, a...)
	if errV != nil {
		return errV
	}
	st.ctrlDisabled = on
	if st.button != nil {
		st.button.SetEnabled(!on)
	}
	if st.edit != nil {
		st.edit.SetEnabled(!on)
	}
	return object.Nil
}

func setControlHint(st *controlState, name string, a ...object.Object) object.Object {
	if len(a) != 1 {
		return errObj(name + " מצפה למחרוזת")
	}
	s, ok := asString(a[0])
	if !ok {
		return errObj(name + " מצפה למחרוזת")
	}
	st.ctrlHint = s
	if st.button != nil {
		_ = st.button.SetToolTipText(s)
	}
	if st.edit != nil {
		_ = st.edit.SetToolTipText(s)
	}
	if st.canvas != nil {
		_ = st.canvas.SetToolTipText(s)
	}
	return object.Nil
}

func setSurfaceHint(st *controlState, a ...object.Object) object.Object {
	return setControlHint(st, "משטח.קבע_רמז", a...)
}

func winSetMenu(st *windowState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חלון.קבע_תפריט מצפה לרשימת תפריטים")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("חלון.קבע_תפריט מצפה לרשימה")
	}
	items, err := buildMenuItems(arr)
	if err != nil {
		return errObj("חלון.קבע_תפריט: " + err.Error())
	}
	st.menuItems = items
	return object.Nil
}

func buildMenuItems(arr *object.Array) ([]MenuItem, error) {
	out := make([]MenuItem, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		h, ok := el.(*object.Hash)
		if !ok {
			return nil, fmt.Errorf("כל פריט תפריט חייב להיות מילון")
		}
		nameObj, ok := h.Pairs["שם"]
		if !ok {
			return nil, fmt.Errorf("חסר מפתח \"שם\"")
		}
		name, ok := asString(nameObj)
		if !ok {
			return nil, fmt.Errorf("שם תפריט חייב מחרוזת")
		}
		if itemsObj, ok := h.Pairs["פריטים"]; ok {
			subArr, ok := itemsObj.(*object.Array)
			if !ok {
				return nil, fmt.Errorf("פריטים חייבים רשימה")
			}
			sub, err := buildMenuItems(subArr)
			if err != nil {
				return nil, err
			}
			out = append(out, Menu{Text: name, Items: sub})
			continue
		}
		if actionObj, ok := h.Pairs["פעולה"]; ok {
			if !isCallable(actionObj) {
				return nil, fmt.Errorf("פעולה חייבת פונקציה")
			}
			fn := actionObj
			out = append(out, Action{
				Text: name,
				OnTriggered: func() {
					invokeYod(fn, nil)
				},
			})
			continue
		}
		if name == "-" || name == "מפריד" {
			out = append(out, Separator{})
			continue
		}
		return nil, fmt.Errorf("פריט %q חייב פעולה או פריטים", name)
	}
	return out, nil
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
	if st.button != nil {
		_ = st.button.SetText(text)
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
	linkParentWindow(cs, st)
	return &object.Null{}
}

func linkParentWindow(ch *controlState, win *windowState) {
	if ch == nil {
		return
	}
	ch.parentWin = win
	for _, c := range ch.children {
		linkParentWindow(c, win)
	}
}

func shouldDeferWindowShow(st *windowState) bool {
	if st == nil {
		return false
	}
	var walkKids func([]*controlState) bool
	walkKids = func(kids []*controlState) bool {
		for _, ch := range kids {
			if ch == nil {
				continue
			}
			if (ch.kind == "דפדפן" || ch.kind == "וידאו") && htmlLooksLikeDesignSplash(ch.html) {
				return true
			}
			if walkKids(ch.children) {
				return true
			}
		}
		return false
	}
	return walkKids(st.children)
}

func revealDeferredMainWindow(st *windowState) {
	revealSplashWindow(st)
}

func releaseDeferredDesignReady(st *windowState) {
	if st == nil {
		return
	}
	var walkKids func([]*controlState)
	walkKids = func(kids []*controlState) {
		for _, ch := range kids {
			if ch == nil {
				continue
			}
			if ch.browser != nil {
				ch.browser.Eval(`(function(){
  window.__יוד_המתן_לחשיפה=false;
  if(typeof window.__יוד_שחרר_מוכן==="function"){
    var f=window.__יוד_שחרר_מוכן;
    window.__יוד_שחרר_מוכן=null;
    try{f();}catch(e){}
  }
})();`)
			}
			walkKids(ch.children)
		}
	}
	walkKids(st.children)
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
	if st.width < 1 {
		st.width = 1
	}
	if st.height < 1 {
		st.height = 1
	}
	if st.mw != nil {
		applyWindowPlacement(st)
		applyFixedWindowSize(st)
		applyWindowShape(st)
	}
	return &object.Null{}
}

func winSetFixedSize(st *windowState, args ...object.Object) object.Object {
	on, errObjVal := parseDarkBool("חלון.קבע_גודל_קבוע", args...)
	if errObjVal != nil {
		return errObjVal
	}
	st.fixedSize = on
	if st.mw != nil {
		applyFixedWindowSize(st)
	}
	return object.Nil
}

func winSetBorderless(st *windowState, args ...object.Object) object.Object {
	on, errObjVal := parseDarkBool("חלון.קבע_בלי_מסגרת", args...)
	if errObjVal != nil {
		return errObjVal
	}
	st.borderless = on
	if st.mw != nil {
		applyBorderlessWindow(st)
	}
	return object.Nil
}

func winSetTransparent(st *windowState, args ...object.Object) object.Object {
	on, errObjVal := parseDarkBool("חלון.קבע_שקוף", args...)
	if errObjVal != nil {
		return errObjVal
	}
	st.transparent = on
	if on {
		// רקע כהה מאחורי WebView A=0 — בלי Color Key (שובר לחיצות/גרירה)
		st.bgColor = walk.RGB(16, 26, 43)
		st.hasBg = true
	}
	if st.mw != nil {
		if on {
			applyTransparentWindow(st)
		} else {
			clearWindowLayered(st.mw.Handle())
			resetFrameIntoClient(st.mw.Handle())
		}
		for _, ch := range st.children {
			applyBrowserTransparency(ch, on)
		}
	}
	return object.Nil
}

func winSetTopMost(st *windowState, args ...object.Object) object.Object {
	on, errObjVal := parseDarkBool("חלון.קבע_תמיד_מעל", args...)
	if errObjVal != nil {
		return errObjVal
	}
	st.topMost = on
	if st.mw != nil {
		applyTopMostWindow(st)
	}
	return object.Nil
}

func winMoveBy(st *windowState, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלון.הזז מצפה ל־dx ו־dy")
	}
	dx, ok1 := args[0].(*object.Number)
	dy, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("חלון.הזז מצפה למספרים")
	}
	if st.mw == nil {
		return object.Nil
	}
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return object.Nil
	}
	var r win.RECT
	if !win.GetWindowRect(hwnd, &r) {
		return object.Nil
	}
	nx := r.Left + int32(dx.Value)
	ny := r.Top + int32(dy.Value)
	win.SetWindowPos(hwnd, 0, nx, ny, 0, 0,
		win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE)
	st.hasPos = true
	st.center = false
	st.posX = int(nx)
	st.posY = int(ny)
	return object.Nil
}

// winStartDrag — גרירת חלון בלי מסגרת כמו כותרת Windows (אמין עם WebView).
func winStartDrag(st *windowState, _ ...object.Object) object.Object {
	if st == nil || st.mw == nil {
		return object.Nil
	}
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return object.Nil
	}
	win.ReleaseCapture()
	win.SendMessage(hwnd, win.WM_NCLBUTTONDOWN, uintptr(win.HTCAPTION), 0)
	return object.Nil
}

func applyBorderlessWindow(st *windowState) {
	if st == nil || st.mw == nil || !st.borderless {
		return
	}
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return
	}
	// WS_POPUP — בלי כותרת/מסגרת אמיתית (מונע שברי מסגרת לבנים)
	style := win.GetWindowLong(hwnd, win.GWL_STYLE)
	visible := style&win.WS_VISIBLE != 0
	var popup uint32 = 0x80000000 // WS_POPUP
	style = int32(popup | uint32(win.WS_CLIPCHILDREN) | uint32(win.WS_CLIPSIBLINGS))
	if visible {
		style |= win.WS_VISIBLE
	}
	win.SetWindowLong(hwnd, win.GWL_STYLE, style)

	ex := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	ex |= win.WS_EX_TOOLWINDOW
	ex &^= win.WS_EX_APPWINDOW | win.WS_EX_CLIENTEDGE | win.WS_EX_WINDOWEDGE | win.WS_EX_DLGMODALFRAME | win.WS_EX_STATICEDGE
	if !st.transparent {
		ex &^= win.WS_EX_LAYERED
	}
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, ex)

	disableDwmChromeArtifacts(hwnd)
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
	applyWindowPlacement(st)
	if st.transparent {
		applyTransparentWindow(st)
	}
}

func transparentWindowMargins(st *windowState) Margins {
	if st != nil && (st.transparent || st.borderless) {
		return Margins{}
	}
	return Margins{Left: 2, Top: 0, Right: 2, Bottom: 0}
}

// widgetPanelBG — רקע כהה לווידג׳טים (תואם CSS #101a2b). בלי לבן מאחורי WebView.
var widgetPanelBG = walk.RGB(16, 26, 43)

func applyTransparentWindow(st *windowState) {
	if st == nil || st.mw == nil || !st.transparent {
		return
	}
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return
	}
	// חשוב: בלי LWA_COLORKEY — עם WebView2 זה שובר לחיצות וגרירה.
	// WebView A=0 + רקע כהה לחלון/מארח + קבע_צורת_חלון = בלי מסגרת לבנה.
	clearWindowLayered(hwnd)
	disableDwmChromeArtifacts(hwnd)
	resetFrameIntoClient(hwnd)
	st.bgColor = widgetPanelBG
	st.hasBg = true
	if brush, err := walk.NewSolidColorBrush(widgetPanelBG); err == nil {
		st.mw.SetBackground(brush)
	}
	paintTransparentHosts(st.children, widgetPanelBG)
	stripHwndChrome(hwnd)
	disableDwmChromeArtifacts(hwnd)
}

func paintTransparentHosts(children []*controlState, key walk.Color) {
	for _, ch := range children {
		if ch == nil {
			continue
		}
		if ch.host != nil {
			if brush, err := walk.NewSolidColorBrush(key); err == nil {
				ch.host.SetBackground(brush)
			}
		}
		paintTransparentHosts(ch.children, key)
	}
}

func disposeWindowBrowsers(st *windowState) {
	if st == nil {
		return
	}
	var walkBrowsers func([]*controlState)
	walkBrowsers = func(children []*controlState) {
		for _, ch := range children {
			if ch == nil {
				continue
			}
			if ch.browser != nil {
				ch.browser.Close()
				ch.browser = nil
			}
			walkBrowsers(ch.children)
		}
	}
	walkBrowsers(st.children)
}

func applyTopMostWindow(st *windowState) {
	if st == nil || st.mw == nil {
		return
	}
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return
	}
	insertAfter := win.HWND_NOTOPMOST
	if st.topMost {
		insertAfter = win.HWND_TOPMOST
	}
	win.SetWindowPos(hwnd, insertAfter, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)
}

func applyBrowserTransparency(ch *controlState, on bool) {
	if ch == nil {
		return
	}
	if ch.kind == "דפדפן" && ch.browser != nil {
		if ch.host != nil {
			stripHwndChromeTree(ch.host.Handle())
		}
		if on {
			_ = ch.browser.SetDefaultBackgroundColor(0, 0, 0, 0)
			if settings, err := ch.browser.GetSettings(); err == nil && settings != nil {
				_ = settings.PutAreDefaultContextMenusEnabled(false)
				_ = settings.PutIsStatusBarEnabled(false)
				_ = settings.PutIsZoomControlEnabled(false)
			}
			if ch.host != nil {
				if brush, err := walk.NewSolidColorBrush(widgetPanelBG); err == nil {
					ch.host.AsWindowBase().SetBackground(brush)
				}
			}
		} else {
			// רקע WebView כהה תואם לחלון — מונע פינות/מסגרת לבנות
			r, g, b := byte(14), byte(17), byte(22)
			if ch.parentWin != nil && ch.parentWin.hasBg {
				c := uint32(ch.parentWin.bgColor)
				r = byte(c & 0xff)
				g = byte((c >> 8) & 0xff)
				b = byte((c >> 16) & 0xff)
			}
			_ = ch.browser.SetDefaultBackgroundColor(255, r, g, b)
		}
	}
	for _, kid := range ch.children {
		applyBrowserTransparency(kid, on)
	}
}

func applyFixedWindowSize(st *windowState) {
	if st == nil || st.mw == nil || !st.fixedSize {
		return
	}
	w, h := st.width, st.height
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	sz := walk.Size{Width: w, Height: h}
	_ = st.mw.SetMinMaxSize(sz, sz)
	_ = st.mw.SetSize(sz)

	// walk מיישם רק PtMinTrackSize ב־WM_GETMINMAXINFO — בלי PtMaxTrackSize.
	// לכן חייבים להסיר WS_THICKFRAME כדי למנוע שינוי גודל בפועל.
	hwnd := st.mw.Handle()
	if hwnd == 0 {
		return
	}
	style := win.GetWindowLong(hwnd, win.GWL_STYLE)
	newStyle := style &^ (win.WS_THICKFRAME | win.WS_MAXIMIZEBOX)
	if newStyle != style {
		win.SetWindowLong(hwnd, win.GWL_STYLE, newStyle)
		win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
			win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
	}
	// רענון כותרת אחרי שינוי מסגרת — מונע היעלמות כפתורי סגירה/מזעור
	applyWindowTitleDir(st.mw)
	if st.dark {
		applyDarkChrome(hwnd)
	}
	if st.transparent {
		applyTransparentWindow(st)
	} else if st.borderless {
		disableDwmChromeArtifacts(hwnd)
	}
	win.RedrawWindow(hwnd, nil, 0, win.RDW_FRAME|win.RDW_INVALIDATE|win.RDW_UPDATENOW)
}

func enforceFixedWindowSize(st *windowState) {
	if st == nil || st.mw == nil || !st.fixedSize || st.fixedSizeBusy {
		return
	}
	cur := st.mw.Size()
	if cur.Width == st.width && cur.Height == st.height {
		return
	}
	st.fixedSizeBusy = true
	_ = st.mw.SetSize(walk.Size{Width: st.width, Height: st.height})
	st.fixedSizeBusy = false
}

func winSetPosition(st *windowState, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלון.קבע_מיקום מצפה ל־x ו־y")
	}
	x, ok1 := args[0].(*object.Number)
	y, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return errObj("קבע_מיקום מצפה למספרים")
	}
	st.posX = int(x.Value)
	st.posY = int(y.Value)
	st.hasPos = true
	st.center = false
	if st.mw != nil {
		applyWindowPlacement(st)
	}
	return object.Nil
}

func winCenter(st *windowState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלון.מרכז לא מצפה לארגומנטים")
	}
	st.center = true
	st.hasPos = false
	if st.mw != nil {
		applyWindowPlacement(st)
	}
	return object.Nil
}

func winScreenSize(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חלונות.גודל_מסך לא מצפה לארגומנטים")
	}
	_, _, w, h := screenWorkArea()
	out := object.NewHash()
	out.Set("רוחב", &object.Number{Value: float64(w)})
	out.Set("גובה", &object.Number{Value: float64(h)})
	return out
}

const spiGetWorkArea = 0x0030

func screenWorkArea() (x, y, w, h int) {
	var r win.RECT
	if win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&r), 0) {
		return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top)
	}
	// נפילה: מסך ראשי מלא
	cw, _, _ := procGetSystemMetricsScreen.Call(0) // SM_CXSCREEN
	ch, _, _ := procGetSystemMetricsScreen.Call(1) // SM_CYSCREEN
	return 0, 0, int(cw), int(ch)
}

var procGetSystemMetricsScreen = syscall.NewLazyDLL("user32.dll").NewProc("GetSystemMetrics")

func applyWindowPlacement(st *windowState) {
	if st == nil || st.mw == nil {
		return
	}
	w, h := st.width, st.height
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if st.center {
		ax, ay, aw, ah := screenWorkArea()
		x := ax + (aw-w)/2
		y := ay + (ah-h)/2
		_ = st.mw.SetBounds(walk.Rectangle{X: x, Y: y, Width: w, Height: h})
		return
	}
	if st.hasPos {
		_ = st.mw.SetBounds(walk.Rectangle{X: st.posX, Y: st.posY, Width: w, Height: h})
	}
}

func winEvery(st *windowState, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("חלון.כל_כמה מצפה לשניות ולפונקציה")
	}
	sec, ok := args[0].(*object.Number)
	if !ok || sec.Value <= 0 {
		return errObj("חלון.כל_כמה מצפה למספר שניות חיובי")
	}
	if !isCallable(args[1]) {
		return errObj("חלון.כל_כמה מצפה לפונקציה")
	}
	st.timers = append(st.timers, windowTimer{seconds: sec.Value, fn: args[1]})
	return object.Nil
}

// חלונות.רשימה() — רשימת בחירה (תהליכים, פריטים וכו')
func winCreateList(args ...object.Object) object.Object {
	if len(args) > 0 {
		return errObj("חלונות.רשימה מצפה ל־0 ארגומנטים")
	}
	st := &controlState{kind: "רשימה", listMinH: 260, listItems: []string{}}
	w := &object.GuiWidget{Kind: "רשימה", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_פריטים"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return listSetItems(st, a...)
	}}
	w.Attrs["קרא_אינדקס"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("רשימה.קרא_אינדקס מצפה ל־0 ארגומנטים")
		}
		if st.listBox == nil {
			return &object.Number{Value: -1}
		}
		return &object.Number{Value: float64(st.listBox.CurrentIndex())}
	}}
	w.Attrs["קרא_פריט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("רשימה.קרא_פריט מצפה ל־0 ארגומנטים")
		}
		idx := -1
		if st.listBox != nil {
			idx = st.listBox.CurrentIndex()
		}
		if idx < 0 || idx >= len(st.listItems) {
			return &object.String{Value: ""}
		}
		return &object.String{Value: st.listItems[idx]}
	}}
	w.Attrs["קבע_גובה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("רשימה.קבע_גובה מצפה למספר")
		}
		n, ok := a[0].(*object.Number)
		if !ok || n.Value < 40 {
			return errObj("רשימה.קבע_גובה מצפה למספר >= 40")
		}
		st.listMinH = int(n.Value)
		return object.Nil
	}}
	w.Attrs["בבחירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("רשימה.בבחירה מצפה לפונקציה")
		}
		st.onSelect = a[0]
		return object.Nil
	}}
	w.Attrs["קבע_כהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		on, errV := parseDarkBool("רשימה.קבע_כהה", a...)
		if errV != nil {
			return errV
		}
		st.ctrlDark = on
		if st.listBox != nil {
			applyListDark(st)
		}
		return object.Nil
	}}
	return w
}

func listSetItems(st *controlState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רשימה.קבע_פריטים מצפה לרשימה")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return errObj("רשימה.קבע_פריטים מצפה לרשימה")
	}
	items := make([]string, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		if s, ok := el.(*object.String); ok {
			items = append(items, s.Value)
		} else {
			items = append(items, el.Inspect())
		}
	}
	st.listItems = items
	if st.listBox != nil {
		cp := append([]string(nil), items...)
		if err := st.listBox.SetModel(cp); err != nil {
			return errObj("רשימה.קבע_פריטים נכשל: " + err.Error())
		}
		_ = st.listBox.SetCurrentIndex(-1)
	}
	return object.Nil
}

// חלונות.שאל(כותרת, טקסט) — כן/לא, מחזיר אמת/שקר
func winAsk(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("חלונות.שאל מצפה לטקסט, או כותרת וטקסט")
	}
	title := "יוד"
	msg := ""
	if len(args) == 1 {
		if s, ok := asString(args[0]); ok {
			msg = s
		} else {
			msg = args[0].Inspect()
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
	res := walk.MsgBox(nil, I18nText(title), I18nText(msg), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion|msgBoxDirStyle())
	return &object.Boolean{Value: res == walk.DlgCmdYes}
}

func winShow(st *windowState) object.Object {
	// חלון GUI — מסתירים CMD נפרד שנפתח עם yod.exe (לא טרמינל קיים)
	console.HideIfOwned()

	var mw *walk.MainWindow
	if st.dark {
		preferAppDarkMode()
		markDarkControlsRecursive(st.children)
	}
	children := make([]Widget, 0, len(st.children))
	for _, ch := range st.children {
		ch := ch
		children = append(children, buildControlWidget(ch))
	}

	iconPath := st.iconPath
	if iconPath == "" {
		iconPath = FindAppIconPath()
		st.iconPath = iconPath
	}
	var winIcon *walk.Icon
	if iconPath != "" {
		if ic, err := walk.NewIconFromFile(iconPath); err == nil {
			winIcon = ic
			st.icon = ic
		}
	}

	minW, minH := min(st.width, 720), min(st.height, 480)
	maxW, maxH := 0, 0
	if st.fixedSize {
		minW, minH = st.width, st.height
		maxW, maxH = st.width, st.height
	}
	cfg := MainWindow{
		AssignTo:           &mw,
		Title:              st.title,
		// MinSize קטן מ־Size — אחרת חלון גדול עם תוכן גבוה ננעל ולא נכנס למסך
		MinSize:            Size{Width: minW, Height: minH},
		MaxSize:            Size{Width: maxW, Height: maxH},
		Size:               Size{Width: st.width, Height: st.height},
		Layout:             VBox{Margins: transparentWindowMargins(st), Spacing: 0},
		Children:           children,
		RightToLeftLayout:  uiLayoutRTL(),
		RightToLeftReading: uiLayoutRTL(),
	}
	if shouldDeferWindowShow(st) {
		st.deferShow = true
		st.revealed = false
		// לא מציגים עדיין — אחרי Create נגדיר alpha=0 ואז Show שקוף
		cfg.Visible = false
	}
	if len(st.menuItems) > 0 {
		cfg.MenuItems = st.menuItems
	}
	if winIcon != nil {
		cfg.Icon = winIcon
	}
	if st.hasBg {
		cfg.Background = SolidColorBrush{Color: st.bgColor}
	}
	if st.dark {
		preferAppDarkMode()
	}
	if err := cfg.Create(); err != nil {
		return errObj("הצגת חלון נכשלה: " + err.Error())
	}

	st.mw = mw
	st.closed = false
	applyWindowTitleDir(mw)
	applyWindowPlacement(st)
	setTimerUISync(func(fn func()) {
		if st.mw != nil && !st.closed {
			st.mw.Synchronize(fn)
		} else {
			fn()
		}
	})

	if st.deferShow {
		applySplashTransparent(st)
	}

	if iconPath != "" {
		_ = applyWindowIcon(st, iconPath)
	}
	if st.dark {
		applyDarkThemeToWidget(mw)
	}
	applyDarkControlsRecursive(st.children, st.dark)

	for _, ch := range st.children {
		linkParentWindow(ch, st)
		wireBrowsersRecursive(ch, mw, st)
		wireGPUSurfacesRecursive(ch, mw)
		wireSurfaceResize(ch)
	}

	mw.SizeChanged().Attach(func() {
		enforceFixedWindowSize(st)
		resizeBrowsersRecursive(st.children)
		resizeGPUSurfacesRecursive(st.children)
		for _, ch := range st.children {
			wireSurfaceResize(ch)
			syncSurfaceSizesRecursive(ch)
		}
		applyWindowShape(st)
		if st.transparent {
			applyTransparentWindow(st)
		}
	})

	mw.Starting().Attach(func() {
		if st.deferShow {
			applySplashTransparent(st)
		}
		applyBorderlessWindow(st)
		if st.transparent {
			if st.mw != nil {
				stripHwndChromeTree(st.mw.Handle())
			}
			applyTransparentWindow(st)
		} else if st.mw != nil {
			clearWindowLayered(st.mw.Handle())
		}
		applyTopMostWindow(st)
		// אחרי שהחלון גלוי — נועלים מסגרת ומרעננים כותרת (כפתורי סגירה)
		applyFixedWindowSize(st)
		applyWindowShape(st)
		if st.transparent {
			applyTransparentWindow(st)
			for _, ch := range st.children {
				applyBrowserTransparency(ch, true)
			}
			if st.mw != nil {
				stripHwndChromeTree(st.mw.Handle())
				disableDwmChromeArtifacts(st.mw.Handle())
			}
		} else {
			for _, ch := range st.children {
				applyBrowserTransparency(ch, false)
			}
			// רק לווידג׳ט בלי מסגרת — לא לגעת בחלון רגיל (כותרת Windows)
			if st.borderless && st.mw != nil {
				clearWindowLayered(st.mw.Handle())
				stripHwndChromeTree(st.mw.Handle())
				disableDwmChromeArtifacts(st.mw.Handle())
				applyWindowShape(st)
			}
		}
		for _, ch := range st.children {
			wireSurfaceResize(ch)
			syncSurfaceSizesRecursive(ch)
			startBrowsersRecursive(ch)
		}
		if st.onStart != nil {
			invokeYod(st.onStart, nil)
		}
	})

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if st.forceClose {
			disposeWindowTray(st)
			st.closed = true
			return
		}
		if st.onClosing != nil {
			res := invokeYodResult(st.onClosing, nil)
			if object.Truthy(res) {
				*canceled = true
				hideMainWindow(st)
				return
			}
		}
		// אם יש מגש — X מסתיר לרקע גם בלי בסגירה מפורש
		if st.tray != nil || st.trayConfigured {
			*canceled = true
			hideMainWindow(st)
			return
		}
		disposeWindowTray(st)
		st.closed = true
	})

	if st.trayConfigured {
		if err := applyWindowTray(st); err != nil {
			// מגש נכשל — לא מונעים את פתיחת החלון
			fmt.Fprintf(os.Stderr, "אזהרה: חלון.מגש: %v\n", err)
		}
	}

	for _, t := range st.timers {
		t := t
		go func() {
			interval := time.Duration(t.seconds * float64(time.Second))
			if interval < 200*time.Millisecond {
				interval = 200 * time.Millisecond
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				if st.closed || st.mw == nil {
					return
				}
				st.mw.Synchronize(func() {
					if st.closed {
						return
					}
					invokeYod(t.fn, nil)
				})
			}
		}()
	}

	// לולאת ההודעות חוסמת עד סגירת החלון — משחררים את מנעול הריצה כדי שמשימות רקע
	// יוכלו לרוץ, וה־callbacks (שרצים על thread זה) יתפסו אותו מחדש דרך invokeYod.
	object.WithoutYodLock(func() {
		mw.Run()
	})
	st.closed = true
	st.mw = nil
	setTimerUISync(nil)
	_ = timerStopAll()
	soundStopAll()
	recordStopAll()
	return &object.Null{}
}

func applyWindowIcon(st *windowState, path string) error {
	ic, err := walk.NewIconFromFile(path)
	if err != nil {
		return err
	}
	if st.icon != nil && st.icon != ic {
		st.icon.Dispose()
	}
	st.icon = ic
	st.iconPath = path
	if st.mw != nil {
		return st.mw.SetIcon(ic)
	}
	return nil
}

func applyDarkTablesRecursive(children []*controlState) {
	applyDarkControlsRecursive(children, false)
}

func applyDarkControlsRecursive(children []*controlState, windowDark bool) {
	for _, ch := range children {
		if ch == nil {
			continue
		}
		if windowDark {
			switch ch.kind {
			case "טבלה":
				ch.tableDark = true
			case "גרף":
				ch.chartDark = true
			case "כפתור", "שדה", "רשימה":
				ch.ctrlDark = true
			case "תווית":
				if ch.textColor == walk.RGB(30, 30, 30) {
					ch.textColor = darkCtlText()
					if ch.label != nil {
						ch.label.SetTextColor(ch.textColor)
					}
				}
			}
		}
		switch ch.kind {
		case "טבלה":
			if ch.tableDark {
				applyTableDarkColors(ch)
			}
		case "כפתור":
			if ch.ctrlDark {
				applyButtonDark(ch)
			}
		case "שדה":
			if ch.ctrlDark {
				applyEditDark(ch)
			}
		case "רשימה":
			if ch.ctrlDark {
				applyListDark(ch)
			}
		case "גרף":
			if ch.chartDark {
				invalidateChart(ch)
			}
		}
		applyDarkControlsRecursive(ch.children, windowDark)
	}
}

func applyButtonDark(st *controlState) {
	if st == nil || st.button == nil || !st.ctrlDark {
		return
	}
	applyDarkThemeToWidget(st.button)
	clearWidgetFocusEffects(st.button)
	// כיבוי עיצוב ויזואלי של Win32 כדי שיאפשר רקע מותאם
	empty, _ := syscall.UTF16PtrFromString("")
	win.SetWindowTheme(st.button.Handle(), empty, empty)
	brush, err := walk.NewSolidColorBrush(darkBtnBG())
	if err == nil {
		st.button.SetBackground(brush)
	}
	st.button.Invalidate()
}

func applyEditDark(st *controlState) {
	if st == nil || st.edit == nil || !st.ctrlDark {
		return
	}
	applyDarkThemeToWidget(st.edit)
	clearWidgetFocusEffects(st.edit)
	brush, err := walk.NewSolidColorBrush(darkFieldBG())
	if err == nil {
		st.edit.SetBackground(brush)
	}
	st.edit.SetTextColor(darkCtlText())
	_ = st.edit.SetTextAlignment(walk.AlignFar)
	st.edit.Invalidate()
}

func applyListDark(st *controlState) {
	if st == nil || st.listBox == nil || !st.ctrlDark {
		return
	}
	applyDarkThemeToWidget(st.listBox)
	brush, err := walk.NewSolidColorBrush(darkPanelBG())
	if err == nil {
		st.listBox.SetBackground(brush)
	}
	st.listBox.Invalidate()
}

func wireBrowsersRecursive(ch *controlState, mw *walk.MainWindow, winSt *windowState) {
	if (ch.kind == "דפדפן" || ch.kind == "וידאו") && ch.host != nil {
		br, err := attachWebView2(ch.host, ch.url, ch.html, winSt)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		ch.browser = br
		ch.parentWin = winSt
		if ch.kind == "דפדפן" || ch.kind == "וידאו" {
			wireBrowserBridge(ch)
		}
		browserStartNavigation(ch)
	}
	for _, c := range ch.children {
		wireBrowsersRecursive(c, mw, winSt)
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
	if ch.kind == "וידאו" {
		videoLoadInitial(ch)
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
	walk.MsgBox(nil, I18nText(title), I18nText(msg), walk.MsgBoxIconInformation|msgBoxDirStyle())
	return &object.Null{}
}

// msgBoxDirStyle מחזיר דגלי כיוון ל־MessageBox לפי שפת הממשק הפעילה:
// RTL (עברית/ערבית וכו') → יישור לימין + סדר קריאה מימין לשמאל; LTR → ללא דגלים.
func msgBoxDirStyle() walk.MsgBoxStyle {
	if I18nDirection() == "rtl" {
		return walk.MsgBoxRight | walk.MsgBoxRTLReading
	}
	return 0
}
