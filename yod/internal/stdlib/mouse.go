package stdlib

import "yod/internal/object"

// שמות כפתורי עכבר — משותפים למשטח ולספריית עכבר
const (
	mouseLeft   = "שמאל"
	mouseRight  = "ימין"
	mouseMiddle = "אמצע"
	mouseOther  = "אחר"
)

func NewMouseModule() *object.Module {
	m := &object.Module{Name: "עכבר", Attrs: map[string]object.Object{}}
	m.Attrs["שמאל"] = &object.String{Value: mouseLeft}
	m.Attrs["ימין"] = &object.String{Value: mouseRight}
	m.Attrs["אמצע"] = &object.String{Value: mouseMiddle}
	m.Attrs["מיקום"] = &object.Builtin{Fn: mousePosition}
	m.Attrs["שמאל_לחוץ"] = &object.Builtin{Fn: mouseLeftDown}
	m.Attrs["ימין_לחוץ"] = &object.Builtin{Fn: mouseRightDown}
	m.Attrs["אמצע_לחוץ"] = &object.Builtin{Fn: mouseMiddleDown}
	m.Attrs["מצב"] = &object.Builtin{Fn: mouseState}
	return m
}

func mousePosHash(x, y int) object.Object {
	return &object.Hash{Pairs: map[string]object.Object{
		"x": &object.Number{Value: float64(x)},
		"y": &object.Number{Value: float64(y)},
	}}
}

func mouseStateHash(x, y int, left, right, middle bool) object.Object {
	return &object.Hash{Pairs: map[string]object.Object{
		"x":     &object.Number{Value: float64(x)},
		"y":     &object.Number{Value: float64(y)},
		"שמאל":  &object.Boolean{Value: left},
		"ימין":  &object.Boolean{Value: right},
		"אמצע":  &object.Boolean{Value: middle},
	}}
}
