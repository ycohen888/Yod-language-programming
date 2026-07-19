package stdlib

import "testing"

func TestMatchTitlePattern(t *testing.T) {
	tests := []struct{ pat, title string; want bool }{
		{"Notepad*", "Notepad", true},
		{"*pad", "Notepad", true},
		{"*pad*", "My Notepad App", true},
		{"Exact", "Exact", true},
		{"Foo", "Bar", false},
		{"*", "Something", true},
	}
	for _, tc := range tests {
		if got := matchTitlePattern(tc.pat, tc.title); got != tc.want {
			t.Errorf("matchTitlePattern(%q,%q)=%v want %v", tc.pat, tc.title, got, tc.want)
		}
	}
}
