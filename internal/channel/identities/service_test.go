package identities

import "testing"

func TestNormalizeChannel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feishu", "feishu"},
		{" FEISHU ", "feishu"},
		{"Web", "web"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeChannel(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeChannel(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestToPgText(t *testing.T) {
	value := toPgText("  display  ")
	if !value.Valid {
		t.Fatal("expected valid text for non-empty input")
	}
	if value.String != "display" {
		t.Fatalf("expected trimmed text display, got %q", value.String)
	}
	empty := toPgText(" ")
	if empty.Valid {
		t.Fatal("expected invalid text for empty input")
	}
}
