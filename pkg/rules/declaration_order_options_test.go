package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestDeclarationOrderOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("declaration-order")
	if !ok || len(metadata.Options) != 2 {
		t.Fatal("declaration order options are missing")
	}
	valid := map[string]any{
		"order":                    []any{"include", "enum", "variable", "function"},
		"locals-before-statements": true,
	}
	invalid := map[string]any{
		"order":                    []any{"include", "include"},
		"locals-before-statements": "yes",
	}
	for _, option := range metadata.Options {
		if _, err := lint.NormalizeOption(option, valid[option.Name]); err != nil {
			t.Fatalf("valid %s rejected: %v", option.Name, err)
		}
		if _, err := lint.NormalizeOption(option, invalid[option.Name]); err == nil {
			t.Fatalf("invalid %s accepted", option.Name)
		}
	}
}
