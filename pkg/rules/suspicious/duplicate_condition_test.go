package suspicious

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestDuplicateConditionReportsOnlyPureRepeats(t *testing.T) {
	t.Parallel()

	source := []byte(`
native Check();

stock Test(value)
{
    if (value == 1) {}
    else if ((value == 1)) {}

    if (Check()) {}
    else if (Check()) {}
}
`)
	registry := lint.NewRegistrar()
	registry.MustRegister(DuplicateCondition{})
	engine := lint.NewEngine(registry)
	diagnostics := engine.LintFile(
		"test.pwn",
		source,
		lint.SemanticAnalysis,
		map[string]diagnostic.Severity{"duplicate-condition": diagnostic.SeverityWarning},
		map[string]struct{}{"duplicate-condition": {}},
		nil,
	)
	if len(diagnostics) != 1 || diagnostics[0].Range.Start.Line != 7 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}
