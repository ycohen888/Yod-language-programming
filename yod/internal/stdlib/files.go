package stdlib

import (
	"io"
	"os"
	"path/filepath"

	"yod/internal/object"
	"yod/internal/vfs"
)

func NewFilesModule() *object.Module {
	m := &object.Module{Name: "קבצים", Attrs: map[string]object.Object{}}
	m.Attrs["קרא"] = &object.Builtin{Fn: filesRead}
	m.Attrs["קרא_תוצאה"] = &object.Builtin{Fn: filesReadResult}
	m.Attrs["כתוב"] = &object.Builtin{Fn: filesWrite}
	m.Attrs["קיים"] = &object.Builtin{Fn: filesExists}
	m.Attrs["מחק"] = &object.Builtin{Fn: filesDelete}
	m.Attrs["מחק_רקורסיבי"] = &object.Builtin{Fn: filesDeleteRecursive}
	m.Attrs["צרף"] = &object.Builtin{Fn: filesJoin}
	m.Attrs["צור_תיקייה"] = &object.Builtin{Fn: filesMkdir}
	m.Attrs["רשימת_קבצים"] = &object.Builtin{Fn: filesList}
	m.Attrs["האם_תיקייה"] = &object.Builtin{Fn: filesIsDir}
	m.Attrs["הוסף_לקובץ"] = &object.Builtin{Fn: filesAppend}
	m.Attrs["גודל"] = &object.Builtin{Fn: filesSize}
	m.Attrs["זמן_שינוי"] = &object.Builtin{Fn: filesModTime}
	m.Attrs["תיקיית_אב"] = &object.Builtin{Fn: filesParentDir}
	m.Attrs["העתק"] = &object.Builtin{Fn: filesCopy}
	m.Attrs["העבר"] = &object.Builtin{Fn: filesMove}
	m.Attrs["שנה_שם"] = &object.Builtin{Fn: filesRename}
	m.Attrs["ארוז"] = &object.Builtin{Fn: filesArchive}
	m.Attrs["חלץ"] = &object.Builtin{Fn: filesExtract}
	m.Attrs["תוכן_ארכיון"] = &object.Builtin{Fn: filesArchiveContents}
	registerCopyFns(m)
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
	data, e := vfs.ReadPrefer(path)
	if e != nil {
		return errObj("לא הצלחתי לקרוא קובץ: " + e.Error())
	}
	return &object.String{Value: string(data)}
}

func filesReadResult(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.קרא_תוצאה", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return object.ResultErr("קבצים.קרא_תוצאה מצפה לנתיב מחרוזת", 1)
	}
	data, e := vfs.ReadPrefer(path)
	if e != nil {
		return object.ResultErr("לא הצלחתי לקרוא קובץ: "+e.Error(), 1)
	}
	return object.ResultOk(&object.String{Value: string(data)})
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
	okExists, e := vfs.ExistsPrefer(path)
	if e != nil {
		return errObj("בדיקת קיום נכשלה: " + e.Error())
	}
	return &object.Boolean{Value: okExists}
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

func filesDeleteRecursive(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.מחק_רקורסיבי", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.מחק_רקורסיבי מצפה לנתיב מחרוזת")
	}
	if e := os.RemoveAll(path); e != nil {
		return errObj("לא הצלחתי למחוק: " + e.Error())
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
	if fs := vfs.Active(); fs != nil {
		if names, ok := fs.List(path); ok {
			arr := &object.Array{Elements: make([]object.Object, 0, len(names))}
			for _, name := range names {
				arr.Elements = append(arr.Elements, &object.String{Value: name})
			}
			return arr
		}
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
	isDir, e := vfs.IsDirPrefer(path)
	if e != nil {
		return errObj("בדיקת תיקייה נכשלה: " + e.Error())
	}
	return &object.Boolean{Value: isDir}
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

func filesModTime(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.זמן_שינוי", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.זמן_שינוי מצפה לנתיב מחרוזת")
	}
	info, e := os.Stat(path)
	if e != nil {
		return errObj("לא הצלחתי לקרוא זמן שינוי: " + e.Error())
	}
	return &object.Number{Value: float64(info.ModTime().Unix())}
}

func filesParentDir(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.תיקיית_אב", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.תיקיית_אב מצפה לנתיב מחרוזת")
	}
	parent := filepath.Dir(path)
	if parent == "." || parent == path {
		return &object.String{Value: ""}
	}
	return &object.String{Value: parent}
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
	if dir := filepath.Dir(dst); dir != "" && dir != "." {
		if e := os.MkdirAll(dir, 0755); e != nil {
			return errObj("קבצים.העתק: לא הצלחתי ליצור תיקיית יעד: " + e.Error())
		}
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

