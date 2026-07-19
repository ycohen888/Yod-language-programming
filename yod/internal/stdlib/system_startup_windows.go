//go:build windows

package stdlib

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"

	"yod/internal/object"
)

const windowsRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func sysAddStartup(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("מערכת.הוסף_להפעלה מצפה לשם [, נתיב_exe]")
	}
	name, ok := asString(args[0])
	if !ok || strings.TrimSpace(name) == "" {
		return errObj("מערכת.הוסף_להפעלה: שם חייב מחרוזת לא ריקה")
	}
	exePath := ""
	if len(args) == 2 {
		p, ok := asString(args[1])
		if !ok {
			return errObj("מערכת.הוסף_להפעלה: נתיב חייב מחרוזת")
		}
		exePath = strings.TrimSpace(p)
	}
	if exePath == "" {
		p, err := os.Executable()
		if err != nil {
			return errObj("מערכת.הוסף_להפעלה: " + err.Error())
		}
		exePath = p
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE)
	if err != nil {
		return errObj("מערכת.הוסף_להפעלה: " + err.Error())
	}
	defer key.Close()
	quoted := exePath
	if !strings.HasPrefix(strings.TrimSpace(exePath), `"`) {
		quoted = `"` + exePath + `"`
	}
	if err := key.SetStringValue(name, quoted); err != nil {
		return errObj("מערכת.הוסף_להפעלה: " + err.Error())
	}
	return object.Nil
}

func sysRemoveStartup(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מערכת.הסר_מהפעלה מצפה לשם")
	}
	name, ok := asString(args[0])
	if !ok || strings.TrimSpace(name) == "" {
		return errObj("מערכת.הסר_מהפעלה: שם חייב מחרוזת לא ריקה")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE)
	if err != nil {
		return errObj("מערכת.הסר_מהפעלה: " + err.Error())
	}
	defer key.Close()
	_ = key.DeleteValue(name)
	return object.Nil
}

func sysIsStartup(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מערכת.רשום_בהפעלה מצפה לשם")
	}
	name, ok := asString(args[0])
	if !ok || strings.TrimSpace(name) == "" {
		return errObj("מערכת.רשום_בהפעלה: שם חייב מחרוזת לא ריקה")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.QUERY_VALUE)
	if err != nil {
		return &object.Boolean{Value: false}
	}
	defer key.Close()
	_, _, err = key.GetStringValue(name)
	return &object.Boolean{Value: err == nil}
}
