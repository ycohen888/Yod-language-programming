package gpu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssetEntry — פריט במניפסט נכסים מקומי.
type AssetEntry struct {
	Name string `json:"שם"`
	Kind string `json:"סוג"` // מודל | תמונה | שמע
	Path string `json:"נתיב"`
}

// AssetManifest — מניפסט JSON בעברית לתיקיית נכסים/.
type AssetManifest struct {
	Assets []AssetEntry `json:"נכסים"`
}

// LoadManifest טוען מניפסט מנתיב קובץ.
func LoadManifest(path string) (*AssetManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("קריאת מניפסט: %w", err)
	}
	var m AssetManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("מניפסט לא תקין: %w", err)
	}
	return &m, nil
}

// SaveManifest שומר מניפסט.
func SaveManifest(path string, m *AssetManifest) error {
	if m == nil {
		m = &AssetManifest{}
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// ResolveAssetPath מחזיר נתיב מלא יחסית לתיקיית המניפסט.
func ResolveAssetPath(manifestPath, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(filepath.Dir(manifestPath), rel)
}

// FindAsset מחפש לפי שם.
func (m *AssetManifest) FindAsset(name string) *AssetEntry {
	if m == nil {
		return nil
	}
	for i := range m.Assets {
		if m.Assets[i].Name == name {
			return &m.Assets[i]
		}
	}
	return nil
}

// ScanAssetsDir סורק תיקייה ובונה מניפסט לפי סיומות.
func ScanAssetsDir(dir string) (*AssetManifest, error) {
	m := &AssetManifest{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		kind := ""
		switch ext {
		case ".gltf", ".glb", ".obj":
			kind = "מודל"
		case ".png", ".jpg", ".jpeg", ".webp":
			kind = "תמונה"
		case ".ogg", ".wav", ".mp3":
			kind = "שמע"
		default:
			continue
		}
		base := strings.TrimSuffix(name, ext)
		m.Assets = append(m.Assets, AssetEntry{
			Name: base,
			Kind: kind,
			Path: name,
		})
	}
	return m, nil
}
