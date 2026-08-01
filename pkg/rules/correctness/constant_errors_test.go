package correctness_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/ruletest"
)

func TestDivisionByZeroEvaluatesKnownLocals(t *testing.T) {
	src := []byte("main() {\nnew zero = 0;\nnew invalid = 1 / zero;\nnew unknown = 1 / read_value();\n}\n")
	diagnostics := ruletest.RunRule(t, "test.pwn", "division-by-zero", src)
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v, want one known zero", diagnostics)
	}
}
