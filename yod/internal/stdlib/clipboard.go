package stdlib

import (
	"yod/internal/object"
)

func NewClipboardModule() *object.Module {
	m := &object.Module{Name: "לוח", Attrs: map[string]object.Object{}}
	m.Attrs["קרא"] = &object.Builtin{Fn: clipboardRead}
	m.Attrs["כתוב"] = &object.Builtin{Fn: clipboardWrite}
	return m
}
