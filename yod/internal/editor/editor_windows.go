//go:build windows

package editor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"yod/internal/compiler"
	"yod/internal/console"
	"yod/internal/evaluator"
	"yod/internal/format"
	"yod/internal/highlight"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/pack"
	"yod/internal/parser"
	"yod/internal/vm"
)

var (
	lineErrRe    = regexp.MustCompile(`(?i)(?:שורה|line)\s*(\d+)`)
	errLeadRe    = regexp.MustCompile(`(?i)^שגיאה\s*:?\s*`)
	errLinePrefRe = regexp.MustCompile(`(?i)^(?:ב?שורה|line)\s*\d+\s*[:：\-—–]?\s*`)
)

// פלטת Dark+ מקצועית — רקע עמוק + מבטא teal של יוד
var (
	colBg      = walk.RGB(24, 25, 28)
	colPanel   = walk.RGB(28, 29, 33)
	colToolbar = walk.RGB(36, 38, 43)
	colTabBar  = walk.RGB(32, 34, 38)
	colStatus  = walk.RGB(32, 34, 38)
	colBorder  = walk.RGB(48, 50, 56)
	colText    = walk.RGB(220, 220, 220)
	colMuted   = walk.RGB(140, 145, 150)
	colErrText = walk.RGB(244, 100, 100)
	colOutText = walk.RGB(78, 201, 176)
	colAccent  = walk.RGB(255, 255, 255)
	colLineNum = walk.RGB(110, 115, 122)
	colGutter  = walk.RGB(22, 23, 26)
	colBrand   = walk.RGB(78, 201, 176) // teal יוד
)

const uiFont = "Segoe UI"

const welcomeTemplate = `// ברוכים הבאים לעורך יוד
// F5 הרץ · Ctrl+Shift+P ארוז · Ctrl+F חיפוש · Tab הזחה

פונקציה שלום(שם)
  הדפס: "שלום " + שם
סוף

משתנה הודעה = "עולם"
שלום: הודעה
`

const newFileTemplate = `// קובץ יוד חדש

הדפס: "שלום עולם"
`

// Run פותח את עורך יוד — IDE RTL מקצועי.
func Run(path string) error {
	ensureProcessDPI()

	var (
		mw        *walk.MainWindow
		codeHost  *walk.Composite
		codeEdit  *CodeEdit
		lineEdit  *walk.TextEdit
		outEdit   *walk.TextEdit
		errEdit   *walk.TextEdit
		tabErrBtn *DarkBtn
		tabOutBtn *DarkBtn
		statusLbl *walk.Label
		fileLbl   *walk.Label
		projLbl   *walk.Label
		posLbl    *walk.Label
		linesLbl  *walk.Label
		modeLbl   *walk.Label
		treeView  *walk.TreeView
		treeEmpty *walk.Label
		treePane  *walk.Composite
		treeSplit *walk.Splitter
		toolbar   *walk.Composite
		tabBar    *walk.Composite
		activeTab = 0
		errCount  int
		busy      bool
	)

	codeFace := pickCodeFont()

	currentPath := path
	projectRoot := ""
	treeModel := NewFileTreeModel()
	dirty := false
	var errLines []int
	errLineSet := map[int]bool{}
	var updateLineNumbers func()
	var syncLineScroll func()
	var updateCaretStatus func()
	var refreshProjectUI func()
	var selectPathInTree func(string)
	lastGutterLines := 0

	fileName := func() string {
		if currentPath == "" {
			return "קובץ-חדש.יוד"
		}
		return filepath.Base(currentPath)
	}

	updateTitle := func() {
		mark := ""
		if dirty {
			mark = " •"
		}
		name := fileName()
		if mw != nil {
			title := "עורך יוד — " + name + mark
			if projectRoot != "" {
				title = filepath.Base(projectRoot) + " — " + name + mark
			}
			mw.SetTitle(title)
		}
		if fileLbl != nil {
			shown := name + mark
			if dirty {
				shown = "● " + name
			}
			fileLbl.SetText(shown)
		}
	}

	refreshProjectUI = func() {
		if projLbl != nil {
			if projectRoot == "" {
				projLbl.SetText("אין תיקייה פתוחה")
			} else {
				projLbl.SetText("תיקייה: " + projectRoot)
			}
		}
		if treeEmpty != nil {
			treeEmpty.SetVisible(projectRoot == "")
		}
		if treeView != nil {
			treeView.SetVisible(projectRoot != "")
		}
		if mw != nil {
			mw.RequestLayout()
		}
	}

	setStatus := func(msg string) {
		if statusLbl != nil {
			statusLbl.SetText(msg)
		}
	}

	updateCaretStatus = func() {
		if codeEdit == nil {
			return
		}
		start, _ := codeEdit.TextSelection()
		line := codeEdit.LineFromChar(start) + 1
		lineStart := codeEdit.LineIndex(line - 1)
		col := start - lineStart + 1
		if col < 1 {
			col = 1
		}
		total := codeEdit.LineCount()
		if posLbl != nil {
			posLbl.SetText(fmt.Sprintf("שורה %d, עמודה %d", line, col))
		}
		if linesLbl != nil {
			linesLbl.SetText(fmt.Sprintf("%d שורות", total))
		}
	}

	refreshTabLabels := func() {
		errLabel := "שגיאות"
		if errCount > 0 {
			errLabel = fmt.Sprintf("שגיאות (%d)", errCount)
		}
		outLabel := "פלט"
		if tabErrBtn != nil {
			tabErrBtn.SetText(errLabel)
			tabErrBtn.SetActive(activeTab == 0)
		}
		if tabOutBtn != nil {
			tabOutBtn.SetText(outLabel)
			tabOutBtn.SetActive(activeTab == 1)
		}
	}

	baseDir := func() string {
		if projectRoot != "" {
			return projectRoot
		}
		if currentPath != "" {
			return filepath.Dir(currentPath)
		}
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}

	selectPathInTree = func(p string) {
		if treeView == nil || projectRoot == "" || p == "" {
			return
		}
		node := treeModel.FindByPath(p)
		if node == nil {
			return
		}
		_ = treeView.SetCurrentItem(node)
		_ = treeView.EnsureVisible(node)
	}

	showErrorsTab := func() {
		activeTab = 0
		if errEdit != nil {
			errEdit.SetVisible(true)
		}
		if outEdit != nil {
			outEdit.SetVisible(false)
		}
		refreshTabLabels()
		if mw != nil {
			mw.RequestLayout()
		}
	}
	showOutputTab := func() {
		activeTab = 1
		if errEdit != nil {
			errEdit.SetVisible(false)
		}
		if outEdit != nil {
			outEdit.SetVisible(true)
		}
		refreshTabLabels()
		if mw != nil {
			mw.RequestLayout()
		}
	}

	clearPanels := func() {
		setPanelText(outEdit, "")
		setPanelText(errEdit, "")
		errLines = nil
		errCount = 0
		errLineSet = map[int]bool{}
		refreshTabLabels()
		updateLineNumbers()
	}

	setErrors := func(msgs []string) {
		errLineSet = map[int]bool{}
		if len(msgs) == 0 {
			setPanelText(errEdit, "אין שגיאות")
			errLines = []int{0}
			errCount = 0
			refreshTabLabels()
			updateLineNumbers()
			return
		}
		display, lines := formatErrorPanel(msgs)
		errCount = len(lines)
		errLines = lines
		for _, ln := range lines {
			if ln > 0 {
				errLineSet[ln] = true
			}
		}
		setPanelText(errEdit, display)
		showErrorsTab()
		updateLineNumbers()
	}

	setOutput := func(s string) {
		setPanelText(outEdit, s)
		if strings.TrimSpace(s) != "" {
			showOutputTab()
		}
	}

	gotoLine := func(line int) {
		if line < 1 || codeEdit == nil {
			return
		}
		src := codeEdit.Text()
		start, end := lineOffsets(src, line)
		if start < 0 {
			return
		}
		// RichEdit: אינדקסי תו (CRLF = 1)
		codeEdit.SetTextSelection(highlight.RichEditIndex(src, start), highlight.RichEditIndex(src, end))
		codeEdit.SetFocus()
		codeEdit.ScrollCaret()
		updateCaretStatus()
	}

	jumpFromErrPanel := func() {
		if errEdit == nil || len(errLines) == 0 {
			return
		}
		start, _ := errEdit.TextSelection()
		idx := lineIndexAt(errEdit.Text(), start)
		if idx >= 0 && idx < len(errLines) && errLines[idx] > 0 {
			gotoLine(errLines[idx])
		}
	}

	updateLineNumbers = func() {
		if lineEdit == nil || codeEdit == nil {
			return
		}
		n := codeEdit.LineCount()
		width := len(strconv.Itoa(n))
		if width < 3 {
			width = 3
		}
		var b strings.Builder
		b.Grow(n * (width + 3))
		for i := 1; i <= n; i++ {
			if i > 1 {
				b.WriteString("\r\n")
			}
			if errLineSet[i] {
				fmt.Fprintf(&b, "●%*d", width, i)
			} else {
				fmt.Fprintf(&b, " %*d", width, i)
			}
		}
		lineEdit.SetText(b.String())
		if len(errLineSet) > 0 {
			lineEdit.SetTextColor(colErrText)
		} else {
			lineEdit.SetTextColor(colLineNum)
		}
		updateCaretStatus()
	}

	syncLineScroll = func() {
		if codeEdit == nil || lineEdit == nil {
			return
		}
		first := codeEdit.FirstVisibleLine()
		gut := int(lineEdit.SendMessage(win.EM_GETFIRSTVISIBLELINE, 0, 0))
		if first != gut {
			lineEdit.SendMessage(win.EM_LINESCROLL, 0, uintptr(first-gut))
		}
		updateCaretStatus()
	}

	confirmDiscard := func(action string) bool {
		if !dirty {
			return true
		}
		r := walk.MsgBox(mw, "עורך יוד",
			"יש שינויים שלא נשמרו.\n"+action+"?",
			walk.MsgBoxYesNoCancel|walk.MsgBoxIconWarning)
		if r == walk.DlgCmdCancel {
			return false
		}
		if r == walk.DlgCmdYes {
			if err := saveFileRef(mw, &currentPath, codeEdit, &dirty, updateTitle, setStatus); err != nil {
				walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
				return false
			}
		}
		return true
	}

	loadFile := func(p string) error {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		currentPath = p
		dirty = false
		codeEdit.SetText(string(data))
		clearPanels()
		setPanelText(errEdit, "אין שגיאות")
		setPanelText(outEdit, "(אין פלט)")
		updateLineNumbers()
		updateTitle()
		setStatus("נפתח · " + filepath.Base(p))
		if modeLbl != nil {
			modeLbl.SetText(fmt.Sprintf("UTF-8 · .יוד · %dpt", codeFontSize))
		}
		selectPathInTree(p)
		return nil
	}

	saveFile := func(forceDialog bool) error {
		p := currentPath
		if p == "" || forceDialog {
			dlg := new(walk.FileDialog)
			dlg.Title = "שמירת קובץ יוד"
			dlg.Filter = "קבצי יוד (*.יוד)|*.יוד|כל הקבצים (*.*)|*.*"
			if p != "" {
				dlg.FilePath = p
			} else if projectRoot != "" {
				dlg.FilePath = filepath.Join(projectRoot, "תוכנית.יוד")
			} else {
				dlg.FilePath = "תוכנית.יוד"
			}
			ok, err := dlg.ShowSave(mw)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			p = dlg.FilePath
			if filepath.Ext(p) == "" {
				p += ".יוד"
			}
		}
		if err := os.WriteFile(p, []byte(codeEdit.Text()), 0644); err != nil {
			return err
		}
		currentPath = p
		dirty = false
		updateTitle()
		setStatus("נשמר · " + filepath.Base(p))
		if projectRoot != "" {
			treeModel.Refresh()
			selectPathInTree(p)
		}
		return nil
	}

	newFile := func() {
		if !confirmDiscard("לשמור לפני קובץ חדש") {
			return
		}
		currentPath = ""
		dirty = false
		codeEdit.SetText(newFileTemplate)
		clearPanels()
		setPanelText(errEdit, "אין שגיאות")
		setPanelText(outEdit, "(אין פלט)")
		updateLineNumbers()
		updateTitle()
		setStatus("קובץ חדש")
		showErrorsTab()
	}

	openFile := func() {
		if !confirmDiscard("לשמור לפני פתיחה") {
			return
		}
		dlg := new(walk.FileDialog)
		dlg.Title = "פתיחת קובץ יוד"
		dlg.Filter = "קבצי יוד (*.יוד)|*.יוד|כל הקבצים (*.*)|*.*"
		ok, err := dlg.ShowOpen(mw)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		if !ok {
			return
		}
		if err := loadFile(dlg.FilePath); err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
		}
	}

	withBusy := func(label string, fn func()) {
		if busy {
			return
		}
		busy = true
		setStatus(label)
		defer func() { busy = false }()
		fn()
	}

	compileCheck := func() {
		withBusy("בודק קומפילציה…", func() {
			source := codeEdit.Text()
			clearPanels()
			pr := parser.New(lexer.New(source))
			program := pr.ParseProgram()
			if errs := pr.Errors(); len(errs) > 0 {
				setErrors(errs)
				setOutput("")
				setStatus(fmt.Sprintf("✗ שגיאות תחביר · %d", len(errs)))
				gotoLine(extractLine(errs[0]))
				return
			}
			c := compiler.New(baseDir())
			if err := c.Compile(program); err != nil {
				setErrors([]string{err.Error()})
				setStatus("✗ שגיאת קומפילציה")
				return
			}
			setErrors(nil)
			setOutput("✓ הקומפילציה למכונה הצליחה.\nאין שגיאות — אפשר להריץ עם «מכונה» (F6).")
			setStatus("✓ קומפילציה תקינה")
		})
	}

	runInterpreter := func() {
		source := codeEdit.Text()
		clearPanels()

		pr := parser.New(lexer.New(source))
		program := pr.ParseProgram()
		if errs := pr.Errors(); len(errs) > 0 {
			setErrors(errs)
			setStatus(fmt.Sprintf("✗ שגיאות תחביר · %d", len(errs)))
			gotoLine(extractLine(errs[0]))
			return
		}
		setErrors(nil)

		// חלונות/ציור: MainWindow בתוך העורך סוגר את כל התהליך — מריצים בתהליך נפרד
		if strings.Contains(source, "חלונות") || strings.Contains(source, "ציור") {
			exe, err := os.Executable()
			if err != nil {
				setErrors([]string{"לא הצלחתי למצוא את yod.exe: " + err.Error()})
				setStatus("✗ הרצה נכשלה")
				return
			}
			dir := baseDir()
			runPath := filepath.Join(dir, ".yod-run-temp.יוד")
			if err := os.WriteFile(runPath, []byte(source), 0644); err != nil {
				setErrors([]string{"לא הצלחתי לכתוב קובץ הרצה זמני: " + err.Error()})
				setStatus("✗ הרצה נכשלה")
				return
			}
			cmd := exec.Command(exe, "run", runPath)
			cmd.Dir = dir
			var outBuf, errBuf bytes.Buffer
			cmd.Stdout = &outBuf
			cmd.Stderr = &errBuf
			if err := cmd.Start(); err != nil {
				_ = os.Remove(runPath)
				setErrors([]string{"לא הצלחתי להפעיל תהליך נפרד: " + err.Error()})
				setStatus("✗ הרצה נכשלה")
				return
			}
			setOutput("הופעל חלון נפרד (ספריית חלונות).\nסגירת חלון התוכנית לא סוגרת את העורך.")
			setStatus("✓ רץ בחלון נפרד")
			go func(c *exec.Cmd, tmp string) {
				waitErr := c.Wait()
				_ = os.Remove(tmp)
				stdout := strings.TrimSpace(outBuf.String())
				stderr := strings.TrimSpace(errBuf.String())
				mw.Synchronize(func() {
					if waitErr != nil || stderr != "" {
						msg := stderr
						if msg == "" {
							msg = waitErr.Error()
						}
						setErrors([]string{msg})
						if stdout != "" {
							setOutput(stdout)
						}
						showErrorsTab()
						setStatus("✗ שגיאה בחלון נפרד")
						return
					}
					if stdout != "" {
						setOutput(stdout)
					}
					setStatus("✓ חלון התוכנית נסגר")
				})
			}(cmd, runPath)
			return
		}

		withBusy("מריץ · מפרש…", func() {
			var buf bytes.Buffer
			console.SetStdout(&buf)
			defer console.SetStdout(nil)

			defer func() {
				if r := recover(); r != nil {
					setErrors([]string{fmt.Sprintf("קריסה פנימית בהרצה: %v", r)})
					setStatus("✗ קריסה")
				}
			}()

			env := evaluator.NewGlobalEnv(baseDir())
			result := evaluator.Eval(program, env)
			out := buf.String()
			if result != nil && result.Type() == object.ErrorObj {
				msg := result.Inspect()
				setErrors([]string{msg})
				if out != "" {
					setOutput(out)
					showErrorsTab()
				}
				setStatus("✗ שגיאת ריצה")
				gotoLine(extractLine(msg))
				return
			}
			if out == "" {
				out = "(אין פלט)"
			}
			setOutput(out)
			setStatus("✓ הושלם · מפרש")
		})
	}

	runMachine := func() {
		withBusy("מריץ · מכונה…", func() {
			source := codeEdit.Text()
			clearPanels()
			var buf bytes.Buffer
			console.SetStdout(&buf)
			defer console.SetStdout(nil)

			pr := parser.New(lexer.New(source))
			program := pr.ParseProgram()
			if errs := pr.Errors(); len(errs) > 0 {
				setErrors(errs)
				setStatus(fmt.Sprintf("✗ שגיאות תחביר · %d", len(errs)))
				gotoLine(extractLine(errs[0]))
				return
			}

			c := compiler.New(baseDir())
			if err := c.Compile(program); err != nil {
				setErrors([]string{err.Error()})
				setOutput("טיפ: אם המכונה לא תומכת בפקודה — נסו «הרץ» (F5).")
				setStatus("✗ שגיאת קומפילציה")
				return
			}
			setErrors(nil)

			machine := vm.New(c.Bytecode())
			object.InvokeCallable = func(fn object.Object, args []object.Object) object.Object {
				switch f := fn.(type) {
				case *object.Function:
					if object.InvokeFunction != nil {
						return object.InvokeFunction(f, args)
					}
				case *object.Closure, *object.CompiledFunction:
					return machine.CallCallable(fn, args)
				}
				return &object.Error{Message: "בלחיצה מצפה לפונקציה"}
			}
			if err := machine.Run(); err != nil {
				msg := err.Error()
				setErrors([]string{msg})
				out := buf.String()
				if out != "" {
					setOutput(out)
					showErrorsTab()
				}
				setStatus("✗ שגיאת מכונה")
				gotoLine(extractLine(msg))
				return
			}
			out := buf.String()
			if out == "" {
				out = "(אין פלט)"
			}
			setOutput(out)
			setStatus("✓ הושלם · מכונה")
		})
	}

	showHighlight := func() {
		html := highlight.ToHTML(codeEdit.Text())
		tmp := filepath.Join(os.TempDir(), "yod-highlight.html")
		if err := os.WriteFile(tmp, []byte(html), 0644); err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		_ = exec.Command("cmd", "/c", "start", "", tmp).Start()
		setStatus("הדגשת תחביר · דפדפן")
	}

	formatCode := func() {
		if codeEdit == nil {
			return
		}
		src := codeEdit.Text()
		out := format.Source(src)
		normIn := strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n")
		if out == normIn {
			setStatus("הקוד כבר מסודר")
			return
		}
		_ = codeEdit.SetText(out)
		codeEdit.SetTextSelection(0, 0)
		dirty = true
		updateTitle()
		setStatus("✓ הקוד סודר")
	}

	packProgram := func() {
		withBusy("אורז להפצה…", func() {
			if currentPath == "" || dirty {
				if err := saveFile(currentPath == ""); err != nil {
					walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
					return
				}
				if currentPath == "" {
					walk.MsgBox(mw, "ארוז", "צריך לשמור את הקובץ לפני אריזה.", walk.MsgBoxIconWarning)
					return
				}
			}
			outPath, err := pack.EXEWithOptions(currentPath, "", pack.EXEOptions{})
			if err != nil {
				setErrors([]string{err.Error()})
				setStatus("✗ אריזה נכשלה")
				walk.MsgBox(mw, "שגיאה באריזה", err.Error(), walk.MsgBoxIconError)
				return
			}
			msg := "נוצר קובץ EXE (בלי חלון CMD):\n" + outPath + "\n\nלחיצה כפולה מריצה את התוכנית."
			setErrors(nil)
			setOutput(msg)
			setStatus("✓ נארז · " + filepath.Base(outPath))
			showOutputTab()
			walk.MsgBox(mw, "ארוז", msg, walk.MsgBoxOK|walk.MsgBoxIconInformation)
		})
	}

	showAbout := func() {
		walk.MsgBox(mw, "עורך יוד",
			"עורך יוד — סביבת פיתוח בעברית\n\n"+
				"סייר קבצים: קובץ ← פתח תיקייה (Ctrl+Shift+O)\n"+
				"לחיצה כפולה על קובץ · תפריט ימני: חדש/מחק/שנה שם\n\n"+
				"השלמת קוד: מילות מפתח, מתודות אחרי נקודה, ספריות ב־כלול\n"+
				"חצים · Tab/Enter אישור · Esc · Ctrl+Space\n\n"+
				"F5 / F6 / F7  הרץ / מכונה / בדוק\n"+
				"Ctrl+Shift+P  ארוז ל־EXE\n"+
				"Ctrl+N / O / S  חדש / פתח / שמור\n"+
				"Ctrl+Shift+O  פתח תיקייה\n"+
				"Ctrl+Z / Y  בטל / בצע שוב\n"+
				"Ctrl+F / H  חיפוש / החלפה\n"+
				"Ctrl+G  מעבר לשורה\n"+
				"Ctrl+/  הערה · Ctrl+Shift+D  שכפול שורה\n"+
				"Tab / Shift+Tab  הזחה / החזרת הזחה\n"+
				"Enter  שורה חדשה עם הזחה\n"+
				"Ctrl+Shift+F  סדר קוד · Ctrl± גודל גופן\n\n"+
				"● בשוליים = שורת שגיאה · לחיצה על שגיאה קופצת לשורה",
			walk.MsgBoxOK|walk.MsgBoxIconInformation)
	}

	showFind := func() { showFindDialog(mw, codeEdit, false) }
	showReplace := func() { showFindDialog(mw, codeEdit, true) }
	showGoto := func() { showGotoLineDialog(mw, gotoLine) }
	editUndo := func() {
		if codeEdit != nil {
			codeEdit.Undo()
		}
	}
	editRedo := func() {
		if codeEdit != nil {
			codeEdit.Redo()
		}
	}
	editComment := func() {
		if codeEdit != nil {
			codeEdit.toggleComment()
		}
	}
	editDuplicate := func() {
		if codeEdit != nil {
			codeEdit.duplicateLines()
		}
	}
	editSelectAll := func() {
		if codeEdit != nil {
			codeEdit.SetTextSelection(0, -1)
		}
	}

	applyCodeZoom := func(size int) {
		if lineEdit != nil {
			if f, err := walk.NewFont(codeFace, size, 0); err == nil {
				lineEdit.SetFont(f)
			}
		}
		if modeLbl != nil {
			modeLbl.SetText(fmt.Sprintf("UTF-8 · יוד · %dpt", size))
		}
	}

	btnRun := &DarkBtn{text: "הרץ", icon: iconRun, primary: true, onClick: runInterpreter}
	btnVM := &DarkBtn{text: "מכונה", icon: iconVM, onClick: runMachine}
	btnCheck := &DarkBtn{text: "בדוק", icon: iconCheck, onClick: compileCheck}
	btnPack := &DarkBtn{text: "ארוז", icon: iconPack, onClick: packProgram}
	btnNew := &DarkBtn{text: "חדש", icon: iconNew, onClick: newFile}
	btnOpen := &DarkBtn{text: "פתח", icon: iconOpen, onClick: openFile}
	btnSave := &DarkBtn{text: "שמור", icon: iconSave, onClick: func() {
		if err := saveFile(false); err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
		}
	}}
	btnFormat := &DarkBtn{text: "סדר", icon: iconFormat, onClick: formatCode}
	btnHighlight := &DarkBtn{text: "הדגש", icon: iconHighlight, onClick: showHighlight}
	tabErrBtn = &DarkBtn{text: "שגיאות", icon: iconError, onClick: func() { showErrorsTab() }}
	tabOutBtn = &DarkBtn{text: "פלט", icon: iconOutput, onClick: func() { showOutputTab() }}

	selectedTreeNode := func() *fileNode {
		if treeView == nil {
			return nil
		}
		it := treeView.CurrentItem()
		if it == nil {
			return nil
		}
		n, _ := it.(*fileNode)
		return n
	}

	openFolder := func() {
		initial := ""
		if projectRoot != "" {
			initial = projectRoot
		} else if currentPath != "" {
			initial = filepath.Dir(currentPath)
		}
		path, ok, err := pickFolder(mw, "פתיחת תיקיית פרויקט", initial)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		if !ok || path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
		projectRoot = path
		treeModel.SetRoot(projectRoot)
		refreshProjectUI()
		updateTitle()
		setStatus("תיקייה · " + filepath.Base(projectRoot))
		if treeView != nil && treeModel.root != nil {
			_ = treeView.SetCurrentItem(treeModel.root)
			_ = treeView.SetExpanded(treeModel.root, true)
		}
	}

	closeFolder := func() {
		projectRoot = ""
		treeModel.SetRoot("")
		refreshProjectUI()
		updateTitle()
		setStatus("תיקייה נסגרה")
	}

	refreshTree := func() {
		if projectRoot == "" {
			return
		}
		treeModel.Refresh()
		refreshProjectUI()
		if currentPath != "" {
			selectPathInTree(currentPath)
		}
		setStatus("העץ רוענן")
	}

	openTreeSelection := func() {
		n := selectedTreeNode()
		if n == nil || n.isDir {
			return
		}
		if !confirmDiscard("לשמור לפני פתיחת קובץ") {
			return
		}
		if err := loadFile(n.path); err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
		}
	}

	treeNewFile := func() {
		if projectRoot == "" {
			walk.MsgBox(mw, "סייר", "פתחו תיקייה קודם (קובץ ← פתח תיקייה).", walk.MsgBoxIconInformation)
			return
		}
		dir := ParentDirForNew(selectedTreeNode(), projectRoot)
		name, ok := promptTextDialog(mw, "קובץ חדש", "שם הקובץ:", "חדש.יוד")
		if !ok {
			return
		}
		path, err := createNewFileOnDisk(dir, name)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		treeModel.Refresh()
		if !confirmDiscard("לשמור לפני פתיחת הקובץ החדש") {
			selectPathInTree(path)
			return
		}
		if err := loadFile(path); err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
		}
	}

	treeNewFolder := func() {
		if projectRoot == "" {
			walk.MsgBox(mw, "סייר", "פתחו תיקייה קודם.", walk.MsgBoxIconInformation)
			return
		}
		dir := ParentDirForNew(selectedTreeNode(), projectRoot)
		name, ok := promptTextDialog(mw, "תיקייה חדשה", "שם התיקייה:", "תיקייה")
		if !ok {
			return
		}
		path, err := createNewFolderOnDisk(dir, name)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		treeModel.Refresh()
		selectPathInTree(path)
		setStatus("נוצרה תיקייה · " + name)
	}

	treeRename := func() {
		n := selectedTreeNode()
		if n == nil || (treeModel.root != nil && n == treeModel.root) {
			walk.MsgBox(mw, "שינוי שם", "בחרו קובץ או תיקייה (לא את שורש הפרויקט).", walk.MsgBoxIconInformation)
			return
		}
		newName, ok := promptTextDialog(mw, "שינוי שם", "שם חדש:", n.name)
		if !ok {
			return
		}
		oldPath := n.path
		newPath, err := renamePathOnDisk(oldPath, newName)
		if err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		if currentPath != "" && filepath.Clean(currentPath) == filepath.Clean(oldPath) {
			currentPath = newPath
			updateTitle()
		}
		treeModel.Refresh()
		selectPathInTree(newPath)
		setStatus("שם שונה · " + newName)
	}

	treeDelete := func() {
		n := selectedTreeNode()
		if n == nil || (treeModel.root != nil && n == treeModel.root) {
			walk.MsgBox(mw, "מחיקה", "בחרו קובץ או תיקייה למחיקה.", walk.MsgBoxIconInformation)
			return
		}
		kind := "קובץ"
		if n.isDir {
			kind = "תיקייה"
		}
		r := walk.MsgBox(mw, "מחיקה",
			fmt.Sprintf("למחוק את ה%s %q?\nהפעולה אינה ניתנת לביטול.", kind, n.name),
			walk.MsgBoxYesNo|walk.MsgBoxIconWarning)
		if r != walk.DlgCmdYes {
			return
		}
		path := n.path
		if err := deletePathOnDisk(path, n.isDir); err != nil {
			walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
			return
		}
		if currentPath != "" && (filepath.Clean(currentPath) == filepath.Clean(path) ||
			strings.HasPrefix(filepath.Clean(currentPath)+string(os.PathSeparator), filepath.Clean(path)+string(os.PathSeparator))) {
			currentPath = ""
			dirty = false
			codeEdit.SetText(newFileTemplate)
			updateTitle()
		}
		treeModel.Refresh()
		setStatus("נמחק · " + n.name)
	}

	treeReveal := func() {
		n := selectedTreeNode()
		path := projectRoot
		if n != nil {
			path = n.path
		}
		if path == "" {
			return
		}
		revealInExplorer(path)
	}

	btnFolder := &DarkBtn{text: "תיקייה", icon: iconOpen, onClick: openFolder}

	err := MainWindow{
		AssignTo:          &mw,
		Title:             "עורך יוד",
		MinSize:           Size{Width: 1000, Height: 680},
		Size:              Size{Width: 1240, Height: 820},
		Layout:            VBox{MarginsZero: true, Spacing: 0},
		RightToLeftLayout: true,
		Background:        SolidColorBrush{Color: colBg},
		Font:              Font{Family: uiFont, PointSize: 10},
		MenuItems: []MenuItem{
			Menu{
				Text: "קובץ",
				Items: []MenuItem{
					Action{Text: "חדש\tCtrl+N", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyN}, OnTriggered: newFile},
					Action{Text: "פתח…\tCtrl+O", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyO}, OnTriggered: openFile},
					Action{Text: "פתח תיקייה…\tCtrl+Shift+O", Shortcut: Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyO}, OnTriggered: openFolder},
					Action{Text: "סגור תיקייה", OnTriggered: closeFolder},
					Separator{},
					Action{Text: "שמור\tCtrl+S", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyS}, OnTriggered: func() {
						if err := saveFile(false); err != nil {
							walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
						}
					}},
					Action{Text: "שמור בשם…", OnTriggered: func() {
						if err := saveFile(true); err != nil {
							walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
						}
					}},
					Separator{},
					Action{Text: "יציאה", OnTriggered: func() { mw.Close() }},
				},
			},
			Menu{
				Text: "סייר",
				Items: []MenuItem{
					Action{Text: "רענון עץ", OnTriggered: refreshTree},
					Separator{},
					Action{Text: "קובץ חדש בתיקייה…", OnTriggered: treeNewFile},
					Action{Text: "תיקייה חדשה…", OnTriggered: treeNewFolder},
					Action{Text: "שינוי שם…", OnTriggered: treeRename},
					Action{Text: "מחק…", OnTriggered: treeDelete},
					Separator{},
					Action{Text: "הצג בסייר Windows", OnTriggered: treeReveal},
				},
			},
			Menu{
				Text: "עריכה",
				Items: []MenuItem{
					Action{Text: "בטל\tCtrl+Z", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyZ}, OnTriggered: editUndo},
					Action{Text: "בצע שוב\tCtrl+Y", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyY}, OnTriggered: editRedo},
					Separator{},
					Action{Text: "בחר הכל\tCtrl+A", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyA}, OnTriggered: editSelectAll},
					Separator{},
					Action{Text: "חיפוש…\tCtrl+F", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyF}, OnTriggered: showFind},
					Action{Text: "החלפה…\tCtrl+H", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyH}, OnTriggered: showReplace},
					Action{Text: "מעבר לשורה…\tCtrl+G", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyG}, OnTriggered: showGoto},
					Separator{},
					Action{Text: "הערה / ביטול הערה\tCtrl+/", OnTriggered: editComment},
					Action{Text: "שכפול שורה\tCtrl+Shift+D", Shortcut: Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyD}, OnTriggered: editDuplicate},
				},
			},
			Menu{
				Text: "הרצה",
				Items: []MenuItem{
					Action{Text: "הרץ (מפרש)\tF5", Shortcut: Shortcut{Key: walk.KeyF5}, OnTriggered: runInterpreter},
					Action{Text: "מכונה (bytecode)\tF6", Shortcut: Shortcut{Key: walk.KeyF6}, OnTriggered: runMachine},
					Action{Text: "בדוק קומפילציה\tF7", Shortcut: Shortcut{Key: walk.KeyF7}, OnTriggered: compileCheck},
					Separator{},
					Action{Text: "ארוז ל־EXE\tCtrl+Shift+P", Shortcut: Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyP}, OnTriggered: packProgram},
				},
			},
			Menu{
				Text: "תצוגה",
				Items: []MenuItem{
					Action{Text: "סדר קוד\tCtrl+Shift+F", Shortcut: Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyF}, OnTriggered: formatCode},
					Action{Text: "הדגשת תחביר בדפדפן", OnTriggered: showHighlight},
					Separator{},
					Action{Text: "הגדל גופן\tCtrl+=", OnTriggered: func() {
						if codeEdit != nil {
							codeEdit.zoom(+1)
						}
					}},
					Action{Text: "הקטן גופן\tCtrl+-", OnTriggered: func() {
						if codeEdit != nil {
							codeEdit.zoom(-1)
						}
					}},
					Action{Text: "איפוס גודל\tCtrl+0", OnTriggered: func() {
						if codeEdit != nil {
							codeFontSize = 14
							codeEdit.rehighlight()
							applyCodeZoom(14)
						}
					}},
					Separator{},
					Action{Text: "לשונית שגיאות", OnTriggered: showErrorsTab},
					Action{Text: "לשונית פלט", OnTriggered: showOutputTab},
				},
			},
			Menu{
				Text: "עזרה",
				Items: []MenuItem{
					Action{Text: "קיצורי מקלדת…", OnTriggered: showAbout},
				},
			},
		},
		Children: []Widget{
			// —— סרגל כלים ——
			Composite{
				AssignTo:   &toolbar,
				Layout:     HBox{Margins: Margins{Left: 6, Right: 6, Top: 2, Bottom: 2}, Spacing: 3},
				Background: SolidColorBrush{Color: colToolbar},
				MinSize:    Size{Height: 26},
				MaxSize:    Size{Height: 26},
				Children: []Widget{
					Label{Text: "יוד", TextColor: colBrand, Font: Font{Family: uiFont, PointSize: 10, Bold: true}},
					Label{Text: "עורך", TextColor: colMuted, Font: Font{Family: uiFont, PointSize: 8}},
					VSeparator{},
				},
			},
			// —— שורת קובץ + תיקייה ——
			Composite{
				Layout:     HBox{Margins: Margins{Left: 10, Right: 10, Top: 2, Bottom: 2}, Spacing: 6},
				Background: SolidColorBrush{Color: colTabBar},
				Children: []Widget{
					Label{AssignTo: &fileLbl, Text: "קובץ-חדש.יוד", TextColor: colText, Font: Font{Family: uiFont, PointSize: 9}, RightToLeftReading: true},
					Label{Text: "·", TextColor: colBorder},
					Label{AssignTo: &projLbl, Text: "אין תיקייה פתוחה", TextColor: colMuted, Font: Font{Family: uiFont, PointSize: 8}, RightToLeftReading: true},
					HSpacer{},
					Label{Text: "RTL · עברית", TextColor: colMuted, Font: Font{Family: uiFont, PointSize: 8}, RightToLeftReading: true},
				},
			},
			Composite{MinSize: Size{Height: 1}, Background: SolidColorBrush{Color: colBorder}},
			// —— סייר קבוע משמאל | עורך מימין (שורת העבודה ב־LTR) ——
			HSplitter{
				AssignTo:      &treeSplit,
				StretchFactor: 1,
				HandleWidth:   8,
				Children: []Widget{
					// סייר — תמיד צד שמאל של המסך
					Composite{
						AssignTo:      &treePane,
						Layout:        VBox{MarginsZero: true, Spacing: 0},
						Background:    SolidColorBrush{Color: colToolbar},
						MinSize:       Size{Width: 100},
						StretchFactor: 1,
						Children: []Widget{
							Composite{
								Layout:     HBox{Margins: Margins{Left: 8, Right: 6, Top: 6, Bottom: 4}, Spacing: 6},
								Background: SolidColorBrush{Color: colToolbar},
								Children: []Widget{
									Label{Text: "סייר", TextColor: colBrand, Font: Font{Family: uiFont, PointSize: 10, Bold: true}, RightToLeftReading: true},
									HSpacer{},
								},
							},
							Composite{MinSize: Size{Height: 1}, Background: SolidColorBrush{Color: colBorder}},
							Label{
								AssignTo:           &treeEmpty,
								Text:               "פתחו תיקייה\n(קובץ ← פתח תיקייה)\nכדי לראות את העץ",
								TextColor:          colMuted,
								Font:               Font{Family: uiFont, PointSize: 9},
								RightToLeftReading: true,
								MinSize:            Size{Height: 60},
							},
							TreeView{
								AssignTo:      &treeView,
								Model:         treeModel,
								Visible:       false,
								MinSize:       Size{Height: 200},
								StretchFactor: 1,
								Font:          Font{Family: uiFont, PointSize: 9},
								Background:    SolidColorBrush{Color: colPanel},
								ContextMenuItems: []MenuItem{
									Action{Text: "פתח", OnTriggered: openTreeSelection},
									Separator{},
									Action{Text: "קובץ חדש…", OnTriggered: treeNewFile},
									Action{Text: "תיקייה חדשה…", OnTriggered: treeNewFolder},
									Action{Text: "שינוי שם…", OnTriggered: treeRename},
									Action{Text: "מחק…", OnTriggered: treeDelete},
									Separator{},
									Action{Text: "רענון", OnTriggered: refreshTree},
									Action{Text: "הצג בסייר Windows", OnTriggered: treeReveal},
								},
								OnItemActivated: openTreeSelection,
							},
						},
					},
					// עמודת עורך (~80%)
					Composite{
						Layout:        VBox{MarginsZero: true, Spacing: 0},
						Background:    SolidColorBrush{Color: colBg},
						StretchFactor: 4,
						Children: []Widget{
							Composite{
								AssignTo:      &codeHost,
								Layout:        HBox{MarginsZero: true, Spacing: 0},
								Background:    SolidColorBrush{Color: colPanel},
								StretchFactor: 5,
								MinSize:       Size{Height: 280},
								Children: []Widget{
									TextEdit{
										AssignTo:      &lineEdit,
										ReadOnly:      true,
										VScroll:       false,
										TextAlignment: AlignFar,
										TextColor:     colLineNum,
										Background:    SolidColorBrush{Color: colGutter},
										Font:          Font{Family: codeFace, PointSize: 14},
										MinSize:       Size{Width: 56, Height: 280},
										MaxSize:       Size{Width: 72},
										OnMouseDown: func(x, y int, button walk.MouseButton) {
											if button != walk.LeftButton || codeEdit == nil || lineEdit == nil {
												return
											}
											var pt win.POINT
											pt.X = int32(x)
											pt.Y = int32(y)
											r := lineEdit.SendMessage(win.EM_CHARFROMPOS, 0, uintptr(unsafe.Pointer(&pt)))
											cp := int(win.LOWORD(uint32(r)))
											ln := int(lineEdit.SendMessage(win.EM_LINEFROMCHAR, uintptr(cp), 0)) + 1
											if ln >= 1 {
												gotoLine(ln)
											}
										},
									},
								},
							},
							Composite{MinSize: Size{Height: 1}, Background: SolidColorBrush{Color: colBorder}},
							Composite{
								Layout:        VBox{MarginsZero: true, Spacing: 0},
								Background:    SolidColorBrush{Color: colBg},
								MinSize:       Size{Height: 160},
								StretchFactor: 2,
								Children: []Widget{
									Composite{
										AssignTo:   &tabBar,
										Layout:     HBox{Margins: Margins{Left: 8, Right: 8, Top: 3, Bottom: 3}, Spacing: 3},
										Background: SolidColorBrush{Color: colTabBar},
										MinSize:    Size{Height: 26},
										MaxSize:    Size{Height: 26},
										Children:   []Widget{},
									},
									TextEdit{
										AssignTo:           &errEdit,
										ReadOnly:           true,
										VScroll:            true,
										RightToLeftReading: true,
										TextAlignment:      AlignFar,
										TextColor:          colErrText,
										Background:         SolidColorBrush{Color: colPanel},
										Font:               Font{Family: uiFont, PointSize: 11},
										MinSize:            Size{Height: 120},
										StretchFactor:      1,
										OnMouseUp: func(x, y int, button walk.MouseButton) {
											if button == walk.LeftButton {
												jumpFromErrPanel()
											}
										},
									},
									TextEdit{
										AssignTo:           &outEdit,
										ReadOnly:           true,
										VScroll:            true,
										Visible:            false,
										RightToLeftReading: true,
										TextAlignment:      AlignFar,
										TextColor:          colOutText,
										Background:         SolidColorBrush{Color: colPanel},
										Font:               Font{Family: codeFace, PointSize: 12},
										MinSize:            Size{Height: 120},
										StretchFactor:      1,
									},
								},
							},
						},
					},
				},
			},
			Composite{MinSize: Size{Height: 1}, Background: SolidColorBrush{Color: colBorder}},
			// —— סרגל סטטוס ——
			Composite{
				Layout:     HBox{Margins: Margins{Left: 14, Right: 14, Top: 4, Bottom: 4}, Spacing: 12},
				Background: SolidColorBrush{Color: colStatus},
				Children: []Widget{
					Label{
						AssignTo:           &statusLbl,
						Text:               "מוכן",
						TextColor:          colBrand,
						Font:               Font{Family: uiFont, PointSize: 9, Bold: true},
						RightToLeftReading: true,
					},
					HSpacer{},
					Label{AssignTo: &posLbl, Text: "שורה 1, עמודה 1", TextColor: colMuted, RightToLeftReading: true},
					Label{Text: "·", TextColor: colBorder},
					Label{AssignTo: &linesLbl, Text: "1 שורות", TextColor: colMuted, RightToLeftReading: true},
					Label{Text: "·", TextColor: colBorder},
					Label{AssignTo: &modeLbl, Text: "UTF-8 · יוד · 14pt", TextColor: colMuted, RightToLeftReading: true},
				},
			},
		},
	}.Create()
	if err != nil {
		return err
	}

	// כפתורי סרגל — גודל קבוע בפיקסלים (לא CustomWidget הרגיל שמתנפח ל־100×100)
	if toolbar != nil {
		_ = toolbar.SetMinMaxSizePixels(walk.Size{Height: btnH + 4}, walk.Size{Height: btnH + 4})
		mountToolbar := []struct {
			btn *DarkBtn
			tip string
		}{
			{btnRun, "הרץ — מפרש מלא (F5)"},
			{btnVM, "מכונה — הרצה ב־bytecode (F6)"},
			{btnCheck, "בדוק קומפילציה בלי להריץ (F7)"},
			{btnPack, "ארוז ל־EXE בודד — לחיצה כפולה מריצה (Ctrl+Shift+P)"},
			{btnNew, "קובץ חדש (Ctrl+N)"},
			{btnOpen, "פתח קובץ (Ctrl+O)"},
			{btnFolder, "פתח תיקיית פרויקט (Ctrl+Shift+O)"},
			{btnSave, "שמור קובץ (Ctrl+S)"},
			{btnFormat, "סדר קוד — הזחה 2 רווחים (Ctrl+Shift+F)"},
			{btnHighlight, "הדגשת תחביר בדפדפן"},
		}
		// מפריד אחרי ארוז
		for i, m := range mountToolbar {
			if i == 4 {
				if _, e := walk.NewVSeparator(toolbar); e != nil {
					return e
				}
			}
			if err := m.btn.Mount(toolbar, m.tip); err != nil {
				return err
			}
		}
		if _, e := walk.NewHSpacer(toolbar); e != nil {
			return e
		}
		toolbar.RequestLayout()
	}
	if tabBar != nil {
		_ = tabBar.SetMinMaxSizePixels(walk.Size{Height: btnH + 4}, walk.Size{Height: btnH + 4})
		if err := tabErrBtn.Mount(tabBar, "לשונית שגיאות — תחביר וריצה"); err != nil {
			return err
		}
		if err := tabOutBtn.Mount(tabBar, "לשונית פלט — פלט הדפס"); err != nil {
			return err
		}
		if _, e := walk.NewHSpacer(tabBar); e != nil {
			return e
		}
		if hint, e := walk.NewLabel(tabBar); e == nil {
			_ = hint.SetText("לחיצה על שגיאה ← מעבר לשורה")
			hint.SetTextColor(colMuted)
			_ = hint.SetRightToLeftReading(true)
		}
		tabBar.RequestLayout()
	}

	var cerr error
	codeEdit, cerr = NewCodeEdit(codeHost)
	if cerr != nil {
		return fmt.Errorf("עורך קוד: %w", cerr)
	}
	codeEdit.TextChanged().Attach(func() {
		dirty = true
		updateTitle()
		n := codeEdit.LineCount()
		if n != lastGutterLines {
			lastGutterLines = n
			updateLineNumbers()
		} else if updateCaretStatus != nil {
			updateCaretStatus()
		}
	})
	codeEdit.onZoom = applyCodeZoom
	codeEdit.onSelChange = updateCaretStatus
	fixCodeEdit(codeEdit)
	fixGutterEdit(lineEdit)
	// Escape ברמת החלון — אם העורך לא קיבל את המקש
	if mw != nil {
		mw.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyEscape && codeEdit != nil && codeEdit.ac != nil && codeEdit.ac.visible {
				codeEdit.ac.hide()
			}
		})
	}
	// codeHost נשאר RTL: gutter מימין ליד תחילת השורה (יישור ימין)
	disableWordWrap(lineEdit)
	fixHebrewEdit(errEdit)
	fixHebrewEdit(outEdit)
	// Tab רק להזחת קוד בעורך — לא מעבר לפאנל שגיאות/פלט
	clearTabStop(lineEdit)
	clearTabStop(errEdit)
	clearTabStop(outEdit)

	preferAppDarkMode()
	styleEditorPane(codeEdit)
	styleEditorPane(lineEdit)
	styleEditorPane(errEdit)
	styleEditorPane(outEdit)

	if mw != nil {
		applyDarkScrollbars(mw.Handle())
		var useDark int32 = 1
		procDwmSetWindowAttr.Call(
			uintptr(mw.Handle()),
			dwmwaUseImmersiveDarkMode,
			uintptr(unsafe.Pointer(&useDark)),
			unsafe.Sizeof(useDark),
		)
	}

	for _, w := range []walk.Window{fileLbl, statusLbl, posLbl, linesLbl, modeLbl} {
		if w != nil {
			fixHebrewEdit(w)
		}
	}
	codeHost.RequestLayout()

	if treeSplit != nil {
		clearLayoutRTL := func(hwnd win.HWND) {
			if hwnd == 0 {
				return
			}
			ex := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
			win.SetWindowLong(hwnd, win.GWL_EXSTYLE, ex&^win.WS_EX_LAYOUTRTL)
		}
		// מבטל RTL על ה־splitter וילדיו — גרירה ב־walk נשברת עם LAYOUTRTL
		clearLayoutRTL(treeSplit.Handle())
		for i := 0; i < treeSplit.Children().Len(); i++ {
			clearLayoutRTL(treeSplit.Children().At(i).Handle())
		}
		if treePane != nil {
			// מינימום בפיקסלים אמיתיים; בלי MaxSize — כדי לא לחסום גרירה
			_ = treePane.SetMinMaxSizePixels(walk.Size{Width: 120}, walk.Size{})
		}
		// בלי SetBoundsPixels / SetFixed — הם נלחמו ב־layout ובגרירה.
		// StretchFactor 1:4 ≈ 20%; גרירת ה־handle (8px) מעדכנת item.size.
	}
	if treeView != nil {
		applyDarkScrollbars(treeView.Handle())
		styleEditorPane(treeView)
	}
	refreshProjectUI()
	// אם נפתח קובץ מה־CLI — תיקיית האב כפרויקט
	if currentPath != "" {
		if abs, err := filepath.Abs(currentPath); err == nil {
			projectRoot = filepath.Dir(abs)
			treeModel.SetRoot(projectRoot)
			refreshProjectUI()
		}
	}

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !dirty {
			return
		}
		r := walk.MsgBox(mw, "עורך יוד",
			"יש שינויים שלא נשמרו.\nלשמור לפני יציאה?",
			walk.MsgBoxYesNoCancel|walk.MsgBoxIconWarning)
		switch r {
		case walk.DlgCmdCancel:
			*canceled = true
		case walk.DlgCmdYes:
			if err := saveFile(false); err != nil {
				walk.MsgBox(mw, "שגיאה", err.Error(), walk.MsgBoxIconError)
				*canceled = true
			}
		}
	})

	updateTitle()
	setPanelText(errEdit, "אין שגיאות")
	setPanelText(outEdit, "(אין פלט עדיין — לחצו «הרץ» או F5)")
	errLines = []int{0}
	showErrorsTab()

	if currentPath != "" {
		if err := loadFile(currentPath); err != nil {
			setStatus("לא נפתח · " + err.Error())
		}
	} else {
		_ = codeEdit.SetText(welcomeTemplate)
		dirty = false
		updateTitle()
		setStatus("מוכן · F5 להרצה")
		applyCodeZoom(codeFontSize)
	}
	updateLineNumbers()
	updateCaretStatus()

	stopSync := make(chan struct{})
	go func() {
		t := time.NewTicker(50 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stopSync:
				return
			case <-t.C:
				mw.Synchronize(func() {
					syncLineScroll()
				})
			}
		}
	}()

	mw.Run()
	close(stopSync)
	return nil
}

// saveFileRef — עזר ל־confirmDiscard
func saveFileRef(mw *walk.MainWindow, currentPath *string, codeEdit *CodeEdit, dirty *bool, updateTitle func(), setStatus func(string)) error {
	p := *currentPath
	if p == "" {
		dlg := new(walk.FileDialog)
		dlg.Title = "שמירת קובץ יוד"
		dlg.Filter = "קבצי יוד (*.יוד)|*.יוד|כל הקבצים (*.*)|*.*"
		dlg.FilePath = "תוכנית.יוד"
		ok, err := dlg.ShowSave(mw)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("השמירה בוטלה")
		}
		p = dlg.FilePath
		if filepath.Ext(p) == "" {
			p += ".יוד"
		}
	}
	if err := os.WriteFile(p, []byte(codeEdit.Text()), 0644); err != nil {
		return err
	}
	*currentPath = p
	*dirty = false
	updateTitle()
	setStatus("נשמר · " + filepath.Base(p))
	return nil
}

func fixGutterEdit(w walk.Window) {
	if w == nil {
		return
	}
	hwnd := w.Handle()
	ex := uint32(win.GetWindowLong(hwnd, win.GWL_EXSTYLE))
	ex |= win.WS_EX_NOINHERITLAYOUT
	ex &^= win.WS_EX_LAYOUTRTL | win.WS_EX_RTLREADING
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(ex))
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
}

// fixCodeEdit — קוד בעברית: קריאה RTL (בלוקים עם סוף, בלי {})
func fixCodeEdit(w walk.Window) {
	if w == nil {
		return
	}
	hwnd := w.Handle()
	ex := uint32(win.GetWindowLong(hwnd, win.GWL_EXSTYLE))
	ex |= win.WS_EX_NOINHERITLAYOUT | win.WS_EX_RTLREADING
	ex &^= win.WS_EX_LAYOUTRTL
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(ex))
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
}

func disableWordWrap(te *walk.TextEdit) {
	if te == nil {
		return
	}
	hwnd := te.Handle()
	style := uint32(win.GetWindowLong(hwnd, win.GWL_STYLE))
	style |= win.ES_AUTOHSCROLL
	win.SetWindowLong(hwnd, win.GWL_STYLE, int32(style))
}

func fixHebrewEdit(w walk.Window) {
	if w == nil {
		return
	}
	hwnd := w.Handle()
	ex := uint32(win.GetWindowLong(hwnd, win.GWL_EXSTYLE))
	ex |= win.WS_EX_NOINHERITLAYOUT | win.WS_EX_RTLREADING
	ex &^= win.WS_EX_LAYOUTRTL
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(ex))
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED)
	win.InvalidateRect(hwnd, nil, true)
}

func clearTabStop(w walk.Window) {
	if w == nil {
		return
	}
	hwnd := w.Handle()
	style := uint32(win.GetWindowLong(hwnd, win.GWL_STYLE))
	if style&win.WS_TABSTOP == 0 {
		return
	}
	style &^= win.WS_TABSTOP
	win.SetWindowLong(hwnd, win.GWL_STYLE, int32(style))
}

func setPanelText(edit *walk.TextEdit, s string) {
	if edit == nil {
		return
	}
	edit.SetText(rtlDisplay(s))
}

func rtlDisplay(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return "\u200F"
	}
	parts := strings.Split(s, "\n")
	for i, p := range parts {
		if !strings.HasPrefix(p, "\u200F") {
			parts[i] = "\u200F" + p
		}
	}
	return strings.Join(parts, "\r\n")
}

func extractLine(msg string) int {
	m := lineErrRe.FindStringSubmatch(msg)
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// formatErrorPanel מציג כל שגיאה בשורה נפרדת: «שגיאה בשורה N — הסבר»
func formatErrorPanel(msgs []string) (string, []int) {
	var lines []string
	var lineNums []int
	for _, raw := range msgs {
		for _, part := range strings.Split(raw, "\n") {
			part = strings.TrimSpace(part)
			if part == "" || part == "שגיאות תחביר:" {
				continue
			}
			display, ln := formatOneError(part)
			lines = append(lines, display)
			lineNums = append(lineNums, ln)
		}
	}
	if len(lines) == 0 {
		return "אין שגיאות", []int{0}
	}
	return strings.Join(lines, "\r\n"), lineNums
}

func formatOneError(raw string) (string, int) {
	s := strings.TrimSpace(raw)
	s = strings.TrimLeft(s, " \t•-")
	s = errLeadRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	ln := extractLine(s)
	explanation := s
	if pref := errLinePrefRe.FindString(s); pref != "" {
		explanation = strings.TrimSpace(s[len(pref):])
	}
	if explanation == "" {
		explanation = s
	}
	if ln > 0 {
		return fmt.Sprintf("שגיאה בשורה %d — %s", ln, explanation), ln
	}
	return fmt.Sprintf("שגיאה — %s", explanation), 0
}

func lineIndexAt(src string, pos int) int {
	if pos < 0 {
		pos = 0
	}
	runes := []rune(src)
	if pos > len(runes) {
		pos = len(runes)
	}
	line := 0
	for i := 0; i < pos; i++ {
		if runes[i] == '\n' {
			line++
		}
	}
	return line
}

func lineOffsets(src string, line int) (start, end int) {
	if line < 1 {
		return -1, -1
	}
	runes := []rune(src)
	curLine := 1
	startIdx := 0
	for i, r := range runes {
		if curLine == line {
			startIdx = i
			break
		}
		if r == '\n' {
			curLine++
			if curLine == line {
				startIdx = i + 1
				break
			}
		}
		if i == len(runes)-1 && curLine < line {
			return -1, -1
		}
	}
	if curLine != line {
		if line != 1 {
			return -1, -1
		}
		startIdx = 0
	}
	endIdx := len(runes)
	for i := startIdx; i < len(runes); i++ {
		if runes[i] == '\n' {
			endIdx = i
			break
		}
	}
	return startIdx, endIdx
}
