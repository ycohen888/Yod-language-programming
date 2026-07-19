package gpu

// TextureID מזהה טקסטורה בצד ה־backend.
type TextureID uint32

// DrawMesh — רשת מוכנה לציור (כבר בטרנספורם מקומי+עולם).
type DrawMesh struct {
	Positions []float32 // x,y,z…
	Normals   []float32 // x,y,z… (אופציונלי, אורך זהה ל־Positions)
	UVs       []float32 // u,v… (אופציונלי)
	Indices   []uint32
	Color     [3]float32
	Texture   TextureID // 0 = בלי טקסטורה
	Model     Mat4
}

// Camera — מצלמת פרספקטיבה.
type Camera struct {
	Eye, Target, Up Vec3
	FOV             float32
	Near, Far       float32
}

func DefaultCamera() Camera {
	return Camera{
		Eye:    Vec3{0, 1.6, 4},
		Target: Vec3{0, 0.4, 0},
		Up:     Vec3{0, 1, 0},
		FOV:    55,
		Near:   0.1,
		Far:    100,
	}
}

// Light — אור כיווני בסיסי.
type Light struct {
	Direction Vec3
	Color     [3]float32
	Ambient   [3]float32
}

func DefaultLight() Light {
	return Light{
		Direction: Vec3{0.35, 1, 0.4}.Normalize(),
		Color:     [3]float32{1, 0.97, 0.9},
		Ambient:   [3]float32{0.22, 0.24, 0.28},
	}
}

// Backend — שכבת הפשטה (OpenGL עכשיו; wgpu בעתיד).
type Backend interface {
	Name() string
	Init(hwnd uintptr) error
	MakeCurrent() error
	Clear(r, g, b, a float32)
	Viewport(w, h int)
	DrawColoredTriangle(angleRad float64)
	DrawScene(cam Camera, light Light, meshes []DrawMesh, aspect float32)
	Swap() error
	LoadTextureRGBA(pix []byte, w, h int) (TextureID, error)
	DeleteTexture(id TextureID)
	Destroy()
}
