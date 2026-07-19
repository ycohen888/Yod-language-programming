//go:build !windows

package gpu

import "fmt"

type stubBackend struct{}

func newOpenGLBackend() Backend { return &stubBackend{} }

func (b *stubBackend) Name() string { return "stub" }
func (b *stubBackend) Init(hwnd uintptr) error {
	return fmt.Errorf("GPU זמין רק ב־Windows")
}
func (b *stubBackend) MakeCurrent() error { return fmt.Errorf("GPU לא זמין") }
func (b *stubBackend) Clear(r, g, bl, a float32) {}
func (b *stubBackend) Viewport(w, h int) {}
func (b *stubBackend) DrawColoredTriangle(angleRad float64) {}
func (b *stubBackend) DrawScene(cam Camera, light Light, meshes []DrawMesh, aspect float32) {}
func (b *stubBackend) Swap() error { return fmt.Errorf("GPU לא זמין") }
func (b *stubBackend) LoadTextureRGBA(pix []byte, w, h int) (TextureID, error) {
	return 0, fmt.Errorf("GPU לא זמין")
}
func (b *stubBackend) DeleteTexture(id TextureID) {}
func (b *stubBackend) Destroy() {}
