package console

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	stdoutMu sync.Mutex
	stdout   io.Writer = os.Stdout
)

// SetStdout מפנה את Print/Println/Printf (למשל לעורך).
func SetStdout(w io.Writer) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	if w == nil {
		stdout = os.Stdout
		return
	}
	stdout = w
}

func currentStdout() io.Writer {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	return stdout
}

// Print / Println / Printf — פלט לקונסול עם תיקון כיוון עברית.
func Print(a ...any) (int, error) {
	return fmt.Fprint(Writer(currentStdout()), a...)
}

func Println(a ...any) (int, error) {
	return fmt.Fprintln(Writer(currentStdout()), a...)
}

func Printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(Writer(currentStdout()), format, a...)
}

func Fprint(w io.Writer, a ...any) (int, error) {
	return fmt.Fprint(Writer(w), a...)
}

func Fprintln(w io.Writer, a ...any) (int, error) {
	return fmt.Fprintln(Writer(w), a...)
}

func Fprintf(w io.Writer, format string, a ...any) (int, error) {
	return fmt.Fprintf(Writer(w), format, a...)
}

type bidiWriter struct {
	w io.Writer
}

func Writer(w io.Writer) io.Writer {
	if w == nil {
		w = os.Stdout
	}
	// בלי היפוך כשמפנים לבופר (עורך) — רק לקונסול אמיתי אם הופעל YOD_CONSOLE_BIDI
	if w != os.Stdout && w != os.Stderr {
		return w
	}
	return &bidiWriter{w: w}
}

func (b *bidiWriter) Write(p []byte) (int, error) {
	fixed := Rewrite(string(p))
	_, err := b.w.Write([]byte(fixed))
	if err != nil {
		return 0, err
	}
	// מדווחים את אורך הקלט המקורי כדי ש־fmt לא יתבלבל
	return len(p), nil
}
