package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestNamingConventionOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("naming-convention")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("naming convention option is missing")
	}
	option := metadata.Options[0]
	valid := []any{map[string]any{"kinds": []any{"function"}, "case": "PascalCase", "exclude": []any{"^main$"}}}
	if _, err := lint.NormalizeOption(option, valid); err != nil {
		t.Fatal(err)
	}
	invalid := [][]any{
		{map[string]any{"kinds": []any{"unknown"}, "case": "PascalCase"}},
		{map[string]any{"pattern": "["}},
		{map[string]any{"exclude": []any{"["}, "case": "camelCase"}},
		{map[string]any{"kinds": []any{"function"}}},
		{map[string]any{"unknown": true, "case": "camelCase"}},
	}
	for _, value := range invalid {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid convention accepted: %#v", value)
		}
	}
}

func TestDisallowedNameOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("disallowed-name")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("disallowed name option is missing")
	}
	option := metadata.Options[0]
	valid := []any{map[string]any{"kinds": []any{"local"}, "names": []any{"foo"}, "patterns": []any{"^temp_"}}}
	if _, err := lint.NormalizeOption(option, valid); err != nil {
		t.Fatal(err)
	}
	invalid := [][]any{
		{map[string]any{"kinds": []any{"unknown"}, "names": []any{"foo"}}},
		{map[string]any{"patterns": []any{"["}}},
		{map[string]any{"patterns": []any{""}}},
		{map[string]any{"names": []any{"bad name"}}},
		{map[string]any{"kinds": []any{"function"}}},
		{map[string]any{"unknown": true, "names": []any{"foo"}}},
	}
	for _, value := range invalid {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid policy accepted: %#v", value)
		}
	}
}
