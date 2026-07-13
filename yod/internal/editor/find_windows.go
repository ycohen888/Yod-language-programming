//go:build windows

package editor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func showGotoLineDialog(owner walk.Form, gotoLine func(int)) {
	var dlg *walk.Dialog
	var lineEdit *walk.LineEdit
	var acceptPB *walk.PushButton

	goNow := func() {
		n, err := strconv.Atoi(strings.TrimSpace(lineEdit.Text()))
		if err != nil || n < 1 {
			walk.MsgBox(dlg, "מעבר לשורה", "נא להזין מספר שורה תקין.", walk.MsgBoxIconWarning)
			return
		}
		gotoLine(n)
		dlg.Accept()
	}

	_, _ = Dialog{
		AssignTo:           &dlg,
		Title:              "מעבר לשורה",
		MinSize:            Size{Width: 320, Height: 140},
		Layout:             VBox{Margins: Margins{Left: 14, Right: 14, Top: 12, Bottom: 12}, Spacing: 10},
		DefaultButton:      &acceptPB,
		RightToLeftReading: true,
		Children: []Widget{
			Label{Text: "מספר שורה:", RightToLeftReading: true},
			LineEdit{
				AssignTo:           &lineEdit,
				RightToLeftReading: true,
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn {
						goNow()
					}
				},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &acceptPB, Text: "עבור", OnClicked: goNow},
					PushButton{Text: "ביטול", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(owner)
}

func showFindDialog(owner walk.Form, ce *CodeEdit, replaceMode bool) {
	if ce == nil {
		return
	}
	var dlg *walk.Dialog
	var findEdit, replEdit *walk.LineEdit
	var matchCase *walk.CheckBox
	var status *walk.Label

	title := "חיפוש"
	if replaceMode {
		title = "החלפה"
	}

	doFind := func() {
		ok := ce.FindNext(findEdit.Text(), matchCase.Checked())
		if status != nil {
			if ok {
				status.SetText("נמצא")
			} else {
				status.SetText("לא נמצא")
			}
		}
	}
	doReplace := func() {
		needle := findEdit.Text()
		if needle == "" {
			return
		}
		start, end := ce.TextSelection()
		sel := ""
		if end > start {
			sel = substringUTF16(ce.indexText(), start, end)
		}
		matched := sel == needle || (!matchCase.Checked() && strings.EqualFold(sel, needle))
		if matched {
			ce.ReplaceSelection(replEdit.Text())
			ce.FindNext(needle, matchCase.Checked())
			if status != nil {
				status.SetText("הוחלף")
			}
			return
		}
		doFind()
	}
	doReplaceAll := func() {
		n := ce.ReplaceAll(findEdit.Text(), replEdit.Text(), matchCase.Checked())
		if status != nil {
			status.SetText(fmt.Sprintf("הוחלפו %d", n))
		}
	}

	children := []Widget{
		Label{Text: "חפש:", RightToLeftReading: true},
		LineEdit{AssignTo: &findEdit, RightToLeftReading: true},
	}
	if replaceMode {
		children = append(children,
			Label{Text: "החלף ב:", RightToLeftReading: true},
			LineEdit{AssignTo: &replEdit, RightToLeftReading: true},
		)
	} else {
		children = append(children, LineEdit{AssignTo: &replEdit, Visible: false})
	}
	children = append(children,
		CheckBox{AssignTo: &matchCase, Text: "התאם רישיות (A/a)", RightToLeftReading: true},
		Label{AssignTo: &status, Text: " ", TextColor: colMuted, RightToLeftReading: true},
		Composite{
			Layout: HBox{Spacing: 6},
			Children: []Widget{
				PushButton{Text: "הבא", OnClicked: doFind},
				PushButton{Text: "החלף", Visible: replaceMode, OnClicked: doReplace},
				PushButton{Text: "החלף הכל", Visible: replaceMode, OnClicked: doReplaceAll},
				HSpacer{},
				PushButton{Text: "סגור", OnClicked: func() { dlg.Accept() }},
			},
		},
	)

	_, _ = Dialog{
		AssignTo:           &dlg,
		Title:              title,
		MinSize:            Size{Width: 420, Height: 220},
		Layout:             VBox{Margins: Margins{Left: 14, Right: 14, Top: 12, Bottom: 12}, Spacing: 8},
		RightToLeftReading: true,
		Children:           children,
	}.Run(owner)
}
