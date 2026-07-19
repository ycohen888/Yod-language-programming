//go:build !windows

package stdlib

import (
	"fmt"

	"yod/internal/gpu"
	"yod/internal/object"
)

func winCreateGPUSurface(args ...object.Object) object.Object {
	return errObj("חלונות.משטח_GPU זמין רק ב־Windows")
}

func surfaceFromWidget(obj object.Object) (*gpu.Surface, error) {
	return nil, fmt.Errorf("משטח_GPU זמין רק ב־Windows")
}

func sceneFromObject(obj object.Object) (*gpu.Scene, error) {
	return nil, fmt.Errorf("סצנת GPU זמינה רק ב־Windows")
}
