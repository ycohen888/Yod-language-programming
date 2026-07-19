//go:build !windows

package stdlib

import "yod/internal/object"

func recordPickRegion(args ...object.Object) object.Object {
	return errObj("הקלטה.בחר_אזור זמין רק ב־Windows")
}

func recordListMics(args ...object.Object) object.Object {
	return errObj("הקלטה.רשימת_מיקרופונים זמינה רק ב־Windows")
}

func recordStartScreen(args ...object.Object) object.Object {
	return errObj("הקלטה.התחל_מסך זמין רק ב־Windows")
}

func recordStartAudio(args ...object.Object) object.Object {
	return errObj("הקלטה.התחל_קול זמין רק ב־Windows")
}

func recordStop(args ...object.Object) object.Object {
	return errObj("הקלטה.עצור זמין רק ב־Windows")
}

func recordStopAll() {}

func recordApplyTexts(args ...object.Object) object.Object {
	return errObj("הקלטה.החל_טקסטים זמין רק ב־Windows")
}

func recordDefaultFont(args ...object.Object) object.Object {
	return &object.String{Value: ""}
}