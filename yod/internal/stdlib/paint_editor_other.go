//go:build !windows

package stdlib

import "yod/internal/object"

func drawOpenEditor(args ...object.Object) object.Object {
	return errObj("ציור.עורך זמין רק ב־Windows")
}
