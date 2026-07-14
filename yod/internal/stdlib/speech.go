package stdlib

import "yod/internal/object"

func NewSpeechModule() *object.Module {
	m := &object.Module{Name: "דיבור", Attrs: map[string]object.Object{}}
	m.Attrs["הקרא"] = &object.Builtin{Fn: speechSpeak}
	m.Attrs["עצור"] = &object.Builtin{Fn: speechStop}
	m.Attrs["קצב"] = &object.Builtin{Fn: speechRate}
	return m
}
