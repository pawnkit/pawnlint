package lint_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestEngineRunsAllRules(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	known := map[string]struct{}{}
	ruleSet := map[string]diagnostic.Severity{}
	for _, id := range reg.IDs() {
		known[id] = struct{}{}
		m, _ := reg.Lookup(id)
		ruleSet[id] = m.DefaultSeverity
	}
	src := []byte("main()\n{\n if (a);\n {\n }\n playerid + 1;\n return f(), g();\n}\n")
	diags := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, ruleSet, known, nil)
	ids := map[string]bool{}
	for _, d := range diags {
		ids[d.RuleID] = true
	}
	if !ids["empty-condition-body"] {
		t.Error("expected empty-condition-body finding")
	}
	if !ids["discarded-expression"] {
		t.Error("expected discarded-expression finding")
	}
	if !ids["suspicious-comma-expression"] {
		t.Error("expected suspicious-comma-expression finding")
	}
	for _, d := range diags {
		if d.Severity == 0 {
			t.Errorf("diagnostic %q has zero severity", d.RuleID)
		}
		if d.Filename != "x.pwn" {
			t.Errorf("diagnostic %q filename %q", d.RuleID, d.Filename)
		}
	}
}

func TestEngineSuppression(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	known := map[string]struct{}{}
	for _, id := range reg.IDs() {
		known[id] = struct{}{}
	}
	ruleSet := map[string]diagnostic.Severity{
		"discarded-expression": diagnostic.SeverityWarning,
		"unknown-suppression":  diagnostic.SeverityWarning,
	}
	src := []byte("// pawnlint-disable-next-line discarded-expression\na + 1;\nb + 2;\n")
	diags := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, ruleSet, known, nil)
	for _, d := range diags {
		if d.RuleID == "discarded-expression" && d.Range.Start.Line == 2 {
			t.Errorf("line 2 should be suppressed")
		}
	}
}

func TestEngineReportsUnmatchedEnableAll(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	ruleSet := map[string]diagnostic.Severity{"unknown-suppression": diagnostic.SeverityWarning}
	known := map[string]struct{}{"unknown-suppression": {}}
	diags := engine.LintFile("x.pwn", []byte("// pawnlint-enable all\nmain(){}\n"), lint.SyntaxAnalysis, ruleSet, known, nil)
	if len(diags) != 1 || diags[0].Message != "unmatched `pawnlint-enable` for \"all\"" {
		t.Fatalf("diagnostics = %+v", diags)
	}
}

func TestEngineReportsParseErrorsWithoutSuppression(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	src := []byte("// pawnlint-disable all\n}\n")
	diags := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, map[string]diagnostic.Severity{}, nil, nil)
	found := false
	for _, d := range diags {
		if d.RuleID == lint.ParseErrorID {
			found = true
			if d.Severity != diagnostic.SeverityError {
				t.Fatalf("parse severity = %v", d.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("parse error not reported: %+v", diags)
	}
}

func TestEngineCoalescesAdjacentParseErrors(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	src := []byte("}\n}\n")
	diagnostics := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, map[string]diagnostic.Severity{}, nil, nil)
	count := 0
	for _, d := range diagnostics {
		if d.RuleID == lint.ParseErrorID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("parse diagnostics = %+v", diagnostics)
	}
}

func TestEngineReportsRulePanics(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(panicRule{})
	engine := lint.NewEngine(reg)
	diags := engine.LintFile("x.pwn", []byte("main(){}"), lint.SyntaxAnalysis, map[string]diagnostic.Severity{"panic-rule": diagnostic.SeverityWarning}, nil, nil)
	if len(diags) != 1 || diags[0].RuleID != lint.InternalErrorID {
		t.Fatalf("diagnostics = %+v", diags)
	}
}

func TestEngineNormalizesEmptyRangeToSourceStart(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(emptyRangeRule{})
	engine := lint.NewEngine(reg)
	diagnostics := engine.LintFile("test.pwn", []byte("main() {}\n"), lint.SyntaxAnalysis, map[string]diagnostic.Severity{"empty-range": diagnostic.SeverityWarning}, nil, nil)
	if len(diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diagnostics))
	}
	position := diagnostics[0].Range.Start
	if position.Offset != 0 || position.Line != 1 || position.Col != 1 {
		t.Fatalf("start = %+v, want offset 0 at 1:1", position)
	}
}

func TestEngineUsesConfiguredDefines(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	engine.Defines = []string{"FEATURE"}
	ruleSet := map[string]diagnostic.Severity{"division-by-zero": diagnostic.SeverityError}
	src := []byte("main() {\n#if defined FEATURE\nnew value = 1 / 0;\n#endif\n}\n")
	diags := engine.LintFile("x.pwn", src, lint.ControlFlowAnalysis, ruleSet, nil, nil)
	if len(diags) != 1 || diags[0].RuleID != "division-by-zero" {
		t.Fatalf("diagnostics = %+v", diags)
	}
}

func TestEngineUsesCustomAPIMetadata(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	metadata, err := api.Merge("openmp", &api.Metadata{Natives: map[string]api.Native{
		"PluginNative": {Parameters: []api.Parameter{{Name: "value"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	engine.API = metadata
	diagnostics := engine.LintFile("x.pwn", []byte("main() { PluginNative(); }\n"), lint.SemanticAnalysis, map[string]diagnostic.Severity{"native-argument-count": diagnostic.SeverityError}, nil, nil)
	if len(diagnostics) != 1 || diagnostics[0].RuleID != "native-argument-count" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestRegistryDeterministic(t *testing.T) {
	reg1 := rules.Default()
	reg2 := rules.Default()
	a, b := reg1.IDs(), reg2.IDs()
	if len(a) != len(b) {
		t.Fatalf("len mismatch")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("order mismatch at %d: %q vs %q", i, a[i], b[i])
		}
	}
}

func TestRegistryRejectsDuplicate(t *testing.T) {
	reg := lint.NewRegistrar()
	if err := reg.Register(dupRule{}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(dupRule{}); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestEngineUsesRegistrationOrder(t *testing.T) {
	var ran []string
	reg := lint.NewRegistrar()
	reg.MustRegister(recordingRule{id: "z-rule", ran: &ran})
	reg.MustRegister(recordingRule{id: "a-rule", ran: &ran})
	engine := lint.NewEngine(reg)
	engine.LintFile("x.pwn", []byte("main(){}"), lint.SyntaxAnalysis, map[string]diagnostic.Severity{
		"a-rule": diagnostic.SeverityWarning,
		"z-rule": diagnostic.SeverityWarning,
	}, nil, nil)
	if len(ran) != 2 || ran[0] != "z-rule" || ran[1] != "a-rule" {
		t.Fatalf("execution order = %v", ran)
	}
}

func TestEngineBuildsControlFlowOnDemand(t *testing.T) {
	rule := &controlFlowProbe{}
	reg := lint.NewRegistrar()
	reg.MustRegister(rule)
	engine := lint.NewEngine(reg)
	engine.LintFile("x.pwn", []byte("main(){}"), lint.ControlFlowAnalysis, map[string]diagnostic.Severity{
		"control-flow-probe": diagnostic.SeverityWarning,
	}, nil, nil)
	if !rule.ran || !rule.hasSemantic || !rule.hasFlow {
		t.Fatalf("probe = %+v", rule)
	}
}

type dupRule struct{}

func (dupRule) Metadata() lint.Metadata {
	return lint.Metadata{ID: "dup", Name: "n", Summary: "s", Category: diagnostic.CategoryCorrectness}
}
func (dupRule) Run(_ *lint.Context) {}

type recordingRule struct {
	id  string
	ran *[]string
}

func (r recordingRule) Metadata() lint.Metadata {
	return lint.Metadata{ID: r.id, Name: r.id, Summary: r.id, Category: diagnostic.CategoryCorrectness, DefaultSeverity: diagnostic.SeverityWarning}
}

func (r recordingRule) Run(_ *lint.Context) {
	*r.ran = append(*r.ran, r.id)
}

type panicRule struct{}

func (panicRule) Metadata() lint.Metadata {
	return lint.Metadata{ID: "panic-rule", Name: "panic", Summary: "panic", DefaultSeverity: diagnostic.SeverityWarning}
}

func (panicRule) Run(_ *lint.Context) {
	panic("boom")
}

type emptyRangeRule struct{}

func (emptyRangeRule) Metadata() lint.Metadata {
	return lint.Metadata{ID: "empty-range", Name: "empty range", Summary: "empty range", DefaultSeverity: diagnostic.SeverityWarning}
}

func (emptyRangeRule) Run(ctx *lint.Context) {
	ctx.Report(diagnostic.Diagnostic{Message: "empty range"})
}

type controlFlowProbe struct {
	ran         bool
	hasSemantic bool
	hasFlow     bool
}

func (*controlFlowProbe) Metadata() lint.Metadata {
	return lint.Metadata{ID: "control-flow-probe", Name: "control flow probe", Summary: "probe", AnalysisLevel: lint.ControlFlowAnalysis, DefaultSeverity: diagnostic.SeverityWarning}
}

func (r *controlFlowProbe) Run(ctx *lint.Context) {
	r.ran = true
	r.hasSemantic = ctx.Semantic != nil
	r.hasFlow = ctx.Flow != nil
}
