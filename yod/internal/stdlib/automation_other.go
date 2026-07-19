//go:build !windows

package stdlib

import "yod/internal/object"

func automationMoveMouse(args ...object.Object) object.Object {
	return errObj("אוטומציה.הזז_עכבר זמין רק ב־Windows")
}

func automationClickMouse(args ...object.Object) object.Object {
	return errObj("אוטומציה.לחץ_עכבר זמין רק ב־Windows")
}

func automationDragMouse(args ...object.Object) object.Object {
	return errObj("אוטומציה.גרור_עכבר זמין רק ב־Windows")
}

func automationScrollWheel(args ...object.Object) object.Object {
	return errObj("אוטומציה.גלגל_עכבר זמין רק ב־Windows")
}

func automationType(args ...object.Object) object.Object {
	return errObj("אוטומציה.הקלד זמין רק ב־Windows")
}

func automationTypeDelayed(args ...object.Object) object.Object {
	return errObj("אוטומציה.הקלד_מעכב זמין רק ב־Windows")
}

func automationKeyPress(args ...object.Object) object.Object {
	return errObj("אוטומציה.לחץ_מקש זמין רק ב־Windows")
}

func automationHotkey(args ...object.Object) object.Object {
	return errObj("אוטומציה.קיצור זמין רק ב־Windows")
}

func automationKeyDown(args ...object.Object) object.Object {
	return errObj("אוטומציה.החזק זמין רק ב־Windows")
}

func automationKeyUp(args ...object.Object) object.Object {
	return errObj("אוטומציה.שחרר זמין רק ב־Windows")
}

func automationSleep(args ...object.Object) object.Object {
	return errObj("אוטומציה.המתן זמין רק ב־Windows")
}

func automationScreenshot(args ...object.Object) object.Object {
	return errObj("אוטומציה.צלם_מסך זמין רק ב־Windows")
}

func automationScreenshotToFile(args ...object.Object) object.Object {
	return errObj("אוטומציה.צלם_מסך_לקובץ זמין רק ב־Windows")
}

func automationFindWindow(args ...object.Object) object.Object {
	return errObj("אוטומציה.מצא_חלון זמין רק ב־Windows")
}

func automationListWindows(args ...object.Object) object.Object {
	return errObj("אוטומציה.רשימת_חלונות זמין רק ב־Windows")
}

func automationWaitForWindow(args ...object.Object) object.Object {
	return errObj("אוטומציה.המתן_עד_חלון זמין רק ב־Windows")
}

func automationActivateWindow(args ...object.Object) object.Object {
	return errObj("אוטומציה.הפעל_חלון זמין רק ב־Windows")
}

func automationWindowRect(args ...object.Object) object.Object {
	return errObj("אוטומציה.מלבן_חלון זמין רק ב־Windows")
}
