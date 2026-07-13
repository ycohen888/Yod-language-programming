//go:build !windows

package stdlib

import "yod/internal/object"

func winCreateChartBridge(args ...object.Object) object.Object {
	return errObj("ספריית גרפים זמינה רק ב־Windows")
}

func newChartWidgetSafe(kind string) object.Object {
	return errObj("ספריית גרפים זמינה רק ב־Windows")
}
