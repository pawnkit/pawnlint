package openmp

import "testing"

func TestFormatArgumentCount(t *testing.T) {
	tests := []struct {
		format string
		count  int
		ok     bool
	}{
		{format: "plain", count: 0, ok: true},
		{format: "%d %s %.2f", count: 3, ok: true},
		{format: "%05d %10s %% %q", count: 3, ok: true},
		{format: "%%%d", count: 1, ok: true},
		{format: "%e", ok: false},
		{format: "%", ok: false},
	}
	for _, test := range tests {
		count, ok := formatArgumentCount(test.format)
		if count != test.count || ok != test.ok {
			t.Errorf("formatArgumentCount(%q) = %d, %t", test.format, count, ok)
		}
	}
}
