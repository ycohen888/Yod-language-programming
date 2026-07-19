package gpu

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type gltfRoot struct {
	Buffers     []gltfBuffer     `json:"buffers"`
	BufferViews []gltfBufferView `json:"bufferViews"`
	Accessors   []gltfAccessor   `json:"accessors"`
	Meshes      []gltfMesh       `json:"meshes"`
	Nodes       []gltfNode       `json:"nodes"`
	Scenes      []gltfScene      `json:"scenes"`
	Scene       int              `json:"scene"`
}

type gltfBuffer struct {
	ByteLength int    `json:"byteLength"`
	URI        string `json:"uri"`
}

type gltfBufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
}

type gltfAccessor struct {
	BufferView    int    `json:"bufferView"`
	ByteOffset    int    `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
}

type gltfMesh struct {
	Name       string          `json:"name"`
	Primitives []gltfPrimitive `json:"primitives"`
}

type gltfPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
}

type gltfNode struct {
	Name        string     `json:"name"`
	Mesh        *int       `json:"mesh"`
	Translation []float64  `json:"translation"`
	Rotation    []float64  `json:"rotation"`
	Scale       []float64  `json:"scale"`
	Matrix      []float64  `json:"matrix"`
	Children    []int      `json:"children"`
}

type gltfScene struct {
	Nodes []int `json:"nodes"`
}

// LoadGLTF טוען glTF 2.0 מינימלי (POSITION + indices, data URI או קובץ .bin ליד ה־gltf).
func LoadGLTF(path string) (*MeshData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("קריאת glTF: %w", err)
	}
	var root gltfRoot
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("JSON glTF לא תקין: %w", err)
	}
	dir := filepath.Dir(path)
	bufs := make([][]byte, len(root.Buffers))
	for i, buf := range root.Buffers {
		data, err := loadGLTFBuffer(dir, buf.URI, buf.ByteLength)
		if err != nil {
			return nil, err
		}
		bufs[i] = data
	}
	if len(root.Meshes) == 0 || len(root.Meshes[0].Primitives) == 0 {
		return nil, fmt.Errorf("glTF בלי רשתות")
	}
	prim := root.Meshes[0].Primitives[0]
	posAcc, ok := prim.Attributes["POSITION"]
	if !ok {
		return nil, fmt.Errorf("חסר POSITION ב־glTF")
	}
	positions, err := readFloatAccessor(root, bufs, posAcc, 3)
	if err != nil {
		return nil, err
	}
	mesh := &MeshData{
		Name:      root.Meshes[0].Name,
		Positions: positions,
	}
	if mesh.Name == "" {
		mesh.Name = filepath.Base(path)
	}
	if nAcc, ok := prim.Attributes["NORMAL"]; ok {
		if normals, err := readFloatAccessor(root, bufs, nAcc, 3); err == nil {
			mesh.Normals = normals
		}
	}
	if tAcc, ok := prim.Attributes["TEXCOORD_0"]; ok {
		if uvs, err := readFloatAccessor(root, bufs, tAcc, 2); err == nil {
			mesh.UVs = uvs
		}
	}
	if prim.Indices != nil {
		idx, err := readIndexAccessor(root, bufs, *prim.Indices)
		if err != nil {
			return nil, err
		}
		mesh.Indices = idx
	} else {
		n := uint32(len(mesh.Positions) / 3)
		mesh.Indices = make([]uint32, n)
		for i := uint32(0); i < n; i++ {
			mesh.Indices[i] = i
		}
	}
	if len(mesh.Normals) == 0 {
		mesh.Normals = estimateNormals(mesh.Positions, mesh.Indices)
	}
	return mesh, nil
}

func loadGLTFBuffer(dir, uri string, wantLen int) ([]byte, error) {
	if uri == "" {
		return nil, fmt.Errorf("buffer בלי uri")
	}
	if strings.HasPrefix(uri, "data:") {
		parts := strings.SplitN(uri, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("data URI לא תקין")
		}
		raw, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("base64 ב־glTF: %w", err)
		}
		return raw, nil
	}
	p := uri
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, uri)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	if wantLen > 0 && len(raw) < wantLen {
		return nil, fmt.Errorf("קובץ buffer קצר מדי")
	}
	return raw, nil
}

func accessorSlice(root gltfRoot, bufs [][]byte, accIdx int) ([]byte, error) {
	if accIdx < 0 || accIdx >= len(root.Accessors) {
		return nil, fmt.Errorf("accessor מחוץ לטווח")
	}
	acc := root.Accessors[accIdx]
	if acc.BufferView < 0 || acc.BufferView >= len(root.BufferViews) {
		return nil, fmt.Errorf("bufferView מחוץ לטווח")
	}
	bv := root.BufferViews[acc.BufferView]
	if bv.Buffer < 0 || bv.Buffer >= len(bufs) {
		return nil, fmt.Errorf("buffer מחוץ לטווח")
	}
	start := bv.ByteOffset + acc.ByteOffset
	compSize := componentSize(acc.ComponentType)
	nComp := typeComponents(acc.Type)
	need := acc.Count * nComp * compSize
	buf := bufs[bv.Buffer]
	if start+need > len(buf) {
		return nil, fmt.Errorf("accessor חורג מגבולות ה־buffer")
	}
	return buf[start : start+need], nil
}

func readFloatAccessor(root gltfRoot, bufs [][]byte, accIdx, comps int) ([]float32, error) {
	acc := root.Accessors[accIdx]
	if acc.ComponentType != 5126 {
		return nil, fmt.Errorf("accessor לא FLOAT")
	}
	raw, err := accessorSlice(root, bufs, accIdx)
	if err != nil {
		return nil, err
	}
	n := acc.Count * comps
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	return out, nil
}

func readIndexAccessor(root gltfRoot, bufs [][]byte, accIdx int) ([]uint32, error) {
	acc := root.Accessors[accIdx]
	raw, err := accessorSlice(root, bufs, accIdx)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, acc.Count)
	switch acc.ComponentType {
	case 5121: // UNSIGNED_BYTE
		for i := 0; i < acc.Count; i++ {
			out[i] = uint32(raw[i])
		}
	case 5123: // UNSIGNED_SHORT
		for i := 0; i < acc.Count; i++ {
			out[i] = uint32(binary.LittleEndian.Uint16(raw[i*2:]))
		}
	case 5125: // UNSIGNED_INT
		for i := 0; i < acc.Count; i++ {
			out[i] = binary.LittleEndian.Uint32(raw[i*4:])
		}
	default:
		return nil, fmt.Errorf("סוג אינדקס לא נתמך: %d", acc.ComponentType)
	}
	return out, nil
}

func componentSize(ct int) int {
	switch ct {
	case 5120, 5121:
		return 1
	case 5122, 5123:
		return 2
	case 5125, 5126:
		return 4
	default:
		return 4
	}
}

func typeComponents(t string) int {
	switch t {
	case "SCALAR":
		return 1
	case "VEC2":
		return 2
	case "VEC3":
		return 3
	case "VEC4":
		return 4
	default:
		return 3
	}
}

func estimateNormals(pos []float32, indices []uint32) []float32 {
	nVert := len(pos) / 3
	normals := make([]float32, nVert*3)
	for i := 0; i+2 < len(indices); i += 3 {
		i0, i1, i2 := int(indices[i]), int(indices[i+1]), int(indices[i+2])
		a := Vec3{pos[i0*3], pos[i0*3+1], pos[i0*3+2]}
		b := Vec3{pos[i1*3], pos[i1*3+1], pos[i1*3+2]}
		c := Vec3{pos[i2*3], pos[i2*3+1], pos[i2*3+2]}
		n := b.Sub(a).Cross(c.Sub(a)).Normalize()
		for _, vi := range []int{i0, i1, i2} {
			normals[vi*3] += n.X
			normals[vi*3+1] += n.Y
			normals[vi*3+2] += n.Z
		}
	}
	for i := 0; i < nVert; i++ {
		v := Vec3{normals[i*3], normals[i*3+1], normals[i*3+2]}.Normalize()
		normals[i*3], normals[i*3+1], normals[i*3+2] = v.X, v.Y, v.Z
	}
	return normals
}
