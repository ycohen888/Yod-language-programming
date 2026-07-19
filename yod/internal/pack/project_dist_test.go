package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yod/internal/project"
)

func TestRewriteSiblingIncludes(t *testing.T) {
	in := []byte(`כלול "..\עיצוב\עיצוב_הכל.יוד"` + "\n" + `תיק + "\\..\\עיצוב"`)
	out := string(rewriteSiblingIncludes(in))
	if strings.Contains(out, `..\עיצוב`) || strings.Contains(out, `\\..\\עיצוב`) {
		t.Fatalf("still has parent design path: %s", out)
	}
	if !strings.Contains(out, `עיצוב\עיצוב_הכל`) {
		t.Fatalf("missing local design include: %s", out)
	}
}

func TestPrepareCleanProjectDistSkipsData(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "גיבוי")
	_ = os.MkdirAll(root, 0755)
	_ = os.WriteFile(filepath.Join(root, project.MainFileName), []byte("כלול \"..\\עיצוב\\x.יוד\"\nהדפס: 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(root, "עזר.יוד"), []byte("הדפס: 2\n"), 0644)
	_ = os.WriteFile(filepath.Join(root, "בדיקה_סנכרון.יוד"), []byte("הדפס: בדיקה\n"), 0644)
	dataDir := filepath.Join(root, "נתונים")
	_ = os.MkdirAll(filepath.Join(dataDir, "מצב"), 0755)
	_ = os.WriteFile(filepath.Join(dataDir, "תוכניות.json"), []byte(`{"a":1}`), 0644)

	design := filepath.Join(parent, "עיצוב")
	_ = os.MkdirAll(design, 0755)
	_ = os.WriteFile(filepath.Join(design, "קליפה.html"), []byte("<html></html>"), 0644)

	dist := filepath.Join(root, project.DistDirName)
	_ = os.MkdirAll(filepath.Join(dist, "נתונים"), 0755)
	_ = os.WriteFile(filepath.Join(dist, "נתונים", "ישן.json"), []byte("{}"), 0644)

	if err := PrepareCleanProjectDist(filepath.Join(root, project.MainFileName), dist); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dist, "נתונים")); !os.IsNotExist(err) {
		t.Fatalf("נתונים should be removed from dist, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dist, "עזר.יוד")); err != nil {
		t.Fatal("expected helper source copied")
	}
	if _, err := os.Stat(filepath.Join(dist, "בדיקה_סנכרון.יוד")); !os.IsNotExist(err) {
		t.Fatal("test source should not be packed")
	}
	if _, err := os.Stat(filepath.Join(dist, "עיצוב", "קליפה.html")); err != nil {
		t.Fatal("expected design assets copied")
	}
	mainData, _ := os.ReadFile(filepath.Join(dist, project.MainFileName))
	if strings.Contains(string(mainData), `..\עיצוב`) {
		t.Fatalf("packed main still has parent include: %s", mainData)
	}
}

func TestPrepareCleanProjectDistCopiesGuideAndIcon(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "גיבוי")
	_ = os.MkdirAll(filepath.Join(root, "מדריך"), 0755)
	_ = os.MkdirAll(filepath.Join(root, "נכסים"), 0755)
	_ = os.WriteFile(filepath.Join(root, project.MainFileName), []byte("הדפס: 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(root, "מדריך", "מדריך.html"), []byte("<html>מדריך</html>"), 0644)
	_ = os.WriteFile(filepath.Join(root, "נכסים", "לוגו.b64"), []byte("abc"), 0644)
	_ = os.WriteFile(filepath.Join(root, "app.ico"), []byte("ico"), 0644)

	dist := filepath.Join(root, project.DistDirName)
	if err := PrepareCleanProjectDist(filepath.Join(root, project.MainFileName), dist); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("מדריך", "מדריך.html"),
		filepath.Join("נכסים", "לוגו.b64"),
		"app.ico",
	} {
		if _, err := os.Stat(filepath.Join(dist, rel)); err != nil {
			t.Fatalf("expected asset %s in dist: %v", rel, err)
		}
	}
}
