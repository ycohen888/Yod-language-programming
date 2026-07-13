//go:build !windows

package stdlib

import "yod/internal/object"

func clipboardRead(args ...object.Object) object.Object {
	return errObj("לוח זמין רק ב־Windows")
}

func clipboardWrite(args ...object.Object) object.Object {
	return errObj("לוח זמין רק ב־Windows")
}
