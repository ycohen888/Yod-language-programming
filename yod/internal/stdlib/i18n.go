package stdlib

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"yod/internal/object"
	"yod/internal/vfs"
)

// I18nUIHook — נקרא אחרי שינוי שפה/קטלוג כדי לעדכן ממשק (עיצוב וכו').
type I18nUIHook func(lang, gender, dir string, catalog map[string]string)

var (
	i18nMu       sync.RWMutex
	i18nLang     = "he"
	i18nGender   = ""
	i18nCatalogs = map[string]*i18nCatalog{} // lang → catalog
	i18nUIHook    I18nUIHook
	i18nYodHooks  []object.Object // פונקציות ביוד: (מילון_מידע) — ניתן לרשום כמה
)

type i18nCatalog struct {
	msgs   map[string]*i18nMsg
	plural pluralFunc
	header string
}

// SetI18nUIHook רושם callback לרענון ממשק.
func SetI18nUIHook(fn I18nUIHook) {
	i18nMu.Lock()
	i18nUIHook = fn
	i18nMu.Unlock()
}

// I18nText — תרגום פשוט לשימוש מ־Go (חלונות וכו').
func I18nText(s string) string {
	return i18nLookup("", s, -1, nil)
}

// I18nDirection מחזיר rtl או ltr.
func I18nDirection() string {
	i18nMu.RLock()
	lang := i18nLang
	i18nMu.RUnlock()
	return i18nDirForLang(lang)
}

func i18nDirForLang(lang string) string {
	base := normalizeLangCode(lang)
	if i := strings.Index(base, "_"); i > 0 {
		base = base[:i]
	}
	switch base {
	case "he", "iw", "ar", "fa", "ur", "yi", "ps":
		return "rtl"
	default:
		return "ltr"
	}
}

func NewI18nModule() *object.Module {
	m := &object.Module{Name: "תרגום", Attrs: map[string]object.Object{}}
	m.Attrs["קבע_שפה"] = &object.Builtin{Fn: i18nSetLang}
	m.Attrs["שפה"] = &object.Builtin{Fn: i18nGetLang}
	m.Attrs["טען"] = &object.Builtin{Fn: i18nLoad}
	m.Attrs["טען_תיקייה"] = &object.Builtin{Fn: i18nLoadDir}
	m.Attrs["רשום"] = &object.Builtin{Fn: i18nRegister}
	m.Attrs["ט"] = &object.Builtin{Fn: i18nT}
	m.Attrs["ט_רבים"] = &object.Builtin{Fn: i18nTN}
	m.Attrs["ט_הקשר"] = &object.Builtin{Fn: i18nTCtx}
	m.Attrs["ט_מגדרי"] = &object.Builtin{Fn: i18nTGender}
	m.Attrs["ט_רבים_הקשר"] = &object.Builtin{Fn: i18nTNCtx}
	m.Attrs["קבע_מגדר"] = &object.Builtin{Fn: i18nSetGender}
	m.Attrs["מגדר"] = &object.Builtin{Fn: i18nGetGender}
	m.Attrs["יש"] = &object.Builtin{Fn: i18nHas}
	m.Attrs["כיוון"] = &object.Builtin{Fn: i18nDir}
	m.Attrs["הוא_rtl"] = &object.Builtin{Fn: i18nIsRTL}
	m.Attrs["שפת_מערכת"] = &object.Builtin{Fn: i18nSystemLang}
	m.Attrs["רענן_ממשק"] = &object.Builtin{Fn: i18nRefreshUI}
	m.Attrs["קטלוג"] = &object.Builtin{Fn: i18nExportCatalog}
	m.Attrs["בעת_שינוי"] = &object.Builtin{Fn: i18nOnChange}
	return m
}

func i18nOnChange(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.בעת_שינוי", 1, args); err != nil {
		return err
	}
	switch args[0].(type) {
	case *object.Function, *object.Builtin, *object.Closure:
		i18nMu.Lock()
		i18nYodHooks = append(i18nYodHooks, args[0])
		i18nMu.Unlock()
	default:
		return errObj("תרגום.בעת_שינוי מצפה לפונקציה")
	}
	return object.Nil
}

func i18nNotifyUI() {
	i18nMu.RLock()
	hook := i18nUIHook
	yodHooks := append([]object.Object{}, i18nYodHooks...)
	lang := i18nLang
	gender := i18nGender
	flat := i18nFlatCatalogLocked(lang)
	i18nMu.RUnlock()
	dir := i18nDirForLang(lang)
	if hook != nil {
		hook(lang, gender, dir, flat)
	}
	if len(yodHooks) > 0 && object.InvokeCallable != nil {
		pairs := map[string]object.Object{}
		for k, v := range flat {
			pairs[k] = &object.String{Value: v}
		}
		info := &object.Hash{Pairs: map[string]object.Object{
			"שפה":   &object.String{Value: lang},
			"מגדר":  &object.String{Value: gender},
			"כיוון": &object.String{Value: dir},
			"מילון": &object.Hash{Pairs: pairs},
		}}
		for _, yodHook := range yodHooks {
			object.InvokeCallable(yodHook, []object.Object{info})
		}
	}
}

func i18nFlatCatalogLocked(lang string) map[string]string {
	lang = normalizeLangCode(lang)
	out := map[string]string{}
	cat := i18nCatalogs[lang]
	if cat == nil {
		// נפילה לעברית
		cat = i18nCatalogs["he"]
	}
	if cat == nil {
		return out
	}
	for k, msg := range cat.msgs {
		if len(msg.Strs) > 0 && msg.Strs[0] != "" {
			out[k] = msg.Strs[0]
		}
	}
	return out
}

func i18nLookup(ctxt, id string, n int, vars map[string]string) string {
	if id == "" {
		return id
	}
	i18nMu.RLock()
	lang := i18nLang
	msg, plural := i18nFindLocked(lang, ctxt, id)
	if msg == nil && lang != "he" {
		msg, plural = i18nFindLocked("he", ctxt, id)
	}
	i18nMu.RUnlock()

	text := id
	if msg != nil {
		if n >= 0 && len(msg.Strs) > 1 {
			idx := 0
			if plural != nil {
				idx = plural(n)
			} else if n != 1 {
				idx = 1
			}
			if idx < 0 {
				idx = 0
			}
			if idx >= len(msg.Strs) {
				idx = len(msg.Strs) - 1
			}
			if msg.Strs[idx] != "" {
				text = msg.Strs[idx]
			} else if msg.IDPlural != "" && n != 1 {
				text = msg.IDPlural
			}
		} else if len(msg.Strs) > 0 && msg.Strs[0] != "" {
			text = msg.Strs[0]
		}
	} else if n >= 0 && n != 1 {
		// אין תרגום — אם הועבר רבים בנפרד לא כאן; מחזירים id
	}
	if vars != nil {
		if _, ok := vars["כמות"]; !ok && n >= 0 {
			vars["כמות"] = fmtNum(float64(n))
			vars["n"] = vars["כמות"]
		}
		text = interpolateI18n(text, vars)
	} else if n >= 0 && strings.Contains(text, "%d") {
		text = strings.ReplaceAll(text, "%d", fmtNum(float64(n)))
	}
	return text
}

func i18nFindLocked(lang, ctxt, id string) (*i18nMsg, pluralFunc) {
	lang = normalizeLangCode(lang)
	cat := i18nCatalogs[lang]
	if cat == nil {
		return nil, nil
	}
	if msg := cat.msgs[i18nKey(ctxt, id)]; msg != nil {
		return msg, cat.plural
	}
	if ctxt != "" {
		if msg := cat.msgs[i18nKey("", id)]; msg != nil {
			return msg, cat.plural
		}
	}
	return nil, cat.plural
}

func i18nSetLang(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.קבע_שפה", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.קבע_שפה מצפה למחרוזת")
	}
	i18nMu.Lock()
	i18nLang = normalizeLangCode(s)
	i18nMu.Unlock()
	i18nNotifyUI()
	return object.Nil
}

func i18nGetLang(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.שפה", 0, args); err != nil {
		return err
	}
	i18nMu.RLock()
	defer i18nMu.RUnlock()
	return &object.String{Value: i18nLang}
}

func i18nSetGender(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.קבע_מגדר", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.קבע_מגדר מצפה למחרוזת")
	}
	i18nMu.Lock()
	i18nGender = strings.TrimSpace(s)
	i18nMu.Unlock()
	i18nNotifyUI()
	return object.Nil
}

func i18nGetGender(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.מגדר", 0, args); err != nil {
		return err
	}
	i18nMu.RLock()
	defer i18nMu.RUnlock()
	return &object.String{Value: i18nGender}
}

func i18nResolvePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return p
	}
	base := AppBaseDir()
	if base == "" {
		base = "."
	}
	return filepath.Join(base, p)
}

func i18nReadFile(path string) ([]byte, error) {
	full := i18nResolvePath(path)
	return vfs.ReadPrefer(full)
}

func i18nLoadFileInto(lang, path string) error {
	data, err := i18nReadFile(path)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var msgs map[string]*i18nMsg
	var header string
	switch ext {
	case ".mo":
		msgs, header, err = parseMO(data)
	default:
		msgs, header, err = parsePOFile(data)
	}
	if err != nil {
		return err
	}
	lang = normalizeLangCode(lang)
	cat := &i18nCatalog{
		msgs:   msgs,
		header: header,
		plural: pluralFuncFromHeader(header, lang),
	}
	i18nMu.Lock()
	if existing := i18nCatalogs[lang]; existing != nil {
		for k, v := range msgs {
			existing.msgs[k] = v
		}
		if header != "" {
			existing.header = header
			existing.plural = pluralFuncFromHeader(header, lang)
		}
	} else {
		i18nCatalogs[lang] = cat
	}
	i18nMu.Unlock()
	return nil
}

func i18nLoad(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.טען", 2, args); err != nil {
		return err
	}
	lang, ok1 := asString(args[0])
	path, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("תרגום.טען מצפה ל־(קוד_שפה, נתיב)")
	}
	if err := i18nLoadFileInto(lang, path); err != nil {
		return errObj("תרגום.טען נכשל: " + err.Error())
	}
	i18nMu.RLock()
	cur := i18nLang
	i18nMu.RUnlock()
	if normalizeLangCode(lang) == cur {
		i18nNotifyUI()
	}
	return object.Nil
}

func i18nLoadDir(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("תרגום.טען_תיקייה מצפה לנתיב [, דומיין]")
	}
	dir, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.טען_תיקייה מצפה לנתיב מחרוזת")
	}
	full := i18nResolvePath(dir)
	entries, err := os.ReadDir(full)
	if err != nil {
		// ניסיון שמות נפוצים דרך VFS / דיסק
		for _, lang := range []string{"en", "he", "ar", "ru", "fr", "de", "es"} {
			for _, ext := range []string{".mo", ".po"} {
				p := filepath.Join(dir, lang+ext)
				if _, e := i18nReadFile(p); e == nil {
					_ = i18nLoadFileInto(lang, p)
				}
			}
		}
		i18nNotifyUI()
		return object.Nil
	}
	// העדף .mo על .po לאותה שפה
	seen := map[string]string{} // lang → best path
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".po" && ext != ".mo" && ext != ".pot" {
			continue
		}
		if ext == ".pot" {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		lang := normalizeLangCode(base)
		path := filepath.Join(full, name)
		prev, ok := seen[lang]
		if !ok || (ext == ".mo" && strings.HasSuffix(strings.ToLower(prev), ".po")) {
			seen[lang] = path
		}
	}
	for lang, path := range seen {
		if err := i18nLoadFileInto(lang, path); err != nil {
			return errObj("תרגום.טען_תיקייה: " + path + ": " + err.Error())
		}
	}
	i18nNotifyUI()
	return object.Nil
}

func i18nRegister(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.רשום", 2, args); err != nil {
		return err
	}
	lang, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.רשום מצפה לקוד שפה מחרוזת")
	}
	h, ok := args[1].(*object.Hash)
	if !ok {
		return errObj("תרגום.רשום מצפה למילון תרגומים")
	}
	lang = normalizeLangCode(lang)
	msgs := map[string]*i18nMsg{}
	for k, v := range h.Pairs {
		s, ok := asString(v)
		if !ok {
			s = v.Inspect()
		}
		msgs[i18nKey("", k)] = &i18nMsg{ID: k, Strs: []string{s}}
	}
	i18nMu.Lock()
	if existing := i18nCatalogs[lang]; existing != nil {
		for k, v := range msgs {
			existing.msgs[k] = v
		}
	} else {
		i18nCatalogs[lang] = &i18nCatalog{
			msgs:   msgs,
			plural: pluralFuncForLang(lang),
		}
	}
	i18nMu.Unlock()
	i18nMu.RLock()
	cur := i18nLang
	i18nMu.RUnlock()
	if lang == cur {
		i18nNotifyUI()
	}
	return object.Nil
}

func i18nVarsFromArg(arg object.Object) map[string]string {
	if arg == nil || arg == object.Nil {
		return nil
	}
	h, ok := arg.(*object.Hash)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for k, v := range h.Pairs {
		if s, ok := asString(v); ok {
			out[k] = s
		} else if n, ok := v.(*object.Number); ok {
			out[k] = fmtNum(n.Value)
		} else {
			out[k] = v.Inspect()
		}
	}
	return out
}

func i18nT(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("תרגום.ט מצפה למפתח [, משתנים]")
	}
	id, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.ט מצפה למחרוזת מפתח")
	}
	var vars map[string]string
	if len(args) == 2 {
		vars = i18nVarsFromArg(args[1])
	}
	return &object.String{Value: i18nLookup("", id, -1, vars)}
}

func i18nTN(args ...object.Object) object.Object {
	if len(args) < 3 || len(args) > 4 {
		return errObj("תרגום.ט_רבים מצפה ל־(יחיד, רבים, כמות [, משתנים])")
	}
	sing, ok1 := asString(args[0])
	plur, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("תרגום.ט_רבים מצפה למחרוזות יחיד ורבים")
	}
	nObj, ok := args[2].(*object.Number)
	if !ok {
		return errObj("תרגום.ט_רבים מצפה למספר כמות")
	}
	n := int(nObj.Value)
	vars := map[string]string{}
	if len(args) == 4 {
		vars = i18nVarsFromArg(args[3])
		if vars == nil {
			vars = map[string]string{}
		}
	}
	vars["כמות"] = fmtNum(nObj.Value)
	vars["n"] = vars["כמות"]

	// נסה למצוא רשומת רבים לפי msgid היחיד
	text := i18nLookup("", sing, n, vars)
	if text == sing && n != 1 {
		// אין תרגום — השתמש בצורת הרבים המקורית
		text = interpolateI18n(plur, vars)
		if strings.Contains(text, "%d") {
			text = strings.ReplaceAll(text, "%d", vars["כמות"])
		}
	} else if text == sing && n == 1 {
		text = interpolateI18n(sing, vars)
	}
	return &object.String{Value: text}
}

func i18nTCtx(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 3 {
		return errObj("תרגום.ט_הקשר מצפה ל־(הקשר, מפתח [, משתנים])")
	}
	ctx, ok1 := asString(args[0])
	id, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("תרגום.ט_הקשר מצפה למחרוזות")
	}
	var vars map[string]string
	if len(args) == 3 {
		vars = i18nVarsFromArg(args[2])
	}
	return &object.String{Value: i18nLookup(ctx, id, -1, vars)}
}

func i18nTGender(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("תרגום.ט_מגדרי מצפה למפתח [, משתנים]")
	}
	id, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.ט_מגדרי מצפה למחרוזת")
	}
	var vars map[string]string
	if len(args) == 2 {
		vars = i18nVarsFromArg(args[1])
	}
	i18nMu.RLock()
	g := i18nGender
	i18nMu.RUnlock()
	if g != "" {
		t := i18nLookup(g, id, -1, vars)
		if t != id {
			return &object.String{Value: t}
		}
	}
	return &object.String{Value: i18nLookup("", id, -1, vars)}
}

func i18nTNCtx(args ...object.Object) object.Object {
	if len(args) < 4 || len(args) > 5 {
		return errObj("תרגום.ט_רבים_הקשר מצפה ל־(הקשר, יחיד, רבים, כמות [, משתנים])")
	}
	ctx, ok0 := asString(args[0])
	sing, ok1 := asString(args[1])
	plur, ok2 := asString(args[2])
	nObj, ok3 := args[3].(*object.Number)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return errObj("תרגום.ט_רבים_הקשר: ארגומנטים לא תקינים")
	}
	n := int(nObj.Value)
	vars := map[string]string{"כמות": fmtNum(nObj.Value), "n": fmtNum(nObj.Value)}
	if len(args) == 5 {
		if v := i18nVarsFromArg(args[4]); v != nil {
			for k, val := range v {
				vars[k] = val
			}
		}
	}
	text := i18nLookup(ctx, sing, n, vars)
	if text == sing && n != 1 {
		text = interpolateI18n(plur, vars)
		text = strings.ReplaceAll(text, "%d", vars["כמות"])
	}
	return &object.String{Value: text}
}

func i18nHas(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.יש", 1, args); err != nil {
		return err
	}
	id, ok := asString(args[0])
	if !ok {
		return errObj("תרגום.יש מצפה למחרוזת")
	}
	i18nMu.RLock()
	msg, _ := i18nFindLocked(i18nLang, "", id)
	i18nMu.RUnlock()
	return &object.Boolean{Value: msg != nil && len(msg.Strs) > 0 && msg.Strs[0] != ""}
}

func i18nDir(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.כיוון", 0, args); err != nil {
		return err
	}
	return &object.String{Value: I18nDirection()}
}

func i18nIsRTL(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.הוא_rtl", 1, args); err != nil {
		return err
	}
	code, ok := args[0].(*object.String)
	if !ok {
		return errObj("תרגום.הוא_rtl מצפה לקוד שפה מחרוזת")
	}
	return &object.Boolean{Value: i18nDirForLang(code.Value) == "rtl"}
}

func i18nSystemLang(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.שפת_מערכת", 0, args); err != nil {
		return err
	}
	return &object.String{Value: detectSystemLang()}
}

func i18nRefreshUI(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.רענן_ממשק", 0, args); err != nil {
		return err
	}
	i18nNotifyUI()
	return object.Nil
}

func i18nExportCatalog(args ...object.Object) object.Object {
	if err := expectArgs("תרגום.קטלוג", 0, args); err != nil {
		return err
	}
	i18nMu.RLock()
	flat := i18nFlatCatalogLocked(i18nLang)
	lang := i18nLang
	gender := i18nGender
	i18nMu.RUnlock()
	pairs := map[string]object.Object{}
	for k, v := range flat {
		pairs[k] = &object.String{Value: v}
	}
	return &object.Hash{Pairs: map[string]object.Object{
		"שפה":   &object.String{Value: lang},
		"מגדר":  &object.String{Value: gender},
		"כיוון": &object.String{Value: i18nDirForLang(lang)},
		"מילון": &object.Hash{Pairs: pairs},
	}}
}
