//go:build !windows

package stdlib

import "yod/internal/object"

func speechSpeak(args ...object.Object) object.Object {
	return errObj("דיבור זמין רק ב־Windows")
}

func speechStop(args ...object.Object) object.Object {
	return errObj("דיבור זמין רק ב־Windows")
}

func speechRate(args ...object.Object) object.Object {
	return errObj("דיבור זמין רק ב־Windows")
}
