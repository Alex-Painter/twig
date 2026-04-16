package worktree

import "testing"

func TestIsPRReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"#123", true},
		{"#1", true},
		{"#99999", true},
		{"feature-branch", false},
		{"123", false},
		{"#", false},
		{"#abc", false},
		{"", false},
		{"fix-#123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsPRReference(tt.input); got != tt.want {
				t.Errorf("IsPRReference(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
