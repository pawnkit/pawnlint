package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRestrictedSyntaxOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("restricted-syntax")
	if !ok || len(metadata.Options) != 6 {
		t.Fatal("restricted syntax options are missing")
	}
	valid := map[string]any{
		"functions": []any{"LegacyFunction", "open@mpFunction"},
		"natives":   []any{"RestrictedNative"},
		"includes":  []any{"legacy/**"},
		"globals":   true,
		"recursion": true,
		"goto":      true,
	}
	for _, option := range metadata.Options {
		if _, err := lint.NormalizeOption(option, valid[option.Name]); err != nil {
			t.Fatalf("valid %s rejected: %v", option.Name, err)
		}
	}
	invalid := map[string]any{
		"functions": []any{"bad name"},
		"natives":   []any{""},
		"includes":  []any{" "},
		"globals":   "yes",
		"recursion": 1,
		"goto":      []any{},
	}
	for _, option := range metadata.Options {
		if _, err := lint.NormalizeOption(option, invalid[option.Name]); err == nil {
			t.Fatalf("invalid %s accepted", option.Name)
		}
	}
}
