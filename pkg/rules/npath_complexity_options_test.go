package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestNPathComplexityOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("npath-complexity")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("NPath complexity options are missing")
	}
	option := metadata.Options[0]
	if _, err := lint.NormalizeOption(option, 500); err != nil {
		t.Fatalf("valid maximum rejected: %v", err)
	}
	for _, value := range []any{0, 1_000_000_000, "two hundred"} {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid maximum %v accepted", value)
		}
	}
}
