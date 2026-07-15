package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/compiler"
	"yod/internal/console"
	"yod/internal/editor"
	"yod/internal/evaluator"
	"yod/internal/format"
	"yod/internal/lexer"
	"yod/internal/lint"
	"yod/internal/object"
	"yod/internal/pack"
	"yod/internal/parser"
	"yod/internal/project"
	ver "yod/internal/version"
	"yod/internal/vm"
)

//go:generate go run ../../tools/mkico.go ../../assets/yod-icon-source.png ../../assets/yod.ico
//go:generate rsrc -arch amd64 -ico ../../assets/yod.ico -manifest yod.exe.manifest -o rsrc_windows_amd64.syso

const version = ver.String

func main() {
	// מסתירים CMD מוקדם כשפותחים את העורך — מצמצם הבהוב לפני Init
	if len(os.Args) < 2 || isEditorArg(os.Args[1]) {
		console.HideIfOwned()
	}
	console.Init()

	if exePath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		if src, ok, err := pack.ReadEmbedded(exePath); err == nil && ok {
			if len(os.Args) > 1 {
				object.ProgramArgs = append([]string{}, os.Args[1:]...)
			} else {
				object.ProgramArgs = nil
			}
			base := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
			virtual := filepath.Join(filepath.Dir(exePath), base+".יוד")
			if err := runSource(src, virtual); err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			return
		}
	}

	// לחיצה כפולה / בלי ארגומנטים → העורך
	if len(os.Args) < 2 {
		openEditor("")
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "גרסה", "version", "-v", "--version":
		console.Printf("יוד %s\n", version)
	case "עזרה", "help", "-h", "--help":
		printHelp()
	case "הרץ", "run":
		if len(os.Args) < 3 {
			console.Fprintln(os.Stderr, "שגיאה: חסר נתיב לקובץ .יוד")
			console.Fprintln(os.Stderr, "שימוש: יוד הרץ תוכנית.יוד")
			os.Exit(1)
		}
		path := ""
		preferVM := false
		checkTypes := false
		var progArgs []string
		for i := 2; i < len(os.Args); i++ {
			a := os.Args[i]
			switch a {
			case "--מכונה", "--vm":
				preferVM = true
			case "--בדוק_טיפוסים", "--check-types":
				checkTypes = true
			default:
				if path == "" {
					path = a
				} else {
					progArgs = append(progArgs, a)
				}
			}
		}
		if path == "" {
			console.Fprintln(os.Stderr, "שימוש: יוד הרץ [--מכונה] [--בדוק_טיפוסים] תוכנית.יוד")
			os.Exit(1)
		}
		object.ProgramArgs = progArgs
		evaluator.CheckTypes = checkTypes
		var err error
		if preferVM {
			err = runSmart(path, true)
		} else if checkTypes {
			// בדיקת טיפוסים במפרש בלבד
			err = runFile(path)
		} else {
			err = runFile(path)
		}
		if err != nil {
			console.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "ארוז", "pack":
		if len(os.Args) < 3 {
			console.Fprintln(os.Stderr, "שגיאה: חסר נתיב לקובץ .יוד")
			console.Fprintln(os.Stderr, "שימוש: יוד ארוז תוכנית.יוד [יעד.exe]")
			console.Fprintln(os.Stderr, "       יוד ארוז תוכנית.יוד --קונסול   (עם חלון CMD)")
			console.Fprintln(os.Stderr, "       יוד ארוז תוכנית.יוד --תיקייה [תיקייה]")
			os.Exit(1)
		}
		var rest []string
		if len(os.Args) > 3 {
			rest = os.Args[3:]
		}
		if err := packApp(os.Args[2], rest); err != nil {
			console.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "מכונה", "vm":
		if len(os.Args) < 3 {
			console.Fprintln(os.Stderr, "שגיאה: חסר נתיב לקובץ .יוד")
			console.Fprintln(os.Stderr, "שימוש: יוד מכונה תוכנית.יוד")
			os.Exit(1)
		}
		if len(os.Args) > 3 {
			object.ProgramArgs = append([]string{}, os.Args[3:]...)
		}
		if err := runVM(os.Args[2]); err != nil {
			console.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "עורך", "editor":
		path := ""
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		openEditor(path)
	case "בדוק", "check", "lint":
		if len(os.Args) < 3 {
			console.Fprintln(os.Stderr, "שגיאה: חסר נתיב לקובץ או תיקייה")
			console.Fprintln(os.Stderr, "שימוש: יוד בדוק תוכנית.יוד")
			console.Fprintln(os.Stderr, "       יוד בדוק תיקיית_פרויקט")
			os.Exit(1)
		}
		issues, err := lint.CheckPath(os.Args[2])
		if err != nil {
			console.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if len(issues) == 0 {
			console.Println("אין ממצאים.")
			return
		}
		for _, iss := range issues {
			console.Println(iss.String())
		}
		os.Exit(1)
	case "סדר", "format":
		writeBack := false
		path := ""
		for _, a := range os.Args[2:] {
			switch a {
			case "-w", "--write":
				writeBack = true
			default:
				if path == "" {
					path = a
				}
			}
		}
		var src []byte
		var err error
		if path == "" || path == "-" {
			src, err = io.ReadAll(os.Stdin)
			if err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		} else {
			src, err = os.ReadFile(path)
			if err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
		out := format.Source(string(src))
		if writeBack {
			if path == "" || path == "-" {
				console.Fprintln(os.Stderr, "שימוש: יוד סדר [-w] תוכנית.יוד")
				os.Exit(1)
			}
			if err := os.WriteFile(path, []byte(out), 0644); err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
		// stdout גולמי — בלי עיבוד קונסול — כדי שה־IDE יקבל את הקוד כמו שהוא
		_, _ = io.WriteString(os.Stdout, out)
		if !strings.HasSuffix(out, "\n") {
			_, _ = io.WriteString(os.Stdout, "\n")
		}
	case "קונסול", "שלד", "repl":
		runREPL()
	case "חבילה", "pkg", "package":
		if len(os.Args) < 3 {
			console.Fprintln(os.Stderr, "שימוש: יוד חבילה הוסף נתיב")
			os.Exit(1)
		}
		sub := os.Args[2]
		switch sub {
		case "הוסף", "add":
			if len(os.Args) < 4 {
				console.Fprintln(os.Stderr, "שימוש: יוד חבילה הוסף נתיב_לקובץ_או_תיקייה")
				os.Exit(1)
			}
			if err := pkgAdd(os.Args[3]); err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			console.Println("החבילה נוספה ל־.יוד_חבילות")
		default:
			console.Fprintf(os.Stderr, "תת־פקודה לא מוכרת: %s\n", sub)
			console.Fprintln(os.Stderr, "שימוש: יוד חבילה הוסף נתיב")
			os.Exit(1)
		}
	default:
		// תיקיית פרויקט או קובץ .יוד
		if fi, err := os.Stat(cmd); err == nil && fi.IsDir() {
			if len(os.Args) > 2 {
				object.ProgramArgs = append([]string{}, os.Args[2:]...)
			}
			if err := runSmart(cmd, false); err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			return
		}
		if strings.HasSuffix(cmd, ".יוד") || strings.HasSuffix(cmd, ".yod") {
			if len(os.Args) > 2 {
				object.ProgramArgs = append([]string{}, os.Args[2:]...)
			}
			// קובץ ישירות: מנסה מכונה, נופל למפרש אם לא נתמך
			if err := runSmart(cmd, false); err != nil {
				console.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			return
		}
		console.Fprintf(os.Stderr, "פקודה לא מוכרת: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func isEditorArg(a string) bool {
	switch a {
	case "עורך", "editor":
		return true
	default:
		return false
	}
}

func openEditor(path string) {
	console.HideIfOwned()
	if err := editor.Run(path); err != nil {
		console.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func printHelp() {
	console.Println("יוד — שפת תכנות בעברית")
	console.Println()
	console.Println("שימוש:")
	console.Println("  yod                    (פותח את העורך Electron)")
	console.Println("  yod עורך [תוכנית.יוד]")
	console.Println("  yod הרץ תוכנית.יוד")
	console.Println("  yod הרץ תיקיית_פרויקט   (מריץ התחל.יוד)")
	console.Println("  yod הרץ --מכונה תוכנית.יוד")
	console.Println("  yod מכונה תוכנית.יוד   (bytecode VM)")
	console.Println("  yod תוכנית.יוד         (מכונה → נפילה למפרש)")
	console.Println("  yod ארוז תוכנית.יוד [יעד.exe]     → EXE בלי חלון CMD")
	console.Println("  yod ארוז תוכנית.יוד --קונסול      → EXE עם חלון CMD")
	console.Println("  yod ארוז תוכנית.יוד --תיקייה [יעד] → תיקיית הפצה")
	console.Println("  yod בדוק תוכנית.יוד   (סגנון ותחביר)")
	console.Println("  yod סדר תוכנית.יוד    (מסדר קוד ל־stdout; -w כותב לקובץ)")
	console.Println("  yod הרץ --בדוק_טיפוסים תוכנית.יוד")
	console.Println("  yod קונסול            (REPL)")
	console.Println("  yod חבילה הוסף נתיב   (עותק ל־.יוד_חבילות)")
	console.Println("  yod גרסה")
	console.Println("  yod עזרה")
	console.Println()
	console.Println("עורך: אחרי build של yod-ide/ — Electron+CodeMirror (npm install && npm run build)")
	console.Println("פרויקט: הקובץ הראשי הוא תמיד התחל.יוד — ממנו כוללים קבצים אחרים עם כלול/יבא.")
}

// runSmart — מנסה VM; אם הקומפילציה נכשלת בגלל צומת לא נתמך — נופל למפרש
func runSmart(path string, forceVM bool) error {
	err := runVM(path)
	if err == nil {
		return nil
	}
	msg := err.Error()
	if forceVM {
		return err
	}
	if strings.Contains(msg, "לא נתמך") || strings.Contains(msg, "שגיאת קומפילציה") {
		if !forceVM {
			console.Fprintf(os.Stderr, "הערה: המכונה לא תומכת בתכונה זו (%s) — מריץ במפרש.\n", shortUnsupported(msg))
		}
		return runFile(path)
	}
	return err
}

func shortUnsupported(msg string) string {
	msg = strings.TrimSpace(msg)
	if i := strings.Index(msg, "\n"); i > 0 {
		msg = msg[:i]
	}
	if len([]rune(msg)) > 80 {
		r := []rune(msg)
		return string(r[:80]) + "…"
	}
	return msg
}

func runFile(path string) error {
	path, err := project.ResolveEntry(path)
	if err != nil {
		return fmt.Errorf("שגיאה: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("שגיאה: לא הצלחתי לקרוא את הקובץ %s: %v", path, err)
	}

	ext := filepath.Ext(path)
	if ext != ".יוד" && ext != ".yod" {
		console.Fprintf(os.Stderr, "אזהרה: הסיומת המומלצת היא .יוד (קיבלתי %q)\n", ext)
	}
	return runSource(string(data), path)
}

func runSource(source, pathForBase string) error {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		var b strings.Builder
		b.WriteString("שגיאות תחביר:\n")
		for _, e := range errs {
			b.WriteString("  ")
			b.WriteString(e)
			b.WriteByte('\n')
		}
		return fmt.Errorf("%s", b.String())
	}

	base := filepath.Dir(pathForBase)
	if base == "" {
		base = "."
	}
	evaluator.PushSourceFile(pathForBase)
	defer evaluator.PopSourceFile()
	env := evaluator.NewGlobalEnv(base)
	result := evaluator.Eval(program, env)
	if result != nil && result.Type() == object.ErrorObj {
		return fmt.Errorf("%s", result.Inspect())
	}
	return nil
}

func runVM(path string) error {
	path, err := project.ResolveEntry(path)
	if err != nil {
		return fmt.Errorf("שגיאה: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("שגיאה: לא הצלחתי לקרוא את הקובץ %s: %v", path, err)
	}
	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		var b strings.Builder
		b.WriteString("שגיאות תחביר:\n")
		for _, e := range errs {
			b.WriteString("  ")
			b.WriteString(e)
			b.WriteByte('\n')
		}
		return fmt.Errorf("%s", b.String())
	}
	comp := compiler.New(filepath.Dir(path))
	if err := comp.Compile(program); err != nil {
		return fmt.Errorf("שגיאת קומפילציה למכונה: %v\n(טיפ: yod הרץ תומך בכל השפה; yod מכונה — ליבה + מחלקות + כלול)", err)
	}
	machine := vm.New(comp.Bytecode())
	object.InvokeCallable = func(fn object.Object, args []object.Object) object.Object {
		switch f := fn.(type) {
		case *object.Function:
			if object.InvokeFunction != nil {
				return object.InvokeFunction(f, args)
			}
			return &object.Error{Message: "אין מפרש לפונקציה"}
		case *object.Closure, *object.CompiledFunction:
			return machine.CallCallable(fn, args)
		default:
			return &object.Error{Message: "בלחיצה מצפה לפונקציה"}
		}
	}
	if err := machine.Run(); err != nil {
		return fmt.Errorf("שגיאת מכונה: %v", err)
	}
	return nil
}
