package correctness_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/ruletest"
)

func TestConstantConditionSkipsInactiveCode(t *testing.T) {
	source := []byte("#if 0\nmain() { if (1) {} }\n#endif\n")
	if diagnostics := ruletest.RunRule(t, "test.pwn", "constant-condition", source); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v, want none for inactive code", diagnostics)
	}
}

func TestConstantConditionIgnoresCall(t *testing.T) {
	source := []byte("main() { if (IsPlayerConnected(playerid)) {} }\n")
	if diagnostics := ruletest.RunRule(t, "test.pwn", "constant-condition", source); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v, want none for a call condition", diagnostics)
	}
}

func TestConstantConditionIgnoresCallExpression(t *testing.T) {
	source := []byte("main() { if (IsPlayerConnected(playerid) == 1) {} }\n")
	if diagnostics := ruletest.RunRule(t, "test.pwn", "constant-condition", source); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v, want none for a call expression", diagnostics)
	}
}
