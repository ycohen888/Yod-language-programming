//go:build !windows

package stdlib

import "yod/internal/object"

func hardwareList(args ...object.Object) object.Object {
	return errObj("חומרה.רשימה זמינה רק ב־Windows")
}

func hardwareOpen(args ...object.Object) object.Object {
	return errObj("חומרה.פתח זמין רק ב־Windows")
}
