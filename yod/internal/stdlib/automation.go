package stdlib

import (
	"strings"

	"yod/internal/object"
)

func matchTitlePattern(pattern, title string) bool {
	if pattern == "*" {
		return title != ""
	}
	p := strings.ToLower(pattern)
	t := strings.ToLower(title)
	if !strings.Contains(p, "*") {
		return strings.EqualFold(pattern, title) || strings.Contains(t, p)
	}
	if strings.HasPrefix(p, "*") && strings.HasSuffix(p, "*") {
		return strings.Contains(t, strings.Trim(p, "*"))
	}
	if strings.HasPrefix(p, "*") {
		return strings.HasSuffix(t, strings.TrimPrefix(p, "*"))
	}
	if strings.HasSuffix(p, "*") {
		return strings.HasPrefix(t, strings.TrimSuffix(p, "*"))
	}
	return t == p
}

func NewAutomationModule() *object.Module {
	m := &object.Module{Name: "אוטומציה", Attrs: map[string]object.Object{}}
	m.Attrs["הזז_עכבר"] = &object.Builtin{Fn: automationMoveMouse}
	m.Attrs["לחץ_עכבר"] = &object.Builtin{Fn: automationClickMouse}
	m.Attrs["גרור_עכבר"] = &object.Builtin{Fn: automationDragMouse}
	m.Attrs["גלגל_עכבר"] = &object.Builtin{Fn: automationScrollWheel}
	m.Attrs["הקלד"] = &object.Builtin{Fn: automationType}
	m.Attrs["הקלד_מעכב"] = &object.Builtin{Fn: automationTypeDelayed}
	m.Attrs["לחץ_מקש"] = &object.Builtin{Fn: automationKeyPress}
	m.Attrs["קיצור"] = &object.Builtin{Fn: automationHotkey}
	m.Attrs["החזק"] = &object.Builtin{Fn: automationKeyDown}
	m.Attrs["שחרר"] = &object.Builtin{Fn: automationKeyUp}
	m.Attrs["המתן"] = &object.Builtin{Fn: automationSleep}
	m.Attrs["צלם_מסך"] = &object.Builtin{Fn: automationScreenshot}
	m.Attrs["צלם_מסך_לקובץ"] = &object.Builtin{Fn: automationScreenshotToFile}
	m.Attrs["מצא_חלון"] = &object.Builtin{Fn: automationFindWindow}
	m.Attrs["רשימת_חלונות"] = &object.Builtin{Fn: automationListWindows}
	m.Attrs["המתן_עד_חלון"] = &object.Builtin{Fn: automationWaitForWindow}
	m.Attrs["הפעל_חלון"] = &object.Builtin{Fn: automationActivateWindow}
	m.Attrs["מלבן_חלון"] = &object.Builtin{Fn: automationWindowRect}
	return m
}
