package openmp_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules/openmp"
)

func TestCallbackSignatureUsesTarget(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(openmp.CallbackSignature{})
	engine := lint.NewEngine(reg)
	src := []byte("public OnPlayerDeath(playerid, killerid, reason) { return 1; }\n")
	rules := map[string]diagnostic.Severity{"callback-signature": diagnostic.SeverityError}
	engine.Target = "samp"
	if diags := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, rules, nil, nil); len(diags) != 0 {
		t.Fatalf("SA-MP diagnostics = %+v", diags)
	}
	engine.Target = "openmp"
	if diags := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, rules, nil, nil); len(diags) != 1 {
		t.Fatalf("open.mp diagnostics = %+v", diags)
	}
}
