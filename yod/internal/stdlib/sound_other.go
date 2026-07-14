//go:build !windows

package stdlib

import "yod/internal/object"

func soundUnsupported(name string) object.Object {
	return errObj(name + " זמין רק ב־Windows")
}

func soundPlayEffect(args ...object.Object) object.Object { return soundUnsupported("שמע.נגן_אפקט") }
func soundPlay(args ...object.Object) object.Object         { return soundUnsupported("שמע.נגן") }
func soundPause(args ...object.Object) object.Object        { return soundUnsupported("שמע.השהה") }
func soundResume(args ...object.Object) object.Object       { return soundUnsupported("שמע.המשך") }
func soundStop(args ...object.Object) object.Object         { return &object.Null{} }
func soundLoop(args ...object.Object) object.Object         { return soundUnsupported("שמע.לולאה") }
func soundVolume(args ...object.Object) object.Object       { return soundUnsupported("שמע.עוצמה") }
func soundIsPlaying(args ...object.Object) object.Object {
	return &object.Boolean{Value: false}
}
func soundOnEnd(args ...object.Object) object.Object { return soundUnsupported("שמע.בעת_סיום") }
