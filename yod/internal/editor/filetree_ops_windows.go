//go:build windows

package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func promptTextDialog(owner walk.Form, title, label, initial string) (string, bool) {
	var dlg *walk.Dialog
	var edit *walk.LineEdit
	var acceptPB *walk.PushButton
	accepted := false
	result := ""

	accept := func() {
		result = strings.TrimSpace(edit.Text())
		if result == "" {
			walk.MsgBox(dlg, title, "נא להזין שם.", walk.MsgBoxIconWarning)
			return
		}
		if strings.ContainsAny(result, `\/:*?"<>|`) {
			walk.MsgBox(dlg, title, "השם מכיל תווים לא חוקיים.", walk.MsgBoxIconWarning)
			return
		}
		accepted = true
		dlg.Accept()
	}

	_, _ = Dialog{
		AssignTo:           &dlg,
		Title:              title,
		MinSize:            Size{Width: 360, Height: 150},
		Layout:             VBox{Margins: Margins{Left: 14, Right: 14, Top: 12, Bottom: 12}, Spacing: 10},
		DefaultButton:      &acceptPB,
		RightToLeftReading: true,
		Children: []Widget{
			Label{Text: label, RightToLeftReading: true},
			LineEdit{
				AssignTo:           &edit,
				Text:               initial,
				RightToLeftReading: true,
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn {
						accept()
					}
				},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &acceptPB, Text: "אישור", OnClicked: accept},
					PushButton{Text: "ביטול", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(owner)

	return result, accepted
}

// promptPathDialog — הזנת נתיב מלא (מאפשר \ ו־:)
func promptPathDialog(owner walk.Form, title, label, initial string) (string, bool) {
	var dlg *walk.Dialog
	var edit *walk.LineEdit
	var acceptPB *walk.PushButton
	accepted := false
	result := ""

	accept := func() {
		result = strings.TrimSpace(edit.Text())
		if result == "" {
			walk.MsgBox(dlg, title, "נא להזין נתיב.", walk.MsgBoxIconWarning)
			return
		}
		accepted = true
		dlg.Accept()
	}

	_, _ = Dialog{
		AssignTo:           &dlg,
		Title:              title,
		MinSize:            Size{Width: 480, Height: 150},
		Layout:             VBox{Margins: Margins{Left: 14, Right: 14, Top: 12, Bottom: 12}, Spacing: 10},
		DefaultButton:      &acceptPB,
		RightToLeftReading: true,
		Children: []Widget{
			Label{Text: label, RightToLeftReading: true},
			LineEdit{
				AssignTo:           &edit,
				Text:               initial,
				RightToLeftReading: true,
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn {
						accept()
					}
				},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &acceptPB, Text: "אישור", OnClicked: accept},
					PushButton{Text: "ביטול", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(owner)

	return result, accepted
}

func createNewFileOnDisk(dir, name string) (string, error) {
	if filepath.Ext(name) == "" {
		name += ".יוד"
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("הקובץ כבר קיים: %s", name)
	}
	content := newFileTemplate
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func createNewFolderOnDisk(dir, name string) (string, error) {
	path := filepath.Join(dir, name)
	if err := os.Mkdir(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func renamePathOnDisk(oldPath, newName string) (string, error) {
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, newName)
	if filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return oldPath, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("כבר קיים פריט בשם %q", newName)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}

func deletePathOnDisk(path string, isDir bool) error {
	if isDir {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func revealInExplorer(path string) {
	path = filepath.Clean(path)
	_ = exec.Command("explorer", "/select,", path).Start()
}
