//go:build !windows

package stdlib

import "yod/internal/object"

func keyboardPressed(args ...object.Object) object.Object {
	return errObj("מקלדת זמינה רק ב־Windows")
}

func keyboardState(args ...object.Object) object.Object {
	return errObj("מקלדת זמינה רק ב־Windows")
}
