package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestTooManyGlobalsOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("too-many-globals")
	if !ok || len(metadata.Options) != 2 {
		t.Fatal("too many globals options are missing")
	}
	valid := map[string]any{
		"maximum":           100,
		"include-constants": true,
	}
	invalid := map[string]any{
		"maximum":           0,
		"include-constants": "yes",
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
