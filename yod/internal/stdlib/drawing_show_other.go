//go:build !windows

package stdlib

import (
	"image"

	"yod/internal/object"
)

func showDrawing(img image.Image, title string) object.Object {
	_ = img
	_ = title
	return errObj("ציור.הצג זמין רק ב־Windows — השתמשו ב־שמור() לשמירת קובץ")
}
