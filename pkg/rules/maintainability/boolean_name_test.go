package maintainability

import "testing"

func TestMatchesBooleanPrefix(t *testing.T) {
	prefixes := []string{"is", "has", "can", "b_"}
	tests := map[string]bool{
		"isReady":  true,
		"is_ready": true,
		"hasValue": true,
		"can2D":    true,
		"b_active": true,
		"island":   false,
		"is":       false,
		"ready":    false,
	}
	for name, expected := range tests {
		if actual := matchesBooleanPrefix(name, prefixes); actual != expected {
			t.Errorf("matchesBooleanPrefix(%q) = %t, want %t", name, actual, expected)
		}
	}
}
