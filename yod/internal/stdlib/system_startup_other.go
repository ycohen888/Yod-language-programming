//go:build !windows

package stdlib

import "yod/internal/object"

func sysAddStartup(args ...object.Object) object.Object {
	return errObj("מערכת.הוסף_להפעלה זמינה רק ב־Windows")
}

func sysRemoveStartup(args ...object.Object) object.Object {
	return errObj("מערכת.הסר_מהפעלה זמינה רק ב־Windows")
}

func sysIsStartup(args ...object.Object) object.Object {
	return errObj("מערכת.רשום_בהפעלה זמינה רק ב־Windows")
}
