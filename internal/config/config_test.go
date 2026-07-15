package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type stubRule struct{ m lint.Metadata }

func (s stubRule) Metadata() lint.Metadata { return s.m }
func (s stubRule) Run(_ *lint.Context)     {}

func regWith(t *testing.T) *lint.Registrar {
	t.Helper()
	reg := lint.NewRegistrar()
	reg.MustRegister(stubRule{m: lint.Metadata{
		ID: "alpha", Name: "Alpha", Summary: "alpha rule", Category: diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning, AnalysisLevel: lint.SyntaxAnalysis, DefaultEnabled: true,
		Options: []lint.Option{{Name: "threshold", Type: lint.OptionInteger, Default: int64(10), Minimum: 1, HasMinimum: true}},
	}})
	reg.MustRegister(stubRule{m: lint.Metadata{ID: "beta", Name: "Beta", Summary: "beta rule", Category: diagnostic.CategorySuspicious, DefaultSeverity: diagnostic.SeverityInfo, AnalysisLevel: lint.SyntaxAnalysis, DefaultEnabled: false}})
	return reg
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadAndResolve(t *testing.T) {
	reg := regWith(t)
	content := `profile = "strict"
target = "samp"
include = ["x/**/*.pwn"]
exclude = ["vendor/**"]

[lint]
warnings-as-errors = true

[rules]
alpha = "error"
`
	f, err := config.Load(writeTemp(t, content))
	if err != nil {
		t.Fatal(err)
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsEnabled("alpha") {
		t.Error("alpha should be enabled")
	}
	if r.SeverityFor("alpha", reg) != diagnostic.SeverityError {
		t.Errorf("alpha severity %v", r.SeverityFor("alpha", reg))
	}
	if r.Target != config.TargetSAMP {
		t.Errorf("target %v", r.Target)
	}
}

func TestExternalRuleConfiguration(t *testing.T) {
	content := `[[external-rules]]
name = "custom"
command = "./pawnlint-custom"
arguments = ["--mode", "lint"]
timeout-ms = 2500
[external-rules.configuration]
mode = "strict"
`
	file, err := config.Load(writeTemp(t, content))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := config.Resolve(file, "", regWith(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Source.ExternalRules) != 1 || resolved.Source.ExternalRules[0].Name != "custom" || resolved.Source.ExternalRules[0].TimeoutMS != 2500 || resolved.Source.ExternalRules[0].Configuration["mode"] != "strict" {
		t.Fatalf("external rules = %#v", resolved.Source.ExternalRules)
	}
}

func TestExternalRuleConfigurationRejectsInvalidEntries(t *testing.T) {
	file := config.Defaults()
	file.ExternalRules = []config.ExternalRule{{Name: "bad/name", Command: "tool"}}
	if _, err := config.Resolve(file, "", regWith(t)); err == nil {
		t.Fatal("invalid external rule name accepted")
	}
	file.ExternalRules = []config.ExternalRule{{Name: "custom", Command: "tool", TimeoutMS: 300001}}
	if _, err := config.Resolve(file, "", regWith(t)); err == nil {
		t.Fatal("invalid external rule timeout accepted")
	}
}

func TestPresetMergeAndLocalOverride(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, "policy")
	if err := os.Mkdir(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	basePath := filepath.Join(policyDir, "base.toml")
	base := `profile = "strict"
[lint]
warnings-as-errors = true
max-diagnostics = 20
[rules.alpha]
severity = "warning"
threshold = 5
`
	if err := os.WriteFile(basePath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "pawnlint.toml")
	local := `presets = ["policy/base.toml"]
[lint]
warnings-as-errors = false
max-diagnostics = 0
[rules.alpha]
severity = "error"
`
	if err := os.WriteFile(configPath, []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := config.Resolve(file, configPath, regWith(t))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Profile != "strict" || resolved.Source.Lint.WarningsAsErrors || resolved.Source.Lint.MaxDiagnostics != 0 {
		t.Fatalf("resolved = %+v", resolved.Source)
	}
	if resolved.Enabled["alpha"] != diagnostic.SeverityError || resolved.RuleConfig["alpha"]["threshold"] != int64(5) {
		t.Fatalf("alpha = %v, config = %#v", resolved.Enabled["alpha"], resolved.RuleConfig["alpha"])
	}
}

func TestPresetOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "first.toml"), []byte("[rules]\nalpha = \"warning\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "second.yaml"), []byte("rules:\n  alpha: info\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "pawnlint.json")
	if err := os.WriteFile(path, []byte(`{"presets":["first.toml","second.yaml"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := config.Resolve(file, path, regWith(t))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Enabled["alpha"] != diagnostic.SeverityInfo {
		t.Fatalf("alpha = %v", resolved.Enabled["alpha"])
	}
}

func TestPresetRejectsProjectContext(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "policy.toml"), []byte("target = \"samp\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(path, []byte("presets = [\"policy.toml\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil || !strings.Contains(err.Error(), "may only contain") {
		t.Fatalf("error = %v", err)
	}
}

func TestPresetCycle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.toml"), []byte("presets = [\"two.toml\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.toml"), []byte("presets = [\"one.toml\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(filepath.Join(dir, "one.toml")); err == nil || !strings.Contains(err.Error(), "preset cycle") {
		t.Fatalf("error = %v", err)
	}
}

func TestPresetRequiresFileLoading(t *testing.T) {
	if _, err := config.DecodeBytes([]byte("presets = [\"policy.toml\"]\n")); err == nil || !strings.Contains(err.Error(), "require loading from a file") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnknownRuleID(t *testing.T) {
	reg := regWith(t)
	content := `[rules]
no-such-rule = "warning"
`
	f, err := config.Load(writeTemp(t, content))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Fatal("expected unknown rule error")
	}
}

func TestRuleAliasResolvesWithMigration(t *testing.T) {
	reg := regWith(t)
	reg.MustRegisterAlias("old-alpha", "alpha")
	file := config.Defaults()
	file.Rules = map[string]any{"old-alpha": map[string]any{"severity": "error", "threshold": 4}}
	resolved, err := config.Resolve(file, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Enabled["alpha"] != diagnostic.SeverityError || resolved.RuleConfig["alpha"]["threshold"] != int64(4) {
		t.Fatalf("resolved = %+v", resolved)
	}
	if len(resolved.RuleMigrations) != 1 || resolved.RuleMigrations[0].Deprecated != "old-alpha" || resolved.RuleMigrations[0].Replacement != "alpha" {
		t.Fatalf("migrations = %+v", resolved.RuleMigrations)
	}
	if err := resolved.ApplyCLIOverrides("", "", []string{"old-alpha"}, nil, reg); err != nil {
		t.Fatal(err)
	}
	if len(resolved.RuleMigrations) != 1 {
		t.Fatalf("migrations = %+v", resolved.RuleMigrations)
	}
}

func TestRuleAliasRejectsDuplicateCanonicalConfiguration(t *testing.T) {
	reg := regWith(t)
	reg.MustRegisterAlias("old-alpha", "alpha")
	file := config.Defaults()
	file.Rules = map[string]any{"old-alpha": "warning", "alpha": "error"}
	if _, err := config.Resolve(file, "", reg); err == nil || !strings.Contains(err.Error(), "configured by both") {
		t.Fatalf("error = %v", err)
	}
}

func TestInvalidSeverity(t *testing.T) {
	reg := regWith(t)
	content := `[rules]
alpha = "bogus"
`
	f, _ := config.Load(writeTemp(t, content))
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Fatal("expected invalid severity error")
	}
}

func TestUnknownProfile(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Profile = "nopenope"
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Fatal("expected unknown profile error")
	}
}

func TestUnknownFields(t *testing.T) {
	reg := regWith(t)
	content := `bogus-field = 1
[rules]
alpha = "warning"
`
	_, err := config.Load(writeTemp(t, content))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "bogus-field") {
		t.Errorf("error should name bogus-field: %v", err)
	}
	_ = reg
}

func TestDisableViaOff(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Profile = "all"
	f.Rules = map[string]any{"alpha": "off"}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsEnabled("alpha") {
		t.Error("alpha should be off")
	}
}

func TestPerRuleConfigTable(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Rules = map[string]any{
		"alpha": map[string]any{"severity": "info", "threshold": 20},
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if r.SeverityFor("alpha", reg) != diagnostic.SeverityInfo {
		t.Errorf("severity %v", r.SeverityFor("alpha", reg))
	}
	if r.RuleConfig["alpha"]["threshold"] != int64(20) {
		t.Errorf("threshold %v", r.RuleConfig["alpha"]["threshold"])
	}
}

func TestPerRuleOptionValidation(t *testing.T) {
	reg := regWith(t)
	for _, rules := range []map[string]any{
		{"alpha": map[string]any{"unknown": true}},
		{"alpha": map[string]any{"threshold": "large"}},
		{"alpha": map[string]any{"threshold": 0}},
		{"beta": map[string]any{"threshold": 10}},
	} {
		f := config.Defaults()
		f.Rules = rules
		if _, err := config.Resolve(f, "", reg); err == nil {
			t.Fatalf("invalid rule options accepted: %#v", rules)
		}
	}
}

func TestPerRuleObjectListOption(t *testing.T) {
	reg := lint.NewRegistrar()
	reg.MustRegister(stubRule{m: lint.Metadata{
		ID: "naming", Name: "Naming", Summary: "naming", Category: diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning, AnalysisLevel: lint.SemanticAnalysis,
		Options: []lint.Option{{
			Name: "conventions", Type: lint.OptionObjectList,
			Fields: []lint.Option{{Name: "kinds", Type: lint.OptionStringList, Choices: []string{"function", "local"}}, {Name: "case", Type: lint.OptionString, Required: true}},
		}},
	}})
	file, err := config.DecodeBytes([]byte(`[rules.naming]
severity = "warning"
conventions = [{ kinds = ["function"], case = "PascalCase" }]
`))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := config.Resolve(file, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	conventions, ok := resolved.RuleConfig["naming"]["conventions"].([]map[string]any)
	if !ok || len(conventions) != 1 || conventions[0]["case"] != "PascalCase" {
		t.Fatalf("conventions = %#v", resolved.RuleConfig["naming"]["conventions"])
	}
	invalid, err := config.DecodeBytes([]byte(`[rules.naming]
conventions = [{ kinds = ["function"], unknown = true }]
`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.Resolve(invalid, "", reg); err == nil {
		t.Fatal("unknown object field was accepted")
	}
}

func TestEnabledForPathAppliesMatchingOverride(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Profile = "all"
	f.Overrides = []config.Override{
		{Paths: []string{"testdata/**"}, Rules: map[string]any{"alpha": "hint"}},
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if sev := r.EnabledForPath("testdata/fixtures/a.pwn")["alpha"]; sev != diagnostic.SeverityHint {
		t.Errorf("overridden path severity = %v, want hint", sev)
	}
	if sev := r.EnabledForPath("gamemodes/main.pwn")["alpha"]; sev != diagnostic.SeverityWarning {
		t.Errorf("non-matching path severity = %v, want unchanged warning", sev)
	}
	if r.Enabled["alpha"] != diagnostic.SeverityWarning {
		t.Errorf("base Enabled must be untouched by overrides, got %v", r.Enabled["alpha"])
	}
}

func TestEnabledForPathOverrideCanDisableOrEnable(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Overrides = []config.Override{
		{Paths: []string{"generated/**"}, Rules: map[string]any{"alpha": "off", "beta": "warning"}},
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	enabled := r.EnabledForPath("generated/x.pwn")
	if _, ok := enabled["alpha"]; ok {
		t.Error("alpha should be disabled for the overridden path")
	}
	if enabled["beta"] != diagnostic.SeverityWarning {
		t.Errorf("beta should be enabled for the overridden path, got %v", enabled["beta"])
	}
	if _, ok := r.EnabledForPath("other.pwn")["beta"]; ok {
		t.Error("beta must remain off for a non-matching path")
	}
}

func TestOverridesLaterEntryWinsOnConflict(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Profile = "all"
	f.Overrides = []config.Override{
		{Paths: []string{"a/**"}, Rules: map[string]any{"alpha": "hint"}},
		{Paths: []string{"a/**"}, Rules: map[string]any{"alpha": "error"}},
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if sev := r.EnabledForPath("a/x.pwn")["alpha"]; sev != diagnostic.SeverityError {
		t.Errorf("later override should win, got %v", sev)
	}
}

func TestRuleConfigForPathAppliesMatchingOverride(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Rules = map[string]any{"alpha": map[string]any{"threshold": 20}}
	f.Overrides = []config.Override{
		{Paths: []string{"testdata/**"}, Rules: map[string]any{"alpha": map[string]any{"threshold": 5}}},
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if v := r.RuleConfigForPath("testdata/x.pwn")["alpha"]["threshold"]; v != int64(5) {
		t.Errorf("overridden threshold = %v, want 5", v)
	}
	if v := r.RuleConfigForPath("other.pwn")["alpha"]["threshold"]; v != int64(20) {
		t.Errorf("non-matching path threshold = %v, want unchanged 20", v)
	}
}

func TestOverridesRejectEmptyPathsOrRules(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Overrides = []config.Override{{Paths: nil, Rules: map[string]any{"alpha": "hint"}}}
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Error("expected error for override with no paths")
	}
	f.Overrides = []config.Override{{Paths: []string{"a/**"}, Rules: nil}}
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Error("expected error for override with no rules")
	}
}

func TestOverridesRejectUnknownRuleID(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Overrides = []config.Override{{Paths: []string{"a/**"}, Rules: map[string]any{"nope": "hint"}}}
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Error("expected error for unknown rule ID in override")
	}
}

func TestResolveLoadsRelativeAPIMetadata(t *testing.T) {
	dir := t.TempDir()
	metadataPath := filepath.Join(dir, "api.json")
	if err := os.WriteFile(metadataPath, []byte(`{"natives":{"PluginNative":{"parameters":[{"name":"value"}]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := config.Defaults()
	f.APIMetadata = []string{"api.json"}
	resolved, err := config.Resolve(f, filepath.Join(dir, "pawnlint.toml"), regWith(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resolved.API.Natives["PluginNative"]; !ok {
		t.Fatalf("API metadata = %#v", resolved.API)
	}
}

func TestApplyCLIOverrides(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyCLIOverrides("all", "", []string{"beta"}, []string{"alpha"}, reg); err != nil {
		t.Fatal(err)
	}
	if !r.IsEnabled("beta") {
		t.Error("beta should be enabled by --enable")
	}
	if r.Source.Profile != "all" {
		t.Errorf("source profile = %q", r.Source.Profile)
	}
	if r.IsEnabled("alpha") {
		t.Error("alpha should be disabled by --disable")
	}
}

func TestApplyCLIOverridesUnknownRule(t *testing.T) {
	reg := regWith(t)
	r, _ := config.Resolve(config.Defaults(), "", reg)
	if err := r.ApplyCLIOverrides("", "", []string{"nope"}, nil, reg); err == nil {
		t.Fatal("expected unknown rule error")
	}
}

func TestProfileOverrideReplacesPreviousProfile(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Profile = "all"
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyCLIOverrides("recommended", "", nil, nil, reg); err != nil {
		t.Fatal(err)
	}
	if r.IsEnabled("beta") {
		t.Error("beta should not leak from the all profile")
	}
}

func TestProfileOverridePreservesRuleOverride(t *testing.T) {
	reg := regWith(t)
	f := config.Defaults()
	f.Profile = "all"
	f.Rules = map[string]any{"beta": "error"}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyCLIOverrides("recommended", "", nil, nil, reg); err != nil {
		t.Fatal(err)
	}
	if r.SeverityFor("beta", reg) != diagnostic.SeverityError {
		t.Error("explicit beta severity should survive a profile override")
	}
}

func TestResolveDefaultsTargetAndRejectsNegativeLimit(t *testing.T) {
	reg := regWith(t)
	f := config.File{Profile: "recommended", Rules: map[string]any{}}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if r.Target != config.TargetOpenMP || r.Source.Target != string(config.TargetOpenMP) {
		t.Fatalf("target = %q, source target = %q", r.Target, r.Source.Target)
	}
	f.Lint.MaxDiagnostics = -1
	if _, err := config.Resolve(f, "", reg); err == nil {
		t.Error("negative max-diagnostics should fail validation")
	}
}

func TestLoadAndResolveBuilds(t *testing.T) {
	reg := regWith(t)
	content := `profile = "strict"
defines = ["GLOBAL"]

[[builds]]
name = "main"
entry = "gamemodes/main.pwn"
working-directory = "server"
files = ["includes/**"]
exclude = ["includes/generated/**"]
include-paths = ["dependencies/library"]
defines = ["FEATURE"]
target = "samp"
`
	f, err := config.Load(writeTemp(t, content))
	if err != nil {
		t.Fatal(err)
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Source.Builds) != 1 {
		t.Fatalf("build count = %d", len(r.Source.Builds))
	}
	build := r.Source.Builds[0]
	if build.Name != "main" || build.Entry != "gamemodes/main.pwn" || build.WorkingDirectory != "server" || build.Target != "samp" {
		t.Fatalf("build = %#v", build)
	}
	if len(build.Files) != 1 || len(build.Exclude) != 1 || len(build.IncludePaths) != 1 || len(build.Defines) != 1 {
		t.Fatalf("build collections = %#v", build)
	}
}

func TestResolveRejectsInvalidBuilds(t *testing.T) {
	reg := regWith(t)
	tests := []struct {
		name string
		file config.File
		want string
	}{
		{name: "missing name", file: config.File{Builds: []config.Build{{Entry: "main.pwn"}}}, want: "non-empty name"},
		{name: "missing entry", file: config.File{Builds: []config.Build{{Name: "main"}}}, want: "non-empty entry"},
		{name: "duplicate", file: config.File{Builds: []config.Build{{Name: "main", Entry: "a.pwn"}, {Name: "main", Entry: "b.pwn"}}}, want: "duplicate build"},
		{name: "target", file: config.File{Builds: []config.Build{{Name: "main", Entry: "main.pwn", Target: "other"}}}, want: "unknown target"},
		{name: "variants", file: config.File{Builds: []config.Build{{Name: "main", Entry: "main.pwn"}}, Variants: []config.Variant{{Name: "feature"}}}, want: "cannot be configured together"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.file.Profile = "recommended"
			test.file.Rules = map[string]any{}
			_, err := config.Resolve(test.file, "", reg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestCLITargetOverridesBuildTargets(t *testing.T) {
	reg := regWith(t)
	f := config.File{
		Profile: "recommended",
		Target:  "openmp",
		Rules:   map[string]any{},
		Builds:  []config.Build{{Name: "main", Entry: "main.pwn", Target: "openmp"}},
	}
	r, err := config.Resolve(f, "", reg)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyCLIOverrides("", "samp", nil, nil, reg); err != nil {
		t.Fatal(err)
	}
	if r.Target != config.TargetSAMP || r.Source.Builds[0].Target != "samp" {
		t.Fatalf("target = %q, build target = %q", r.Target, r.Source.Builds[0].Target)
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(p, []byte("profile = \"all\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "nested", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	found, _, err := config.Discover(sub)
	if err != nil {
		t.Fatal(err)
	}
	if found != p {
		t.Errorf("discovered %q want %q", found, p)
	}
}

func TestInitConfigTextAndListRules(t *testing.T) {
	reg := regWith(t)
	s := config.InitConfigText(reg)
	if !strings.Contains(s, "alpha") || !strings.Contains(s, "profile =") {
		t.Errorf("init text missing content")
	}
	f, err := config.DecodeBytes([]byte(extractInitRules(s, reg)))
	if err != nil {
		t.Logf("init derived rules snippet:\n%s", extractInitRules(s, reg))
	}
	_ = f
	list := config.ListRulesText(reg)
	if !strings.Contains(list, "alpha\t") {
		t.Errorf("list missing alpha")
	}
	expl := config.ExplainText(reg.Sorted()[0])
	if !strings.Contains(expl, "alpha") {
		t.Errorf("explain missing alpha")
	}
}

func extractInitRules(s string, reg *lint.Registrar) string {
	return "[rules]\n"
}
