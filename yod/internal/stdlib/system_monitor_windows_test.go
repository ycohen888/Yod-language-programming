//go:build windows

package stdlib

import (
	"testing"

	"yod/internal/object"
)

func TestListOSProcessesNonEmpty(t *testing.T) {
	list, err := listOSProcesses()
	if err != nil {
		t.Fatalf("listOSProcesses: %v", err)
	}
	if len(list) < 5 {
		t.Fatalf("צפוי לפחות כמה תהליכים, קיבל %d", len(list))
	}
	foundNamed := false
	for _, p := range list {
		if p.PID == 0 {
			t.Fatalf("מזהה 0 לא אמור להופיע")
		}
		if p.Name != "" {
			foundNamed = true
		}
	}
	if !foundNamed {
		t.Fatal("לא נמצא תהליך עם שם")
	}
}

func TestTableParseRowSortKeys(t *testing.T) {
	fields := []string{"שם", "מזהה", "זיכרון"}
	row, err := parseTableRow(fields, &object.Hash{Pairs: map[string]object.Object{
		"שם":     &object.String{Value: "a.exe"},
		"מזהה":   &object.Number{Value: 42},
		"זיכרון": &object.String{Value: "12 מגה"},
		"מיון": &object.Hash{Pairs: map[string]object.Object{
			"זיכרון": &object.Number{Value: 12 * 1024 * 1024},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if row.display[0] != "a.exe" || row.display[1] != "42" {
		t.Fatalf("display = %#v", row.display)
	}
	if row.sortKey[2].(float64) != 12*1024*1024 {
		t.Fatalf("sortKey זיכרון = %#v", row.sortKey[2])
	}
}
