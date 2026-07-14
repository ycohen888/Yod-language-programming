package stdlib

import "yod/internal/object"

func NewVideoModule() *object.Module {
	m := &object.Module{Name: "וידאו", Attrs: map[string]object.Object{}}
	m.Attrs["חלונית"] = &object.Builtin{Fn: videoCreatePanel}
	return m
}
