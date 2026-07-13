//go:build !windows

package stdlib

import "yod/internal/object"

func mouseUnsupported(name string) object.Object {
	return errObj(name + " זמין רק ב־Windows")
}

func mousePosition(args ...object.Object) object.Object {
	return mouseUnsupported("עכבר.מיקום")
}

func mouseLeftDown(args ...object.Object) object.Object {
	return mouseUnsupported("עכבר.שמאל_לחוץ")
}

func mouseRightDown(args ...object.Object) object.Object {
	return mouseUnsupported("עכבר.ימין_לחוץ")
}

func mouseMiddleDown(args ...object.Object) object.Object {
	return mouseUnsupported("עכבר.אמצע_לחוץ")
}

func mouseState(args ...object.Object) object.Object {
	return mouseUnsupported("עכבר.מצב")
}
