package stdlib

import (
	"bytes"
	"os/exec"

	"yod/internal/object"
)

func sysRun(args ...object.Object) object.Object {
	if len(args) < 1 {
		return errObj("מערכת.הפעל מצפה לפקודה")
	}
	cmdName, ok := asString(args[0])
	if !ok {
		return errObj("מערכת.הפעל: הפקודה חייבת להיות מחרוזת")
	}
	cmdArgs := make([]string, 0, len(args)-1)
	for _, a := range args[1:] {
		s, ok := asString(a)
		if !ok {
			return errObj("מערכת.הפעל: ארגומנטים חייבים להיות מחרוזות")
		}
		cmdArgs = append(cmdArgs, s)
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	code := 0
	errMsg := ""
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = -1
			errMsg = err.Error()
		}
	}
	if errMsg == "" && stderr.Len() > 0 {
		errMsg = stderr.String()
	}

	return &object.Hash{Pairs: map[string]object.Object{
		"קוד":    &object.Number{Value: float64(code)},
		"פלט":    &object.String{Value: stdout.String()},
		"שגיאה": &object.String{Value: errMsg},
	}}
}

func sysRunBackground(args ...object.Object) object.Object {
	if len(args) < 1 {
		return errObj("מערכת.הפעל_ברקע מצפה לפקודה")
	}
	cmdName, ok := asString(args[0])
	if !ok {
		return errObj("מערכת.הפעל_ברקע: הפקודה חייבת להיות מחרוזת")
	}
	cmdArgs := make([]string, 0, len(args)-1)
	for _, a := range args[1:] {
		s, ok := asString(a)
		if !ok {
			return errObj("מערכת.הפעל_ברקע: ארגומנטים חייבים להיות מחרוזות")
		}
		cmdArgs = append(cmdArgs, s)
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	if err := cmd.Start(); err != nil {
		return errObj("מערכת.הפעל_ברקע נכשל: " + err.Error())
	}
	return object.Nil
}
