package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/console"
	"yod/internal/evaluator"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/parser"
)

func runREPL() {
	console.Println("יוד קונסול — הקלידו קוד. יציאה = יציאה / exit")
	console.Println()
	env := evaluator.NewGlobalEnv(".")
	sc := bufio.NewScanner(os.Stdin)
	var buf strings.Builder
	depth := 0
	for {
		if buf.Len() == 0 {
			console.Print("יוד> ")
		} else {
			console.Print("...> ")
		}
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if buf.Len() == 0 && (trim == "יציאה" || trim == "exit" || trim == "quit") {
			return
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
		depth += strings.Count(line, "פונקציה") + strings.Count(line, "אם") + strings.Count(line, "כל_עוד") +
			strings.Count(line, "עבור") + strings.Count(line, "מחלקה") + strings.Count(line, "נסה") +
			strings.Count(line, "בחר")
		depth -= strings.Count(line, "סוף")
		if depth > 0 {
			continue
		}
		depth = 0
		src := buf.String()
		buf.Reset()
		if strings.TrimSpace(src) == "" {
			continue
		}
		l := lexer.New(src)
		p := parser.New(l)
		prog := p.ParseProgram()
		if errs := p.Errors(); len(errs) > 0 {
			for _, e := range errs {
				console.Fprintln(os.Stderr, e)
			}
			continue
		}
		result := evaluator.Eval(prog, env)
		if result != nil && result.Type() == object.ErrorObj {
			console.Fprintln(os.Stderr, result.Inspect())
			continue
		}
		if result != nil && result.Type() != object.NullObj {
			console.Println(result.Inspect())
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func pkgAdd(src string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("לא נמצא: %v", err)
	}
	base := filepath.Base(src)
	if !fi.IsDir() {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	destRoot := filepath.Join(".", ".יוד_חבילות")
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return err
	}
	dest := filepath.Join(destRoot, base)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if fi.IsDir() {
		return copyDir(src, dest)
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	name := filepath.Base(src)
	return os.WriteFile(filepath.Join(dest, name), data, 0644)
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

