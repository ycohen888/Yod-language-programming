package stdlib

import (
	"image"

	"yod/internal/object"
)

func NewImagesModule() *object.Module {
	m := &object.Module{Name: "תמונות", Attrs: map[string]object.Object{}}
	// מעטפת דקה מעל ציור — אותו יישום (loadImageFile / wrapImage / saveImageFile)
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
	return imageLoadFromPath(path)
}

func imagesSave(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("תמונות.שמור מצפה לתמונה ונתיב")
	}
	path, ok := asString(args[1])
	if !ok {
		return errObj("תמונות.שמור מצפה לנתיב מחרוזת")
	}
	return imageSaveWidget(args[0], path, "תמונות.שמור")
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
