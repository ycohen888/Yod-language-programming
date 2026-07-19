package stdlib

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"yod/internal/gpu"
	"yod/internal/object"
)

// NewGPUModule — ספריית GPU מקומית (OpenGL על משטח_GPU).
func NewGPUModule() *object.Module {
	m := &object.Module{Name: "GPU", Attrs: map[string]object.Object{}}
	m.Attrs["סצנה"] = &object.Builtin{Fn: gpuNewScene}
	m.Attrs["קוביה"] = &object.Builtin{Fn: gpuMakeCube}
	m.Attrs["מישור"] = &object.Builtin{Fn: gpuMakePlane}
	m.Attrs["טען_מודל"] = &object.Builtin{Fn: gpuLoadModel}
	m.Attrs["טען_תמונה"] = &object.Builtin{Fn: gpuLoadImage}
	m.Attrs["טען_מניפסט"] = &object.Builtin{Fn: gpuLoadManifest}
	m.Attrs["סרוק_נכסים"] = &object.Builtin{Fn: gpuScanAssets}
	m.Attrs["שמור_מניפסט"] = &object.Builtin{Fn: gpuSaveManifest}
	m.Attrs["ייבא_zip"] = &object.Builtin{Fn: gpuImportZip}
	return m
}

func gpuNewScene(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("GPU.סצנה מצפה ל־0 ארגומנטים")
	}
	sc := gpu.NewScene()
	w := &object.GuiWidget{Kind: "סצנת_GPU", Data: sc, Attrs: map[string]object.Object{}}
	w.Attrs["הוסף"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return gpuSceneAdd(sc, a...)
	}}
	w.Attrs["קבע_מצלמה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return gpuSetCamera(sc, a...)
	}}
	w.Attrs["מצלמת_מסלול"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return gpuOrbit(sc, a...)
	}}
	w.Attrs["קבע_אור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return gpuSetLight(sc, a...)
	}}
	w.Attrs["מצא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("סצנה.מצא מצפה לשם")
		}
		name, ok := asString(a[0])
		if !ok {
			return errObj("סצנה.מצא מצפה למחרוזת")
		}
		n := sc.Find(name)
		if n == nil {
			return object.Nil
		}
		return wrapGPUNode(n)
	}}
	w.Attrs["שמות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		arr := &object.Array{Elements: make([]object.Object, 0, len(sc.Nodes))}
		for _, n := range sc.Nodes {
			arr.Elements = append(arr.Elements, &object.String{Value: n.Name})
		}
		return arr
	}}
	return w
}

func wrapGPUNode(n *gpu.Node) *object.GuiWidget {
	w := &object.GuiWidget{Kind: "צומת_GPU", Data: n, Attrs: map[string]object.Object{}}
	w.Attrs["קבע_מיקום"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, z, err := threeFloats("צומת.קבע_מיקום", a)
		if err != nil {
			return err
		}
		n.Position = gpu.Vec3{X: float32(x), Y: float32(y), Z: float32(z)}
		return object.Nil
	}}
	w.Attrs["קבע_סיבוב"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, z, err := threeFloats("צומת.קבע_סיבוב", a)
		if err != nil {
			return err
		}
		n.Rotation = gpu.Vec3{
			X: float32(x * math.Pi / 180),
			Y: float32(y * math.Pi / 180),
			Z: float32(z * math.Pi / 180),
		}
		return object.Nil
	}}
	w.Attrs["קבע_קנה_מידה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		x, y, z, err := threeFloats("צומת.קבע_קנה_מידה", a)
		if err != nil {
			return err
		}
		n.Scale = gpu.Vec3{X: float32(x), Y: float32(y), Z: float32(z)}
		return object.Nil
	}}
	w.Attrs["קבע_צבע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		r, g, b, err := threeFloats("צומת.קבע_צבע", a)
		if err != nil {
			return err
		}
		n.Color = [3]float32{float32(r), float32(g), float32(b)}
		return object.Nil
	}}
	w.Attrs["קבע_טקסטורה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("צומת.קבע_טקסטורה מצפה למזהה טקסטורה")
		}
		nNum, ok := a[0].(*object.Number)
		if !ok {
			return errObj("צומת.קבע_טקסטורה מצפה למספר מזהה")
		}
		n.Texture = gpu.TextureID(uint32(nNum.Value))
		return object.Nil
	}}
	w.Attrs["קרא_שם"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return &object.String{Value: n.Name}
	}}
	return w
}

func gpuSceneAdd(sc *gpu.Scene, args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("סצנה.הוסף מצפה לרשת [, שם]")
	}
	meshObj, ok := args[0].(*object.GuiWidget)
	if !ok || meshObj.Kind != "רשת_GPU" {
		return errObj("סצנה.הוסף מצפה לרשת מ־GPU.קוביה / מישור / טען_מודל")
	}
	md, ok := meshObj.Data.(*gpu.MeshData)
	if !ok || md == nil {
		return errObj("רשת GPU לא תקינה")
	}
	name := md.Name
	if len(args) == 2 {
		if s, ok := asString(args[1]); ok && s != "" {
			name = s
		}
	}
	node := &gpu.Node{
		Name:     name,
		Mesh:     md,
		Scale:    gpu.Vec3{1, 1, 1},
		Color:    [3]float32{0.75, 0.8, 0.95},
	}
	sc.AddNode(node)
	return wrapGPUNode(node)
}

func wrapMesh(md *gpu.MeshData) *object.GuiWidget {
	return &object.GuiWidget{Kind: "רשת_GPU", Data: md, Attrs: map[string]object.Object{}}
}

func gpuMakeCube(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("GPU.קוביה מצפה ל־0 ארגומנטים")
	}
	return wrapMesh(gpu.NewCubeMesh())
}

func gpuMakePlane(args ...object.Object) object.Object {
	size := 4.0
	if len(args) == 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return errObj("GPU.מישור מצפה למספר אופציונלי")
		}
		size = n.Value
	} else if len(args) > 1 {
		return errObj("GPU.מישור מצפה ל־0 או 1 ארגומנטים")
	}
	return wrapMesh(gpu.NewPlaneMesh(float32(size)))
}

func gpuLoadModel(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("GPU.טען_מודל מצפה לנתיב")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("GPU.טען_מודל מצפה למחרוזת")
	}
	md, err := gpu.LoadGLTF(path)
	if err != nil {
		return errObj(err.Error())
	}
	return wrapMesh(md)
}

func gpuLoadImage(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("GPU.טען_תמונה מצפה לנתיב [, משטח_GPU]")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("GPU.טען_תמונה מצפה לנתיב מחרוזת")
	}
	pix, w, h, err := gpu.LoadImageRGBA(path, true)
	if err != nil {
		return errObj(err.Error())
	}
	if len(args) == 1 {
		// מחזיר מילון עם נתונים — טעינה ל־GPU דורשת משטח מחובר
		hsh := object.NewHash()
		hsh.Set("רוחב", &object.Number{Value: float64(w)})
		hsh.Set("גובה", &object.Number{Value: float64(h)})
		hsh.Set("נתיב", &object.String{Value: path})
		return hsh
	}
	surf, err := surfaceFromWidget(args[1])
	if err != nil {
		return errObj(err.Error())
	}
	id, err := surf.LoadTextureRGBA(pix, w, h)
	if err != nil {
		return errObj(err.Error())
	}
	return &object.Number{Value: float64(id)}
}

func gpuSetCamera(sc *gpu.Scene, args ...object.Object) object.Object {
	if len(args) != 6 {
		return errObj("סצנה.קבע_מצלמה מצפה ל־eye(x,y,z) ו־target(x,y,z)")
	}
	vals := make([]float64, 6)
	for i := 0; i < 6; i++ {
		n, ok := args[i].(*object.Number)
		if !ok {
			return errObj("קבע_מצלמה מצפה למספרים")
		}
		vals[i] = n.Value
	}
	sc.Camera.Eye = gpu.Vec3{X: float32(vals[0]), Y: float32(vals[1]), Z: float32(vals[2])}
	sc.Camera.Target = gpu.Vec3{X: float32(vals[3]), Y: float32(vals[4]), Z: float32(vals[5])}
	return object.Nil
}

func gpuOrbit(sc *gpu.Scene, args ...object.Object) object.Object {
	if len(args) != 3 {
		return errObj("סצנה.מצלמת_מסלול מצפה ל־yaw, pitch, מרחק (מעלות)")
	}
	yaw, ok1 := args[0].(*object.Number)
	pitch, ok2 := args[1].(*object.Number)
	dist, ok3 := args[2].(*object.Number)
	if !ok1 || !ok2 || !ok3 {
		return errObj("מצלמת_מסלול מצפה למספרים")
	}
	gpu.OrbitCamera(&sc.Camera, yaw.Value, pitch.Value, dist.Value)
	return object.Nil
}

func gpuSetLight(sc *gpu.Scene, args ...object.Object) object.Object {
	if len(args) != 3 {
		return errObj("סצנה.קבע_אור מצפה לכיוון x,y,z")
	}
	x, y, z, err := threeFloats("קבע_אור", args)
	if err != nil {
		return err
	}
	sc.Light.Direction = gpu.Vec3{X: float32(x), Y: float32(y), Z: float32(z)}.Normalize()
	return object.Nil
}

func threeFloats(name string, args []object.Object) (float64, float64, float64, *object.Error) {
	if len(args) != 3 {
		return 0, 0, 0, errObj(name + " מצפה ל־3 מספרים")
	}
	a, ok1 := args[0].(*object.Number)
	b, ok2 := args[1].(*object.Number)
	c, ok3 := args[2].(*object.Number)
	if !ok1 || !ok2 || !ok3 {
		return 0, 0, 0, errObj(name + " מצפה למספרים")
	}
	return a.Value, b.Value, c.Value, nil
}

func gpuLoadManifest(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("GPU.טען_מניפסט מצפה לנתיב")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("GPU.טען_מניפסט מצפה למחרוזת")
	}
	man, err := gpu.LoadManifest(path)
	if err != nil {
		return errObj(err.Error())
	}
	return manifestToHash(path, man)
}

func gpuScanAssets(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("GPU.סרוק_נכסים מצפה לתיקייה")
	}
	dir, ok := asString(args[0])
	if !ok {
		return errObj("GPU.סרוק_נכסים מצפה למחרוזת")
	}
	man, err := gpu.ScanAssetsDir(dir)
	if err != nil {
		return errObj(err.Error())
	}
	manPath := filepath.Join(dir, "מניפסט.json")
	return manifestToHash(manPath, man)
}

func gpuSaveManifest(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("GPU.שמור_מניפסט מצפה לנתיב ולמילון מניפסט")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("נתיב חייב להיות מחרוזת")
	}
	h, ok := args[1].(*object.Hash)
	if !ok {
		return errObj("מניפסט חייב להיות מילון")
	}
	man := hashToManifest(h)
	if err := gpu.SaveManifest(path, man); err != nil {
		return errObj(err.Error())
	}
	return object.Nil
}

func gpuImportZip(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("GPU.ייבא_zip מצפה לנתיב ZIP ולתיקיית יעד")
	}
	zipPath, ok1 := asString(args[0])
	dest, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("GPU.ייבא_zip מצפה למחרוזות")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return errObj(fmt.Sprintf("יצירת תיקייה: %v", err))
	}
	out := filesExtract(&object.String{Value: zipPath}, &object.String{Value: dest})
	if err, ok := out.(*object.Error); ok {
		return err
	}
	man, err := gpu.ScanAssetsDir(dest)
	if err != nil {
		return errObj(err.Error())
	}
	manPath := filepath.Join(dest, "מניפסט.json")
	_ = gpu.SaveManifest(manPath, man)
	return manifestToHash(manPath, man)
}

func manifestToHash(manPath string, man *gpu.AssetManifest) *object.Hash {
	h := object.NewHash()
	h.Set("נתיב", &object.String{Value: manPath})
	arr := &object.Array{Elements: []object.Object{}}
	for _, e := range man.Assets {
		item := object.NewHash()
		item.Set("שם", &object.String{Value: e.Name})
		item.Set("סוג", &object.String{Value: e.Kind})
		item.Set("נתיב", &object.String{Value: gpu.ResolveAssetPath(manPath, e.Path)})
		arr.Elements = append(arr.Elements, item)
	}
	h.Set("נכסים", arr)
	return h
}

func hashToManifest(h *object.Hash) *gpu.AssetManifest {
	man := &gpu.AssetManifest{}
	if arr, ok := h.Pairs["נכסים"].(*object.Array); ok {
		for _, el := range arr.Elements {
			item, ok := el.(*object.Hash)
			if !ok {
				continue
			}
			e := gpu.AssetEntry{}
			if s, ok := item.Pairs["שם"].(*object.String); ok {
				e.Name = s.Value
			}
			if s, ok := item.Pairs["סוג"].(*object.String); ok {
				e.Kind = s.Value
			}
			if s, ok := item.Pairs["נתיב"].(*object.String); ok {
				e.Path = s.Value
			}
			man.Assets = append(man.Assets, e)
		}
	}
	return man
}
