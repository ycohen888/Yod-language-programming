package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/compiler"
	"yod/internal/console"
	"yod/internal/editor"
	"yod/internal/evaluator"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/pack"
	"yod/internal/parser"
	"yod/internal/vm"
)

//go:generate go run ../../tools/mkico.go ../../assets/yod-icon-source.png ../../assets/yod.ico
//go:generate rsrc -arch amd64 -ico ../../assets/yod.ico -manifest yod.exe.manifest -o rsrc_windows_amd64.syso

const version = "0.50.6"

func main() {
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
		// הרץ --מכונה קובץ / הרץ קובץ
		path := os.Args[2]
		preferVM := false
		argStart := 3
		if path == "--מכונה" || path == "--vm" {
			preferVM = true
			if len(os.Args) < 4 {
				console.Fprintln(os.Stderr, "שימוש: יוד הרץ --מכונה תוכנית.יוד")
				os.Exit(1)
			}
			path = os.Args[3]
			argStart = 4
		}
		if argStart < len(os.Args) {
			object.ProgramArgs = append([]string{}, os.Args[argStart:]...)
		} else {
			object.ProgramArgs = nil
		}
		var err error
		if preferVM {
			err = runSmart(path, true)
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
	default:
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
	console.Println("  yod                    (פותח את העורך)")
	console.Println("  yod עורך [תוכנית.יוד]")
	console.Println("  yod הרץ תוכנית.יוד")
	console.Println("  yod הרץ --מכונה תוכנית.יוד")
	console.Println("  yod מכונה תוכנית.יוד   (bytecode VM)")
	console.Println("  yod תוכנית.יוד         (מכונה → נפילה למפרש)")
	console.Println("  yod ארוז תוכנית.יוד [יעד.exe]     → EXE בלי חלון CMD")
	console.Println("  yod ארוז תוכנית.יוד --קונסול      → EXE עם חלון CMD")
	console.Println("  yod ארוז תוכנית.יוד --תיקייה [יעד] → תיקיית הפצה")
	console.Println("  yod גרסה")
	console.Println("  yod עזרה")
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
		return runFile(path)
	}
	return err
}

func runFile(path string) error {
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
	env := evaluator.NewGlobalEnv(base)
	result := evaluator.Eval(program, env)
	if result != nil && result.Type() == object.ErrorObj {
		return fmt.Errorf("%s", result.Inspect())
	}
	return nil
}

func runVM(path string) error {
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
