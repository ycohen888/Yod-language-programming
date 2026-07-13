package stdlib

import (
	"io"
	"os"
	"path/filepath"

	"yod/internal/object"
)

func NewFilesModule() *object.Module {
	m := &object.Module{Name: "קבצים", Attrs: map[string]object.Object{}}
	m.Attrs["קרא"] = &object.Builtin{Fn: filesRead}
	m.Attrs["כתוב"] = &object.Builtin{Fn: filesWrite}
	m.Attrs["קיים"] = &object.Builtin{Fn: filesExists}
	m.Attrs["מחק"] = &object.Builtin{Fn: filesDelete}
	m.Attrs["צרף"] = &object.Builtin{Fn: filesJoin}
	m.Attrs["צור_תיקייה"] = &object.Builtin{Fn: filesMkdir}
	m.Attrs["רשימת_קבצים"] = &object.Builtin{Fn: filesList}
	m.Attrs["האם_תיקייה"] = &object.Builtin{Fn: filesIsDir}
	m.Attrs["הוסף_לקובץ"] = &object.Builtin{Fn: filesAppend}
	m.Attrs["גודל"] = &object.Builtin{Fn: filesSize}
	m.Attrs["העתק"] = &object.Builtin{Fn: filesCopy}
	m.Attrs["העבר"] = &object.Builtin{Fn: filesMove}
	m.Attrs["שנה_שם"] = &object.Builtin{Fn: filesRename}
	m.Attrs["ארוז"] = &object.Builtin{Fn: filesArchive}
	m.Attrs["חלץ"] = &object.Builtin{Fn: filesExtract}
	m.Attrs["תוכן_ארכיון"] = &object.Builtin{Fn: filesArchiveContents}
	return m
}

func filesRead(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.קרא", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.קרא מצפה לנתיב מחרוזת")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return errObj("לא הצלחתי לקרוא קובץ: " + e.Error())
	}
	return &object.String{Value: string(data)}
}

func filesWrite(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.כתוב", 2, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.כתוב: נתיב חייב להיות מחרוזת")
	}
	content := args[1].Inspect()
	if s, ok := args[1].(*object.String); ok {
		content = s.Value
	}
	if e := os.WriteFile(path, []byte(content), 0644); e != nil {
		return errObj("לא הצלחתי לכתוב קובץ: " + e.Error())
	}
	return &object.Null{}
}

func filesExists(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.קיים", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.קיים מצפה לנתיב מחרוזת")
	}
	_, e := os.Stat(path)
	if e == nil {
		return &object.Boolean{Value: true}
	}
	if os.IsNotExist(e) {
		return &object.Boolean{Value: false}
	}
	return errObj("בדיקת קיום נכשלה: " + e.Error())
}

func filesDelete(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.מחק", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.מחק מצפה לנתיב מחרוזת")
	}
	if e := os.Remove(path); e != nil {
		return errObj("לא הצלחתי למחוק קובץ: " + e.Error())
	}
	return &object.Null{}
}

func filesJoin(args ...object.Object) object.Object {
	if len(args) < 1 {
		return errObj("קבצים.צרף מצפה לפחות לנתיב אחד")
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		s, ok := asString(a)
		if !ok {
			return errObj("קבצים.צרף מצפה למחרוזות")
		}
		parts = append(parts, s)
	}
	return &object.String{Value: filepath.Join(parts...)}
}

func filesMkdir(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.צור_תיקייה", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.צור_תיקייה מצפה לנתיב מחרוזת")
	}
	if e := os.MkdirAll(path, 0755); e != nil {
		return errObj("לא הצלחתי ליצור תיקייה: " + e.Error())
	}
	return object.Nil
}

func filesList(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.רשימת_קבצים", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.רשימת_קבצים מצפה לנתיב מחרוזת")
	}
	entries, e := os.ReadDir(path)
	if e != nil {
		return errObj("לא הצלחתי לקרוא תיקייה: " + e.Error())
	}
	arr := &object.Array{Elements: make([]object.Object, 0, len(entries))}
	for _, ent := range entries {
		arr.Elements = append(arr.Elements, &object.String{Value: ent.Name()})
	}
	return arr
}

func filesIsDir(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.האם_תיקייה", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.האם_תיקייה מצפה לנתיב מחרוזת")
	}
	info, e := os.Stat(path)
	if e != nil {
		if os.IsNotExist(e) {
			return &object.Boolean{Value: false}
		}
		return errObj("בדיקת תיקייה נכשלה: " + e.Error())
	}
	return &object.Boolean{Value: info.IsDir()}
}

func filesAppend(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.הוסף_לקובץ", 2, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.הוסף_לקובץ: נתיב חייב להיות מחרוזת")
	}
	content := args[1].Inspect()
	if s, ok := args[1].(*object.String); ok {
		content = s.Value
	}
	f, e := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return errObj("לא הצלחתי לפתוח קובץ להוספה: " + e.Error())
	}
	defer f.Close()
	if _, e := f.WriteString(content); e != nil {
		return errObj("לא הצלחתי להוסיף לקובץ: " + e.Error())
	}
	return object.Nil
}

func filesSize(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.גודל", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.גודל מצפה לנתיב מחרוזת")
	}
	info, e := os.Stat(path)
	if e != nil {
		return errObj("לא הצלחתי לקרוא גודל: " + e.Error())
	}
	return &object.Number{Value: float64(info.Size())}
}

func filesCopy(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.העתק", 2, args); err != nil {
		return err
	}
	src, ok1 := asString(args[0])
	dst, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("קבצים.העתק מצפה לנתיבי מחרוזת")
	}
	in, e := os.Open(src)
	if e != nil {
		return errObj("קבצים.העתק: לא הצלחתי לפתוח מקור: " + e.Error())
	}
	defer in.Close()
	out, e := os.Create(dst)
	if e != nil {
		return errObj("קבצים.העתק: לא הצלחתי ליצור יעד: " + e.Error())
	}
	defer out.Close()
	if _, e := io.Copy(out, in); e != nil {
		return errObj("קבצים.העתק נכשל: " + e.Error())
	}
	return object.Nil
}

func filesMove(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.העבר", 2, args); err != nil {
		return err
	}
	src, ok1 := asString(args[0])
	dst, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("קבצים.העבר מצפה לנתיבי מחרוזת")
	}
	if e := os.Rename(src, dst); e == nil {
		return object.Nil
	}
	in, e := os.Open(src)
	if e != nil {
		return errObj("קבצים.העבר: לא הצלחתי לפתוח מקור: " + e.Error())
	}
	defer in.Close()
	out, e := os.Create(dst)
	if e != nil {
		return errObj("קבצים.העבר: לא הצלחתי ליצור יעד: " + e.Error())
	}
	if _, e := io.Copy(out, in); e != nil {
		out.Close()
		return errObj("קבצים.העבר נכשל: " + e.Error())
	}
	out.Close()
	if e := os.Remove(src); e != nil {
		return errObj("קבצים.העבר: הועתק אך מחיקת מקור נכשלה: " + e.Error())
	}
	return object.Nil
}

func filesRename(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.שנה_שם", 2, args); err != nil {
		return err
	}
	src, ok1 := asString(args[0])
	dst, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("קבצים.שנה_שם מצפה לנתיבי מחרוזת")
	}
	if e := os.Rename(src, dst); e != nil {
		return errObj("קבצים.שנה_שם נכשל: " + e.Error())
	}
	return object.Nil
}

