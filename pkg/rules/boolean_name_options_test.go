package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestBooleanNameOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("boolean-name")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("boolean name option is missing")
	}
	option := metadata.Options[0]
	valid := []any{map[string]any{"kinds": []any{"local", "parameter"}, "prefixes": []any{"is", "has", "can", "b_"}}}
	if _, err := lint.NormalizeOption(option, valid); err != nil {
		t.Fatal(err)
	}
	invalid := [][]any{
		{map[string]any{"kinds": []any{"enum-entry"}, "prefixes": []any{"is"}}},
		{map[string]any{"kinds": []any{"local"}}},
		{map[string]any{"prefixes": []any{}}},
		{map[string]any{"prefixes": []any{"bad prefix"}}},
		{map[string]any{"prefixes": []any{"is"}, "exclude": []any{"["}}},
		{map[string]any{"prefixes": []any{"is"}, "unknown": true}},
	}
	for _, value := range invalid {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid policy accepted: %#v", value)
		}
	}
}
