package maintainability

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/semantic"
)

func TestDominantEnumPrefix(t *testing.T) {
	tests := []struct {
		names    []string
		expected string
	}{
		{[]string{"PLAYER_STATE_NONE", "PLAYER_STATE_ACTIVE", "PLAYER_STATE_PAUSED", "STATE_UNKNOWN"}, "PLAYER_"},
		{[]string{"pLoggedIn", "pScore", "pHealth", "dataName"}, "p"},
		{[]string{"ModeNone", "ModeRace", "ModeDerby", "Unknown"}, "Mode"},
		{[]string{"RED", "GREEN", "BLUE", "ALPHA"}, ""},
		{[]string{"ITEM_ONE", "ITEM_TWO", "THREE"}, ""},
		{[]string{"ITEM_ONE", "ITEM_TWO", "ITEM_THREE", "ITEM_FOUR"}, ""},
		{[]string{"bpObject", "bpLabel", "bp_X", "bp_Y", "bp_Z"}, ""},
	}
	for _, test := range tests {
		entries := make([]*semantic.Symbol, len(test.names))
		for index, name := range test.names {
			entries[index] = &semantic.Symbol{Name: name}
		}
		if actual := dominantEnumPrefix(entries); actual != test.expected {
			t.Errorf("dominantEnumPrefix(%q) = %q, want %q", test.names, actual, test.expected)
		}
	}
}
