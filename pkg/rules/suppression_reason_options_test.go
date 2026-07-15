package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestSuppressionReasonOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("suppression-reason")
	if !ok || len(metadata.Options) != 2 {
		t.Fatal("suppression reason options are missing")
	}
	valid := map[string]any{"minimum-length": 10, "pattern": `(?:[A-Z]+-[0-9]+|because)`}
	invalid := map[string]any{"minimum-length": 0, "pattern": "["}
	for _, option := range metadata.Options {
		if _, err := lint.NormalizeOption(option, valid[option.Name]); err != nil {
			t.Fatalf("valid %s rejected: %v", option.Name, err)
		}
		if _, err := lint.NormalizeOption(option, invalid[option.Name]); err == nil {
			t.Fatalf("invalid %s accepted", option.Name)
		}
	}
}
