package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestIdentifierLengthOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("identifier-length")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("identifier length option is missing")
	}
	option := metadata.Options[0]
	valid := []any{map[string]any{"kinds": []any{"local"}, "minimum": 3, "maximum": 20, "allow-loop-indices": true}}
	if _, err := lint.NormalizeOption(option, valid); err != nil {
		t.Fatal(err)
	}
	invalid := [][]any{
		{map[string]any{"kinds": []any{"unknown"}, "minimum": 3}},
		{map[string]any{"kinds": []any{"local"}}},
		{map[string]any{"minimum": 5, "maximum": 4}},
		{map[string]any{"minimum": 0}},
		{map[string]any{"maximum": 1025}},
		{map[string]any{"minimum": 3, "exclude": []any{"["}}},
		{map[string]any{"minimum": 3, "unknown": true}},
	}
	for _, value := range invalid {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid limit accepted: %#v", value)
		}
	}
}
