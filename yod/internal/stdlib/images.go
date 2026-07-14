package stdlib

import (
	"image"

	"yod/internal/object"
)

func NewImagesModule() *object.Module {
	m := &object.Module{Name: "תמונות", Attrs: map[string]object.Object{}}
	m.Attrs["טען"] = &object.Builtin{Fn: imagesLoad}
	m.Attrs["שמור"] = &object.Builtin{Fn: imagesSave}
	return m
}

func imagesLoad(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("תמונות.טען מצפה לנתיב אחד")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("תמונות.טען מצפה למחרוזת")
	}
	img, err := loadImageFile(path)
	if err != nil {
		return errObj(err.Error())
	}
	return wrapImage(img)
}

func imagesSave(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("תמונות.שמור מצפה לתמונה ונתיב")
	}
	gw, ok := args[0].(*object.GuiWidget)
	if !ok {
		return errObj("תמונות.שמור מצפה לרכיב תמונה")
	}
	di, ok := gw.Data.(*drawImage)
	if !ok || di.img == nil {
		return errObj("תמונות.שמור מצפה לרכיב תמונה")
	}
	path, ok := asString(args[1])
	if !ok {
		return errObj("תמונות.שמור מצפה לנתיב מחרוזת")
	}
	out := resolveAppPath(path)
	if err := saveImageFile(di.img, out); err != nil {
		return errObj(err.Error())
	}
	return object.Nil
}

// imagesFromWidget — עזר פנימי לציור למשטח.
func imagesFromWidget(obj object.Object) (image.Image, bool) {
	gw, ok := obj.(*object.GuiWidget)
	if !ok {
		return nil, false
	}
	di, ok := gw.Data.(*drawImage)
	if !ok || di.img == nil {
		return nil, false
	}
	return di.img, true
}
