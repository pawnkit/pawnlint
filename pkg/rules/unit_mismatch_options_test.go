package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestUnitMismatchOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("unit-mismatch")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("unit mismatch option is missing")
	}
	option := metadata.Options[0]
	valid := []any{
		map[string]any{"name": "milliseconds", "tags": []any{"Milliseconds", "DurationMs"}},
		map[string]any{"name": "seconds", "tags": []any{"Seconds"}},
	}
	if _, err := lint.NormalizeOption(option, valid); err != nil {
		t.Fatal(err)
	}
	invalid := [][]any{
		{map[string]any{"name": "", "tags": []any{"Milliseconds"}}},
		{map[string]any{"name": "milliseconds", "tags": []any{}}},
		{map[string]any{"name": "milliseconds", "tags": []any{"Milliseconds"}}, map[string]any{"name": "milliseconds", "tags": []any{"DurationMs"}}},
		{map[string]any{"name": "milliseconds", "tags": []any{"Milliseconds"}}, map[string]any{"name": "seconds", "tags": []any{"Milliseconds"}}},
	}
	for _, value := range invalid {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid units accepted: %#v", value)
		}
	}
}
