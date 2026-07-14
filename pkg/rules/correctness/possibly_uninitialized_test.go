package correctness

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestPossiblyUninitializedParameterDirection(t *testing.T) {
	source := []byte(`read_value()
{
    new value;
    PluginRead(value);
}

write_value()
{
    new value;
    PluginOutput(value);
    return value;
}
`)
	reg := lint.NewRegistrar()
	reg.MustRegister(PossiblyUninitialized{})
	engine := lint.NewEngine(reg)
	engine.API = &api.Metadata{Natives: map[string]api.Native{
		"PluginRead":   {Parameters: []api.Parameter{{Name: "value"}}},
		"PluginOutput": {Parameters: []api.Parameter{{Name: "value", Reference: true, Output: true}}},
	}}
	diagnostics := engine.LintFile("test.pwn", source, lint.ControlFlowAnalysis, map[string]diagnostic.Severity{
		"possibly-uninitialized": diagnostic.SeverityWarning,
	}, map[string]struct{}{"possibly-uninitialized": {}}, nil)
	if len(diagnostics) != 1 || diagnostics[0].Range.Start.Line != 4 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}
