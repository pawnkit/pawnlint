package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestFunctionLengthOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("function-length")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("function length options are missing")
	}
	option := metadata.Options[0]
	if _, err := lint.NormalizeOption(option, 120); err != nil {
		t.Fatalf("valid maximum rejected: %v", err)
	}
	for _, value := range []any{0, 1_000_001, "one hundred"} {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid maximum %v accepted", value)
		}
	}
}
