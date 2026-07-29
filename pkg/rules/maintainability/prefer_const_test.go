package maintainability

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestPreferConstIgnoresForLoopCounter(t *testing.T) {
	t.Parallel()

	source := []byte(`
public OnPlayerConnect(playerid)
{
    for (new i = 12; i >= 0; --i)
    {
        playerid += i;
    }
    return 1;
}
`)
	registry := lint.NewRegistrar()
	registry.MustRegister(PreferConst{})
	engine := lint.NewEngine(registry)
	diagnostics := engine.LintFile(
		"test.pwn",
		source,
		lint.SemanticAnalysis,
		map[string]diagnostic.Severity{"prefer-const": diagnostic.SeverityWarning},
		map[string]struct{}{"prefer-const": {}},
		nil,
	)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}
