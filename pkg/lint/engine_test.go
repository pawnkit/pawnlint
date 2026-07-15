package lint_test

import (
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/fix"
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

func TestEnginePreservesAdjacentParseErrors(t *testing.T) {
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
	if count != 2 {
		t.Fatalf("parse diagnostics = %+v", diagnostics)
	}
}

func TestEngineUsesStructuredParseRecovery(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	src := []byte("main() { return (1; }\n")
	diagnostics := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, map[string]diagnostic.Severity{}, nil, nil)
	var parseDiagnostic *diagnostic.Diagnostic
	for i := range diagnostics {
		if diagnostics[i].RuleID == lint.ParseErrorID && diagnostics[i].Code == "missing_token" {
			parseDiagnostic = &diagnostics[i]
			break
		}
	}
	if parseDiagnostic == nil || parseDiagnostic.Fix == nil {
		t.Fatalf("structured parse recovery missing: %+v", diagnostics)
	}
	plan, err := fix.Build(map[string][]byte{"x.pwn": src}, []diagnostic.Diagnostic{*parseDiagnostic})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 1 || string(plan.Changes[0].After) != "main() { return (1); }\n" {
		t.Fatalf("parse recovery plan = %+v", plan)
	}
}

func TestEngineDoesNotFixSuggestedParseRecovery(t *testing.T) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	diagnostics := engine.LintFile("x.pwn", []byte("main() { value = ; }\n"), lint.SyntaxAnalysis, map[string]diagnostic.Severity{}, nil, nil)
	for _, item := range diagnostics {
		if item.RuleID == lint.ParseErrorID && item.Code == "missing_expression" {
			if item.Fix != nil {
				t.Fatalf("suggested parser recovery became a fix: %+v", item)
			}
			return
		}
	}
	t.Fatalf("missing structured parse diagnostic: %+v", diagnostics)
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

func TestProfilesExcludePreviewRules(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(previewRule{})
	for _, profile := range []lint.Profile{lint.ProfileRecommended, lint.ProfileStrict, lint.ProfileAll} {
		if _, enabled := reg.EnabledForProfile(profile)["preview-rule"]; enabled {
			t.Fatalf("preview rule enabled by %s", profile)
		}
	}
}

func TestRegistryRejectsInvalidStability(t *testing.T) {
	reg := lint.NewRegistrar()
	err := reg.Register(metadataRule{metadata: lint.Metadata{
		ID: "invalid-stability", Name: "invalid", Summary: "invalid",
		Stability: lint.Stability(100),
	}})
	if err == nil {
		t.Fatal("invalid stability accepted")
	}
}

func TestRegistryRejectsInvalidOptions(t *testing.T) {
	tests := [][]lint.Option{
		{{Name: "severity", Type: lint.OptionString}},
		{{Name: "value", Type: lint.OptionString}, {Name: "value", Type: lint.OptionString}},
		{{Name: "value", Type: lint.OptionString, Minimum: 1, HasMinimum: true}},
		{{Name: "value", Type: lint.OptionInteger, Choices: []string{"one"}}},
		{{Name: "value", Type: lint.OptionInteger, Default: 0, Minimum: 1, HasMinimum: true}},
	}
	for _, options := range tests {
		reg := lint.NewRegistrar()
		err := reg.Register(metadataRule{metadata: lint.Metadata{ID: "invalid-options", Name: "invalid", Summary: "invalid", Options: options}})
		if err == nil {
			t.Fatalf("invalid options accepted: %+v", options)
		}
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

func TestEngineReportsTimings(t *testing.T) {
	rule := &controlFlowProbe{}
	reg := lint.NewRegistrar()
	reg.MustRegister(rule)
	engine := lint.NewEngine(reg)
	var events []lint.TimingEvent
	engine.ObserveTiming = func(event lint.TimingEvent) {
		events = append(events, event)
	}
	engine.LintFile("x.pwn", []byte("main(){}"), lint.ControlFlowAnalysis, map[string]diagnostic.Severity{
		"control-flow-probe": diagnostic.SeverityWarning,
	}, nil, nil)
	wanted := map[lint.TimingStage]bool{
		lint.TimingParse:       false,
		lint.TimingSemantic:    false,
		lint.TimingControlFlow: false,
		lint.TimingRule:        false,
	}
	for _, event := range events {
		if event.Duration < 0 {
			t.Fatalf("negative duration: %+v", event)
		}
		if event.Stage == lint.TimingRule && event.RuleID != "control-flow-probe" {
			t.Fatalf("rule event = %+v", event)
		}
		wanted[event.Stage] = true
	}
	for stage, found := range wanted {
		if !found {
			t.Errorf("missing %s event: %+v", stage, events)
		}
	}
}

func TestDeprecatedSuppressionAliasResolves(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(aliasSuppressionRule{})
	reg.MustRegisterAlias("old-rule", "canonical-rule")
	engine := lint.NewEngine(reg)
	src := []byte("// pawnlint-disable-next-line old-rule\nvalue;\n")
	diagnostics := engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, map[string]diagnostic.Severity{
		"canonical-rule": diagnostic.SeverityWarning,
	}, map[string]struct{}{"canonical-rule": {}, "old-rule": {}}, nil)
	if len(diagnostics) != 1 || diagnostics[0].RuleID != lint.DeprecatedRuleID || !strings.Contains(diagnostics[0].Message, "canonical-rule") {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestRegistrarAliases(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(recordingRule{id: "canonical", ran: &[]string{}})
	reg.MustRegisterAlias("old", "canonical")
	if id, deprecated, known := reg.ResolveID("old"); id != "canonical" || !deprecated || !known {
		t.Fatalf("resolve = %q, %v, %v", id, deprecated, known)
	}
	if aliases := reg.Aliases(); len(aliases) != 1 || aliases[0].Deprecated != "old" || aliases[0].Replacement != "canonical" {
		t.Fatalf("aliases = %+v", aliases)
	}
	if metadata, found := reg.Lookup("old"); !found || metadata.ID != "canonical" {
		t.Fatalf("metadata = %+v, found = %v", metadata, found)
	}
	if err := reg.RegisterAlias("missing", "unknown"); err == nil {
		t.Fatal("unknown alias target accepted")
	}
	if err := reg.RegisterAlias("old", "canonical"); err == nil {
		t.Fatal("duplicate alias accepted")
	}
}

type dupRule struct{}

func (dupRule) Metadata() lint.Metadata {
	return lint.Metadata{ID: "dup", Name: "n", Summary: "s", Category: diagnostic.CategoryCorrectness}
}
func (dupRule) Run(_ *lint.Context) {}

type previewRule struct{}

func (previewRule) Metadata() lint.Metadata {
	return lint.Metadata{
		ID: "preview-rule", Name: "preview", Summary: "preview",
		DefaultSeverity: diagnostic.SeverityWarning, DefaultEnabled: true,
		Stability: lint.StabilityPreview,
	}
}

func (previewRule) Run(_ *lint.Context) {}

type metadataRule struct {
	metadata lint.Metadata
}

func (r metadataRule) Metadata() lint.Metadata { return r.metadata }
func (metadataRule) Run(_ *lint.Context)       {}

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

type aliasSuppressionRule struct{}

func (aliasSuppressionRule) Metadata() lint.Metadata {
	return lint.Metadata{ID: "canonical-rule", Name: "canonical", Summary: "canonical", DefaultSeverity: diagnostic.SeverityWarning}
}

func (aliasSuppressionRule) Run(ctx *lint.Context) {
	offset := strings.Index(string(ctx.File.Source), "value")
	ctx.Report(diagnostic.Diagnostic{Message: "finding", Range: ctx.File.LineTable.Range(offset, offset+5)})
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
