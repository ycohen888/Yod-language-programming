//go:build !windows

package editor

func tryLaunchWebIDE(openPath string) bool { return false }

func webIDEHint() string { return "עורך Electron זמין כרגע ב־Windows" }
