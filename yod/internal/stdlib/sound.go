package stdlib

import (
	"path/filepath"
	"sync"

	"yod/internal/object"
)

func NewSoundModule() *object.Module {
	m := &object.Module{Name: "שמע", Attrs: map[string]object.Object{}}
	m.Attrs["נגן_אפקט"] = &object.Builtin{Fn: soundPlayEffect}
	m.Attrs["נגן"] = &object.Builtin{Fn: soundPlay}
	m.Attrs["השהה"] = &object.Builtin{Fn: soundPause}
	m.Attrs["המשך"] = &object.Builtin{Fn: soundResume}
	m.Attrs["עצור"] = &object.Builtin{Fn: soundStop}
	m.Attrs["לולאה"] = &object.Builtin{Fn: soundLoop}
	m.Attrs["עוצמה"] = &object.Builtin{Fn: soundVolume}
	m.Attrs["מנגן"] = &object.Builtin{Fn: soundIsPlaying}
	m.Attrs["בעת_סיום"] = &object.Builtin{Fn: soundOnEnd}
	return m
}

var (
	soundMu       sync.Mutex
	soundLooping  bool
	soundVol      = 100
	soundOnEndFn  object.Object
	soundAlias    = "yodmusic"
	soundOpenPath string
)

func resolveMediaPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	base := AppBaseDir()
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, path))
}

// soundStopAll — נקרא בסגירת חלון ראשי.
func soundStopAll() {
	_ = soundStop(nil)
}
