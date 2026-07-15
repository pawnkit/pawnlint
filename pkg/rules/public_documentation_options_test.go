package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestPublicDocumentationOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("public-documentation")
	if !ok || len(metadata.Options) != 6 {
		t.Fatal("public documentation options are missing")
	}
	valid := map[string]any{
		"storage":                    []any{"public", "stock"},
		"include":                    []any{"^API_"},
		"exclude":                    []any{"Internal$"},
		"minimum-description-length": 20,
		"require-parameters":         true,
		"require-return":             true,
	}
	invalid := map[string]any{
		"storage":                    []any{},
		"include":                    []any{"["},
		"exclude":                    []any{"("},
		"minimum-description-length": 0,
		"require-parameters":         "yes",
		"require-return":             1,
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
