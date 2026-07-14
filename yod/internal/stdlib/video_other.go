//go:build !windows

package stdlib

import "yod/internal/object"

func videoCreatePanel(args ...object.Object) object.Object {
	return errObj("וידאו זמין רק ב־Windows")
}
