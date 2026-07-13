//go:build windows

package editor

import (
	"errors"

	"github.com/lxn/walk"
	"github.com/ncruces/zenity"
)

// pickFolder פותח דיאלוג בחירת תיקייה (מימוש יציב דרך zenity / IFileOpenDialog).
func pickFolder(owner walk.Form, title, initial string) (string, bool, error) {
	opts := []zenity.Option{
		zenity.Directory(),
		zenity.Title(title),
	}
	if initial != "" {
		opts = append(opts, zenity.Filename(initial))
	}
	if owner != nil {
		opts = append(opts, zenity.Attach(uintptr(owner.Handle())))
	}

	path, err := zenity.SelectFile(opts...)
	if err != nil {
		if errors.Is(err, zenity.ErrCanceled) {
			return "", false, nil
		}
		return "", false, err
	}
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}
