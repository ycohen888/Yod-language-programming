//go:build !windows

package stdlib

import "yod/internal/object"

func NewWindowsModule() *object.Module {
	m := &object.Module{Name: "חלונות", Attrs: map[string]object.Object{}}
	unsupported := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return errObj("ספריית חלונות זמינה רק ב־Windows")
	}}
	m.Attrs["חלון"] = unsupported
	m.Attrs["כפתור"] = unsupported
	m.Attrs["תווית"] = unsupported
	m.Attrs["שדה"] = unsupported
	m.Attrs["נורית"] = unsupported
	m.Attrs["דפדפן"] = unsupported
	m.Attrs["שורה"] = unsupported
	m.Attrs["עמודה"] = unsupported
	m.Attrs["מסגרת"] = unsupported
	m.Attrs["משטח"] = unsupported
	m.Attrs["הודעה"] = unsupported
	m.Attrs["בחר_שמירה"] = unsupported
	m.Attrs["בחר_פתיחה"] = unsupported
	return m
}
