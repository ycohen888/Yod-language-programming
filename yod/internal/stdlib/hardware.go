package stdlib

import "yod/internal/object"

func NewHardwareModule() *object.Module {
	m := &object.Module{Name: "חומרה", Attrs: map[string]object.Object{}}
	m.Attrs["רשימה"] = &object.Builtin{Fn: hardwareList}
	m.Attrs["פתח"] = &object.Builtin{Fn: hardwareOpen}
	return m
}
