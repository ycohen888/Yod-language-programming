//go:build !windows

package console

// Init אין פעולה במערכות שאינן Windows.
func Init() {}

// HideIfOwned אין פעולה מחוץ ל־Windows.
func HideIfOwned() {}
