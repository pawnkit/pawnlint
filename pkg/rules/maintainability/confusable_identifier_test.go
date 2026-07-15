package maintainability

import "testing"

func TestConfusableSkeleton(t *testing.T) {
	tests := map[string]string{
		"PlayerO": "P1ayer0",
		"Playero": "P1ayer0",
		"SlotI":   "S10t1",
		"Slotl":   "S10t1",
		"plain":   "p1ain",
		"value_2": "va1ue_2",
	}
	for input, expected := range tests {
		if actual := confusableSkeleton(input); actual != expected {
			t.Errorf("confusableSkeleton(%q) = %q, want %q", input, actual, expected)
		}
	}
}
