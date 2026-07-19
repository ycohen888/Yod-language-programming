package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"yod/internal/ast"
	"yod/internal/lexer"
	"yod/internal/parser"
	"yod/internal/project"
	"yod/internal/stdlib"
	"yod/internal/vfs"
)

// BundleFiles — תוצאת איסוף לחבילת YODBUND1.
type BundleFiles struct {
	Files map[string][]byte // מפתחות יחסיים עם /
	Entry string            // קובץ כניסה יחסי
	Root  string            // שורש הפרויקט
}

// CollectBundle אוסף מודולי כלול/יבא + עיצוב אחות + נכסי פרויקט.
func CollectBundle(entryPath string) (*BundleFiles, error) {
	entryPath = filepath.Clean(entryPath)
	root := project.FindProjectRoot(entryPath)
	if root == "" {
		root = filepath.Dir(entryPath)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	entryAbs, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, err
	}

	out := &BundleFiles{
		Files: map[string][]byte{},
		Root:  root,
	}
	entryRel, err := filepath.Rel(root, entryAbs)
	if err != nil {
		return nil, err
	}
	out.Entry = vfs.Normalize(entryRel)

	siblingDesign := filepath.Join(filepath.Dir(root), "עיצוב")
	if st, err := os.Stat(siblingDesign); err != nil || !st.IsDir() {
		siblingDesign = ""
	}

	builtin := map[string]bool{}
	for _, n := range stdlib.BuiltinNames() {
		builtin[n] = true
		builtin[strings.ToLower(n)] = true
	}

	type item struct {
		abs string
		key string
	}
	seen := map[string]bool{}
	var queue []item

	enqueue := func(abs string) {
		abs = filepath.Clean(abs)
		if abs == "" {
			return
		}
		key, ok := virtualKey(root, siblingDesign, abs)
		if !ok {
			return
		}
		if seen[key] {
			return
		}
		seen[key] = true
		queue = append(queue, item{abs: abs, key: key})
	}

	enqueue(entryAbs)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		data, err := os.ReadFile(cur.abs)
		if err != nil {
			return nil, fmt.Errorf("קריאת %s: %w", cur.abs, err)
		}
		if project.IsYodSource(cur.abs) {
			data = rewriteSiblingIncludes(data)
			out.Files[cur.key] = data
			for _, dep := range extractUserDeps(string(data), builtin) {
				resolved := resolveCollectPath(filepath.Dir(cur.abs), dep, root, siblingDesign)
				if resolved == "" {
					continue
				}
				enqueue(resolved)
			}
		} else {
			out.Files[cur.key] = data
		}
	}

	// נכסי פרויקט (מדריך, איקונים…)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			if info != nil && info.IsDir() && path != root && isRuntimeDataDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if isPackSkipSource(info.Name()) {
			return nil
		}
		if !isProjectAsset(path, rel) {
			return nil
		}
		key := vfs.Normalize(rel)
		if seen[key] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		seen[key] = true
		out.Files[key] = data
		return nil
	})

	// כל ספריית עיצוב אחות (HTML/CSS/JS + מודולים)
	if siblingDesign != "" {
		_ = filepath.Walk(siblingDesign, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				if path != siblingDesign && isRuntimeDataDir(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			rel, relErr := filepath.Rel(siblingDesign, path)
			if relErr != nil {
				return nil
			}
			key := vfs.Normalize(filepath.Join("עיצוב", rel))
			if seen[key] {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if project.IsYodSource(path) {
				data = rewriteSiblingIncludes(data)
			}
			seen[key] = true
			out.Files[key] = data
			return nil
		})
	}

	if _, ok := out.Files[out.Entry]; !ok {
		return nil, fmt.Errorf("קובץ כניסה לא נאסף: %s", out.Entry)
	}
	return out, nil
}

func virtualKey(root, siblingDesign, abs string) (string, bool) {
	if rel, err := filepath.Rel(root, abs); err == nil {
		relSlash := filepath.ToSlash(rel)
		if !strings.HasPrefix(relSlash, "../") && relSlash != ".." {
			return vfs.Normalize(rel), true
		}
	}
	if siblingDesign != "" {
		if rel, err := filepath.Rel(siblingDesign, abs); err == nil {
			relSlash := filepath.ToSlash(rel)
			if !strings.HasPrefix(relSlash, "../") && relSlash != ".." {
				return vfs.Normalize(filepath.Join("עיצוב", rel)), true
			}
		}
	}
	return "", false
}

func resolveCollectPath(fromDir, dep, root, siblingDesign string) string {
	dep = strings.TrimSpace(dep)
	if dep == "" {
		return ""
	}
	// אחרי rewriteSiblingIncludes נתיבי ..\עיצוב כבר עיצוב\ — אבל מקורות בזמן איסוף
	// לפני rewrite עדיין יכולים להצביע על אחות.
	candidates := []string{
		filepath.Clean(filepath.Join(fromDir, dep)),
	}
	if strings.Contains(dep, "עיצוב") || strings.HasPrefix(dep, "..") {
		rewritten := string(rewriteSiblingIncludes([]byte(dep)))
		if rewritten != dep {
			candidates = append(candidates, filepath.Clean(filepath.Join(fromDir, rewritten)))
			if siblingDesign != "" {
				// עיצוב\x מתוך שורש הפרויקט
				candidates = append(candidates, filepath.Clean(filepath.Join(root, rewritten)))
				rest := strings.TrimPrefix(strings.ReplaceAll(rewritten, "/", "\\"), "עיצוב\\")
				rest = strings.TrimPrefix(rest, `עיצוב/`)
				candidates = append(candidates, filepath.Clean(filepath.Join(siblingDesign, rest)))
			}
		}
		if siblingDesign != "" && (strings.Contains(dep, `..\עיצוב`) || strings.Contains(dep, `../עיצוב`)) {
			rest := dep
			for _, p := range []string{`..\עיצוב\`, `../עיצוב/`, `..\עיצוב/`, `../עיצוב\`} {
				if i := strings.Index(rest, p); i >= 0 {
					rest = rest[i+len(p):]
					break
				}
			}
			candidates = append(candidates, filepath.Clean(filepath.Join(siblingDesign, rest)))
		}
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
		// ניסיון עם .יוד
		if !project.IsYodSource(c) {
			for _, ext := range []string{".יוד", ".yod"} {
				try := c + ext
				if st, err := os.Stat(try); err == nil && !st.IsDir() {
					return try
				}
			}
		}
	}
	return ""
}

func extractUserDeps(src string, builtin map[string]bool) []string {
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()
	var deps []string
	seen := map[string]bool{}
	add := func(path string) {
		if path == "" || builtin[path] || builtin[strings.ToLower(path)] || seen[path] {
			return
		}
		seen[path] = true
		deps = append(deps, path)
	}
	var walkStmt func(ast.Statement)
	walkStmt = func(s ast.Statement) {
		if s == nil {
			return
		}
		switch node := s.(type) {
		case *ast.IncludeStatement:
			add(node.Path)
		case *ast.ImportStatement:
			add(node.Path)
		case *ast.BlockStatement:
			for _, st := range node.Statements {
				walkStmt(st)
			}
		case *ast.IfStatement:
			walkStmt(node.Consequence)
			walkStmt(node.Alternative)
		case *ast.WhileStatement:
			walkStmt(node.Body)
		case *ast.ForInStatement:
			walkStmt(node.Body)
		case *ast.ForRangeStatement:
			walkStmt(node.Body)
		case *ast.FunctionLiteral:
			walkStmt(node.Body)
		case *ast.ClassStatement:
			walkStmt(node.Body)
		case *ast.TryStatement:
			walkStmt(node.Body)
			walkStmt(node.CatchBody)
		case *ast.SwitchStatement:
			for _, c := range node.Cases {
				walkStmt(c.Body)
			}
			walkStmt(node.Default)
		case *ast.ExpressionStatement:
			if fl, ok := node.Expr.(*ast.FunctionLiteral); ok {
				walkStmt(fl.Body)
			}
		case *ast.ExportStatement:
			walkStmt(node.Stmt)
		}
	}
	for _, s := range program.Statements {
		walkStmt(s)
	}
	return deps
}

// ListBundlePaths מחזיר רשימה ממוינת של קבצים שייכנסו לחבילה (לבדיקה / --רשימה).
func ListBundlePaths(entryPath string) ([]string, string, error) {
	b, err := CollectBundle(entryPath)
	if err != nil {
		return nil, "", err
	}
	keys := make([]string, 0, len(b.Files))
	for k := range b.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, b.Entry, nil
}
