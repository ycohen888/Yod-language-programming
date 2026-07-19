//go:build windows

package gpu

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

type pixelFormatDescriptor struct {
	nSize           uint16
	nVersion        uint16
	dwFlags         uint32
	iPixelType      byte
	cColorBits      byte
	cRedBits        byte
	cRedShift       byte
	cGreenBits      byte
	cGreenShift     byte
	cBlueBits       byte
	cBlueShift      byte
	cAlphaBits      byte
	cAlphaShift     byte
	cAccumBits      byte
	cAccumRedBits   byte
	cAccumGreenBits byte
	cAccumBlueBits  byte
	cAccumAlphaBits byte
	cDepthBits      byte
	cStencilBits    byte
	cAuxBuffers     byte
	iLayerType      byte
	bReserved       byte
	dwLayerMask     uint32
	dwVisibleMask   uint32
	dwDamageMask    uint32
}

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procGetDC             = user32.NewProc("GetDC")
	procReleaseDC         = user32.NewProc("ReleaseDC")
	procChoosePixelFormat = gdi32.NewProc("ChoosePixelFormat")
	procSetPixelFormat    = gdi32.NewProc("SetPixelFormat")
	procSwapBuffers       = gdi32.NewProc("SwapBuffers")

	glInited bool

	wglCreateContext func(hdc uintptr) uintptr
	wglMakeCurrent   func(hdc, hrc uintptr) int32
	wglDeleteContext func(hrc uintptr) int32

	glClearColor     func(r, g, b, a float32)
	glClear          func(mask uint32)
	glViewport       func(x, y, w, h int32)
	glEnable         func(cap uint32)
	glDisable        func(cap uint32)
	glDepthFunc      func(fn uint32)
	glMatrixMode     func(mode uint32)
	glLoadIdentity   func()
	glLoadMatrixf    func(m *float32)
	glBegin          func(mode uint32)
	glEnd            func()
	glColor3f        func(r, g, b float32)
	glVertex3f       func(x, y, z float32)
	glNormal3f       func(x, y, z float32)
	glTexCoord2f     func(u, v float32)
	glRotatef        func(angle, x, y, z float32)
	glLightfv        func(light, pname uint32, params *float32)
	glColorMaterial  func(face, mode uint32)
	glGenTextures    func(n int32, textures *uint32)
	glBindTexture    func(target uint32, texture uint32)
	glTexImage2D     func(target uint32, level int32, internalformat int32, width, height, border int32, format, typ uint32, pixels unsafe.Pointer)
	glTexParameteri  func(target, pname uint32, param int32)
	glDeleteTextures func(n int32, textures *uint32)
)

func ensureGLFuncs() error {
	if glInited {
		return nil
	}
	handle, err := syscall.LoadLibrary("opengl32.dll")
	if err != nil {
		return fmt.Errorf("טעינת opengl32 נכשלה: %w", err)
	}
	lib := uintptr(handle)
	purego.RegisterLibFunc(&wglCreateContext, lib, "wglCreateContext")
	purego.RegisterLibFunc(&wglMakeCurrent, lib, "wglMakeCurrent")
	purego.RegisterLibFunc(&wglDeleteContext, lib, "wglDeleteContext")
	purego.RegisterLibFunc(&glClearColor, lib, "glClearColor")
	purego.RegisterLibFunc(&glClear, lib, "glClear")
	purego.RegisterLibFunc(&glViewport, lib, "glViewport")
	purego.RegisterLibFunc(&glEnable, lib, "glEnable")
	purego.RegisterLibFunc(&glDisable, lib, "glDisable")
	purego.RegisterLibFunc(&glDepthFunc, lib, "glDepthFunc")
	purego.RegisterLibFunc(&glMatrixMode, lib, "glMatrixMode")
	purego.RegisterLibFunc(&glLoadIdentity, lib, "glLoadIdentity")
	purego.RegisterLibFunc(&glLoadMatrixf, lib, "glLoadMatrixf")
	purego.RegisterLibFunc(&glBegin, lib, "glBegin")
	purego.RegisterLibFunc(&glEnd, lib, "glEnd")
	purego.RegisterLibFunc(&glColor3f, lib, "glColor3f")
	purego.RegisterLibFunc(&glVertex3f, lib, "glVertex3f")
	purego.RegisterLibFunc(&glNormal3f, lib, "glNormal3f")
	purego.RegisterLibFunc(&glTexCoord2f, lib, "glTexCoord2f")
	purego.RegisterLibFunc(&glRotatef, lib, "glRotatef")
	purego.RegisterLibFunc(&glLightfv, lib, "glLightfv")
	purego.RegisterLibFunc(&glColorMaterial, lib, "glColorMaterial")
	purego.RegisterLibFunc(&glGenTextures, lib, "glGenTextures")
	purego.RegisterLibFunc(&glBindTexture, lib, "glBindTexture")
	purego.RegisterLibFunc(&glTexImage2D, lib, "glTexImage2D")
	purego.RegisterLibFunc(&glTexParameteri, lib, "glTexParameteri")
	purego.RegisterLibFunc(&glDeleteTextures, lib, "glDeleteTextures")
	glInited = true
	return nil
}

type openGLBackend struct {
	hwnd syscall.Handle
	hdc  syscall.Handle
	hrc  syscall.Handle
}

func newOpenGLBackend() Backend {
	return &openGLBackend{}
}

func (b *openGLBackend) Name() string { return "OpenGL" }

func (b *openGLBackend) Init(hwnd uintptr) error {
	if err := ensureGLFuncs(); err != nil {
		return err
	}
	b.hwnd = syscall.Handle(hwnd)
	r, _, err := procGetDC.Call(uintptr(b.hwnd))
	if r == 0 {
		return fmt.Errorf("GetDC נכשל: %v", err)
	}
	b.hdc = syscall.Handle(r)

	pfd := pixelFormatDescriptor{
		nSize:        uint16(unsafe.Sizeof(pixelFormatDescriptor{})),
		nVersion:     1,
		dwFlags:      pfdDrawToWindow | pfdSupportOpenGL | pfdDoubleBuffer,
		iPixelType:   pfdTypeRGBA,
		cColorBits:   32,
		cDepthBits:   24,
		cStencilBits: 8,
		iLayerType:   pfdMainPlane,
	}
	fmtIdx, _, err := procChoosePixelFormat.Call(uintptr(b.hdc), uintptr(unsafe.Pointer(&pfd)))
	if fmtIdx == 0 {
		_ = b.releaseDC()
		return fmt.Errorf("ChoosePixelFormat נכשל: %v", err)
	}
	ok, _, err := procSetPixelFormat.Call(uintptr(b.hdc), fmtIdx, uintptr(unsafe.Pointer(&pfd)))
	if ok == 0 {
		_ = b.releaseDC()
		return fmt.Errorf("SetPixelFormat נכשל: %v", err)
	}
	rc := wglCreateContext(uintptr(b.hdc))
	if rc == 0 {
		_ = b.releaseDC()
		return fmt.Errorf("wglCreateContext נכשל")
	}
	b.hrc = syscall.Handle(rc)
	if err := b.MakeCurrent(); err != nil {
		b.Destroy()
		return err
	}
	glEnable(cDepthTest)
	glDepthFunc(cLess)
	glEnable(cCullFace)
	return nil
}

func (b *openGLBackend) MakeCurrent() error {
	if wglMakeCurrent(uintptr(b.hdc), uintptr(b.hrc)) == 0 {
		return fmt.Errorf("wglMakeCurrent נכשל")
	}
	return nil
}

func (b *openGLBackend) Clear(r, g, bl, a float32) {
	glClearColor(r, g, bl, a)
	glClear(cColorBufferBit | cDepthBufferBit)
}

func (b *openGLBackend) Viewport(w, h int) {
	glViewport(0, 0, int32(w), int32(h))
}

func (b *openGLBackend) DrawColoredTriangle(angleRad float64) {
	glDisable(cLighting)
	glDisable(cTexture2D)
	glMatrixMode(cProjection)
	glLoadIdentity()
	glMatrixMode(cModelview)
	glLoadIdentity()
	deg := float32(angleRad * 180 / 3.141592653589793)
	glRotatef(deg, 0, 1, 0)
	glBegin(cTriangles)
	glColor3f(1, 0.35, 0.2)
	glVertex3f(0, 0.75, 0)
	glColor3f(0.2, 0.75, 1)
	glVertex3f(-0.75, -0.65, 0)
	glColor3f(0.35, 1, 0.4)
	glVertex3f(0.75, -0.65, 0)
	glEnd()
}

func (b *openGLBackend) DrawScene(cam Camera, light Light, meshes []DrawMesh, aspect float32) {
	glEnable(cDepthTest)
	glEnable(cLighting)
	glEnable(cLight0)
	glEnable(cColorMaterial)
	glColorMaterial(cFrontAndBack, cAmbientAndDiffuse)

	pos := [4]float32{-light.Direction.X, -light.Direction.Y, -light.Direction.Z, 0}
	diff := [4]float32{light.Color[0], light.Color[1], light.Color[2], 1}
	amb := [4]float32{light.Ambient[0], light.Ambient[1], light.Ambient[2], 1}
	glLightfv(cLight0, cPosition, &pos[0])
	glLightfv(cLight0, cDiffuse, &diff[0])
	glLightfv(cLight0, cAmbient, &amb[0])

	proj := Perspective(cam.FOV, aspect, cam.Near, cam.Far)
	view := LookAt(cam.Eye, cam.Target, cam.Up)

	glMatrixMode(cProjection)
	glLoadMatrixf(&proj[0])
	glMatrixMode(cModelview)

	for _, m := range meshes {
		mv := view.Mul(m.Model)
		glLoadMatrixf(&mv[0])
		glColor3f(m.Color[0], m.Color[1], m.Color[2])
		if m.Texture != 0 {
			glEnable(cTexture2D)
			glBindTexture(cTexture2D, uint32(m.Texture))
		} else {
			glDisable(cTexture2D)
		}
		drawMeshImmediate(m)
	}
	glDisable(cLighting)
	glDisable(cTexture2D)
}

func drawMeshImmediate(m DrawMesh) {
	useUV := len(m.UVs) >= len(m.Positions)/3*2
	useN := len(m.Normals) >= len(m.Positions)
	if len(m.Indices) > 0 {
		glBegin(cTriangles)
		for _, idx := range m.Indices {
			i := int(idx)
			if useN {
				base := i * 3
				if base+2 < len(m.Normals) {
					glNormal3f(m.Normals[base], m.Normals[base+1], m.Normals[base+2])
				}
			}
			if useUV {
				base := i * 2
				if base+1 < len(m.UVs) {
					glTexCoord2f(m.UVs[base], m.UVs[base+1])
				}
			}
			base := i * 3
			if base+2 < len(m.Positions) {
				glVertex3f(m.Positions[base], m.Positions[base+1], m.Positions[base+2])
			}
		}
		glEnd()
		return
	}
	n := len(m.Positions) / 3
	glBegin(cTriangles)
	for i := 0; i < n; i++ {
		if useN {
			base := i * 3
			glNormal3f(m.Normals[base], m.Normals[base+1], m.Normals[base+2])
		}
		if useUV {
			base := i * 2
			if base+1 < len(m.UVs) {
				glTexCoord2f(m.UVs[base], m.UVs[base+1])
			}
		}
		base := i * 3
		glVertex3f(m.Positions[base], m.Positions[base+1], m.Positions[base+2])
	}
	glEnd()
}

func (b *openGLBackend) Swap() error {
	ok, _, err := procSwapBuffers.Call(uintptr(b.hdc))
	if ok == 0 {
		return fmt.Errorf("SwapBuffers נכשל: %v", err)
	}
	return nil
}

func (b *openGLBackend) LoadTextureRGBA(pix []byte, w, h int) (TextureID, error) {
	if w <= 0 || h <= 0 || len(pix) < w*h*4 {
		return 0, fmt.Errorf("נתוני טקסטורה לא תקינים")
	}
	var id uint32
	glGenTextures(1, &id)
	glBindTexture(cTexture2D, id)
	glTexParameteri(cTexture2D, cTextureMinFilter, cLinear)
	glTexParameteri(cTexture2D, cTextureMagFilter, cLinear)
	glTexParameteri(cTexture2D, cTextureWrapS, cRepeat)
	glTexParameteri(cTexture2D, cTextureWrapT, cRepeat)
	glTexImage2D(cTexture2D, 0, cRGBA, int32(w), int32(h), 0, cRGBA, cUnsignedByte, unsafe.Pointer(&pix[0]))
	return TextureID(id), nil
}

func (b *openGLBackend) DeleteTexture(id TextureID) {
	if id == 0 {
		return
	}
	v := uint32(id)
	glDeleteTextures(1, &v)
}

func (b *openGLBackend) releaseDC() error {
	if b.hdc != 0 && b.hwnd != 0 {
		procReleaseDC.Call(uintptr(b.hwnd), uintptr(b.hdc))
		b.hdc = 0
	}
	return nil
}

func (b *openGLBackend) Destroy() {
	if b.hrc != 0 {
		wglMakeCurrent(0, 0)
		wglDeleteContext(uintptr(b.hrc))
		b.hrc = 0
	}
	_ = b.releaseDC()
}
