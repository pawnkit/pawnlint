package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/baseline"
	"github.com/pawnkit/pawnlint/internal/cli"
)

func runCLI(t *testing.T, args []string, stdin string) (stdout, stderr string, code int) {
	t.Helper()
	in := strings.NewReader(stdin)
	var out, errb bytes.Buffer
	code = cli.Run(args, in, &out, &errb)
	return out.String(), errb.String(), code
}

func TestCLIVersion(t *testing.T) {
	out, _, code := runCLI(t, []string{"--version"}, "")
	if code != 0 || !strings.Contains(out, "dev") {
		t.Errorf("version: out=%q code=%d", out, code)
	}
}

func TestCLIListRules(t *testing.T) {
	out, _, code := runCLI(t, []string{"--list-rules"}, "")
	if code != 0 || !strings.Contains(out, "empty-condition-body") {
		t.Errorf("list-rules: out=%q code=%d", out, code)
	}
}

func TestCLIExplain(t *testing.T) {
	out, _, code := runCLI(t, []string{"--explain", "empty-condition-body"}, "")
	if code != 0 || !strings.Contains(out, "empty-condition-body") {
		t.Errorf("explain: out=%q code=%d", out, code)
	}
	_, _, code = runCLI(t, []string{"--explain", "no-such-rule"}, "")
	if code == 0 {
		t.Error("unknown rule should fail")
	}
}

func TestCLIInitConfig(t *testing.T) {
	out, _, code := runCLI(t, []string{"--init-config"}, "")
	if code != 0 || !strings.Contains(out, "profile =") || !strings.Contains(out, "empty-condition-body") {
		t.Errorf("init-config: out=%q code=%d", out, code)
	}
}

func TestCLIBadFormat(t *testing.T) {
	_, _, code := runCLI(t, []string{"--format", "xml", "x.pwn"}, "")
	if code != 2 {
		t.Errorf("bad format code=%d want 2", code)
	}
}

func TestCLIBadColor(t *testing.T) {
	_, _, code := runCLI(t, []string{"--color", "sometimes", "x.pwn"}, "")
	if code != 2 {
		t.Errorf("bad color code=%d want 2", code)
	}
}

func TestCLIUsesPreset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	presetPath := filepath.Join(dir, "policy.toml")
	sourcePath := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(configPath, []byte("presets = [\"policy.toml\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(presetPath, []byte("[rules]\nempty-condition-body = \"off\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("main() { if (value); { return; } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--config", configPath, sourcePath}, "")
	if code != 0 || out != "" || stderr != "" {
		t.Fatalf("code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLITimings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(path, []byte("main() { if (value); { return; } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--timings", "--format", "compact", path}, "")
	if code != 1 || !strings.Contains(out, "empty-condition-body") {
		t.Fatalf("code=%d output=%q", code, out)
	}
	for _, expected := range []string{"pawnlint timings:", "parse", "semantic", "control-flow", "project", "rules", "output", "total", "pawnlint rule timings:", "empty-condition-body"} {
		if !strings.Contains(stderr, expected) {
			t.Errorf("stderr missing %q: %q", expected, stderr)
		}
	}
}

func TestCLIBaselineGenerateApplyAndPrune(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	baselinePath := filepath.Join(dir, "pawnlint-baseline.json")
	sourcePath := filepath.Join(dir, "main.pwn")
	bad := []byte("main() { if (value); { return; } }\n")
	if err := os.WriteFile(configPath, []byte("baseline = \"pawnlint-baseline.json\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, bad, 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--config", configPath, "--generate-baseline", "--format", "compact", sourcePath}, "")
	if code != 0 || out != "" || !strings.Contains(stderr, "wrote") {
		t.Fatalf("generate: code=%d output=%q stderr=%q", code, out, stderr)
	}
	generated, err := baseline.Load(baselinePath)
	if err != nil || len(generated.Entries) == 0 {
		t.Fatalf("generated = %#v, err = %v", generated, err)
	}
	out, stderr, code = runCLI(t, []string{"--config", configPath, "--format", "compact", sourcePath}, "")
	if code != 0 || out != "" || stderr != "" {
		t.Fatalf("apply: code=%d output=%q stderr=%q", code, out, stderr)
	}
	if err := os.WriteFile(sourcePath, []byte("main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code = runCLI(t, []string{"--config", configPath, "--prune-baseline", "--format", "compact", sourcePath}, "")
	if code != 0 || out != "" || !strings.Contains(stderr, "pruned") {
		t.Fatalf("prune: code=%d output=%q stderr=%q", code, out, stderr)
	}
	pruned, err := baseline.Load(baselinePath)
	if err != nil || len(pruned.Entries) != 0 {
		t.Fatalf("pruned = %#v, err = %v", pruned, err)
	}
	if err := os.WriteFile(sourcePath, bad, 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code = runCLI(t, []string{"--config", configPath, "--format", "compact", sourcePath}, "")
	if code != 1 || !strings.Contains(out, "empty-condition-body") {
		t.Fatalf("new finding: code=%d output=%q", code, out)
	}
}

func TestCLIBaselineValidation(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(sourcePath, []byte("main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runCLI(t, []string{"--generate-baseline", sourcePath}, "")
	if code != 2 || !strings.Contains(stderr, "requires --baseline") {
		t.Fatalf("missing path: code=%d stderr=%q", code, stderr)
	}
	_, stderr, code = runCLI(t, []string{"--baseline", filepath.Join(dir, "missing.json"), sourcePath}, "")
	if code != 2 || !strings.Contains(stderr, "baseline") {
		t.Fatalf("missing file: code=%d stderr=%q", code, stderr)
	}
	_, stderr, code = runCLI(t, []string{"--baseline", filepath.Join(dir, "baseline.json"), "--generate-baseline", "--prune-baseline", sourcePath}, "")
	if code != 2 || !strings.Contains(stderr, "cannot be combined") {
		t.Fatalf("conflict: code=%d stderr=%q", code, stderr)
	}
}

func TestCLIBaselineOverrideResolvesFromWorkingDirectory(t *testing.T) {
	configDir := t.TempDir()
	workingDir := t.TempDir()
	configPath := filepath.Join(configDir, "pawnlint.toml")
	sourcePath := filepath.Join(workingDir, "main.pwn")
	if err := os.WriteFile(configPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workingDir)
	_, stderr, code := runCLI(t, []string{"--config", configPath, "--baseline", "baseline.json", "--generate-baseline", sourcePath}, "")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if _, err := baseline.Load(filepath.Join(workingDir, "baseline.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "baseline.json")); !os.IsNotExist(err) {
		t.Fatalf("config-relative override exists: %v", err)
	}
}

func TestCLIConfiguredBaselineWithRelativeConfigPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pawnlint.toml"), []byte("baseline = \"baseline.json\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.pwn"), []byte("main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	_, stderr, code := runCLI(t, []string{"--config", "pawnlint.toml", "--generate-baseline", "main.pwn"}, "")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if _, err := baseline.Load(filepath.Join(dir, "baseline.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCLIFixAppliesAndRechecks(t *testing.T) {
	for _, flag := range []string{"--fix", "--fix-safe"} {
		dir := t.TempDir()
		path := filepath.Join(dir, "fix.pwn")
		if err := os.WriteFile(path, []byte("main() { new value; value = value; if (true); { return; } }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, stderr, code := runCLI(t, []string{flag, "--format", "compact", path}, "")
		if code != 0 || out != "" || stderr != "" {
			t.Fatalf("%s: code=%d output=%q stderr=%q", flag, code, out, stderr)
		}
		fixed, err := os.ReadFile(path)
		if err != nil || strings.Contains(string(fixed), "if (true);") || strings.Contains(string(fixed), "value = value") {
			t.Fatalf("%s: fixed=%q err=%v", flag, fixed, err)
		}
	}
}

func TestCLIFixAppliesExactParseRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "parse.pwn")
	if err := os.WriteFile(path, []byte("main() { return (1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--fix-safe", "--format", "compact", path}, "")
	if code != 0 || out != "" || stderr != "" {
		t.Fatalf("code=%d output=%q stderr=%q", code, out, stderr)
	}
	fixed, err := os.ReadFile(path)
	if err != nil || string(fixed) != "main() { return (1); }\n" {
		t.Fatalf("fixed=%q err=%v", fixed, err)
	}
}

func TestCLIDiffDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fix.pwn")
	source := []byte("main() { if (true); { return; } }\n")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--diff", path}, "")
	if code != 1 || stderr != "" || !strings.Contains(out, "--- ") || !strings.Contains(out, "-main() { if (true);") {
		t.Fatalf("diff: code=%d output=%q stderr=%q", code, out, stderr)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, source) {
		t.Fatalf("source changed: %q, %v", got, err)
	}
}

func TestCLIRejectsWritingStdin(t *testing.T) {
	_, stderr, code := runCLI(t, []string{"--stdin", "--fix"}, "main() {}")
	if code != 2 || !strings.Contains(stderr, "cannot write stdin") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestCLIPrintsDiffForStdin(t *testing.T) {
	out, stderr, code := runCLI(t, []string{"--stdin", "--stdin-filename", "input.pwn", "--diff"}, "main() { if (true); { return; } }\n")
	if code != 1 || stderr != "" || !strings.Contains(out, "--- input.pwn") {
		t.Fatalf("code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLIRejectsFixWithDiff(t *testing.T) {
	_, stderr, code := runCLI(t, []string{"--fix", "--diff", "test.pwn"}, "")
	if code != 2 || !strings.Contains(stderr, "cannot be combined") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestCLINoInput(t *testing.T) {
	_, _, code := runCLI(t, []string{}, "")
	if code != 2 {
		t.Errorf("no input code=%d want 2", code)
	}
}

func TestCLIMissingPath(t *testing.T) {
	_, _, code := runCLI(t, []string{filepath.Join(t.TempDir(), "missing.pwn")}, "")
	if code != 2 {
		t.Errorf("missing path code=%d want 2", code)
	}
}

func TestCLILintFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.pwn")
	if err := os.WriteFile(p, []byte("main()\n{\n if (a);\n {\n }\n playerid + 1;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--format", "compact", p}, "")
	if code != 1 {
		t.Errorf("lint code=%d want 1 (errors present)", code)
	}
	if !strings.Contains(out, "empty-condition-body") {
		t.Errorf("missing finding: %q", out)
	}
}

func TestCLICleanFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ok.pwn")
	if err := os.WriteFile(p, []byte("main()\n{\n if (a)\n {\n }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, code := runCLI(t, []string{p}, "")
	if code != 0 {
		t.Errorf("clean file code=%d want 0", code)
	}
}

func TestCLIStdin(t *testing.T) {
	out, _, code := runCLI(t, []string{"--stdin", "--stdin-filename", "in/main.pwn"}, "main()\n{\n playerid + 1;\n}\n")
	if code != 0 {
		t.Errorf("stdin code=%d", code)
	}
	if !strings.Contains(out, "discarded-expression") {
		t.Errorf("stdin missing finding: %q", out)
	}
}

func TestCLIStdinFailOnError(t *testing.T) {
	_, _, code := runCLI(t, []string{"--stdin", "--stdin-filename", "in/main.pwn"}, "main()\n{\n if (a);\n {\n }\n}\n")
	if code != 1 {
		t.Errorf("stdin error code=%d want 1", code)
	}
}

func TestCLIConfigDiscovery(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(cfg, []byte(`profile = "all"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "bad.pwn")
	if err := os.WriteFile(p, []byte("main()\n{\n if (a);\n {\n }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--format", "compact", p}, "")
	_ = out
	if code != 1 {
		t.Errorf("config discovery code=%d want 1", code)
	}
}

func TestCLIEnableDisable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.pwn")
	if err := os.WriteFile(p, []byte("main()\n{\n if (a);\n {\n }\n playerid + 1;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--disable", "empty-condition-body", "--format", "compact", p}, "")
	if code != 0 {
		t.Errorf("disabled rule code=%d want 0", code)
	}
	if strings.Contains(out, "empty-condition-body") {
		t.Errorf("should not emit disabled rule: %q", out)
	}
	if !strings.Contains(out, "discarded-expression") {
		t.Errorf("should still emit other rule: %q", out)
	}
}

func TestCLIUnknownEnableRule(t *testing.T) {
	_, _, code := runCLI(t, []string{"--enable", "nope", "x.pwn"}, "")
	if code != 2 {
		t.Errorf("unknown enable code=%d want 2", code)
	}
}

func TestCLIRunsSemanticRules(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "semantic.pwn")
	if err := os.WriteFile(p, []byte("main() { new value; value = value; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--format", "compact", p}, "")
	if code != 0 {
		t.Fatalf("semantic warning code=%d", code)
	}
	if !strings.Contains(out, "self-assignment") {
		t.Fatalf("semantic rule did not run: %q", out)
	}
}

func TestCLIRunsProjectRulesAcrossIncludes(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includePath := filepath.Join(includeDir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(configPath, []byte("include-paths = [\"include\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include <shared>\nShared() {}\nmain() {}\n")
	if err := os.WriteFile(mainPath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--config", configPath, "--format", "compact", mainPath}, "")
	if code != 1 || !strings.Contains(out, "duplicate-function-definition") {
		t.Fatalf("project rule did not run: code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLIParallelLintingIsRaceFreeAndDeterministic(t *testing.T) {
	dir := t.TempDir()
	const n = 60
	paths := make([]string, 0, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(dir, fmt.Sprintf("file%d.pwn", i))
		src := fmt.Sprintf("main%d() {\nnew value = 1 / 0;\n}\n", i)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	cfg := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(cfg, []byte("profile = \"strict\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := append([]string{"--config", cfg, "--format", "compact"}, paths...)
	for run := 0; run < 5; run++ {
		out, _, code := runCLI(t, args, "")
		if code != 1 {
			t.Fatalf("run %d: code=%d", run, code)
		}
		if got := strings.Count(out, "division-by-zero"); got != n {
			t.Fatalf("run %d: expected %d division-by-zero findings, got %d: %q", run, n, got, out)
		}
		if got := strings.Count(out, "unused-local"); got != n {
			t.Fatalf("run %d: expected %d unused-local findings, got %d: %q", run, n, got, out)
		}
	}
}

func TestCLIHonorsSuppressionInLoadedInclude(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	includeSource := "// pawnlint-disable-next-line duplicate-function-definition\nShared() {}\n"
	if err := os.WriteFile(includePath, []byte(includeSource), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nShared() {}\nmain() {}\n")
	if err := os.WriteFile(mainPath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--format", "compact", mainPath}, "")
	if code != 0 || strings.Contains(out, "duplicate-function-definition") {
		t.Fatalf("include suppression was ignored: code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLICrossFileReferencesPreventUnusedFindings(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	includeSource := "UseRoot() { RootHelper(); root_value = 1; }\n"
	if err := os.WriteFile(includePath, []byte(includeSource), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nnew root_value;\nRootHelper() {}\nmain() { UseRoot(); }\n")
	if err := os.WriteFile(mainPath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--profile", "strict", "--format", "compact", mainPath}, "")
	if code != 0 || strings.Contains(out, "unused-function") || strings.Contains(out, "unused-global") {
		t.Fatalf("cross-file references were ignored: code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLIStrictProfileRunsNativeRulesForOpenMPTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.pwn")
	source := []byte("main() { new name[8]; Kick(); SendRconCommandf(\"echo ready\"); printf(\"%d\"); GetPlayerName(0, name, 16); EnableTirePopping(false); GetPlayerPoolSize(); }\n")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--profile", "strict", "--format", "compact", path}, "")
	if code != 1 || !strings.Contains(out, "native-argument-count") || !strings.Contains(out, "deprecated-native") || !strings.Contains(out, "deprecated-function") || !strings.Contains(out, "format-argument-count") || !strings.Contains(out, "buffer-size") || !strings.Contains(out, "unimplemented-function") {
		t.Fatalf("native rules did not run: code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLIStrictProfileChecksTargetNativesForSAMPTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.pwn")
	if err := os.WriteFile(path, []byte("main() { new DB:forgotten = db_open(\"forgotten.db\"); new DB:overwritten = db_open(\"first.db\"); overwritten = db_open(\"second.db\"); SetPlayerAdmin(0, true); db_open(\"server.db\"); fclose(DB:1); return CAM_MODE_FIXED; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--profile", "strict", "--target", "samp", "--format", "compact", path}, "")
	if code != 1 || !strings.Contains(out, "target-native-availability") || !strings.Contains(out, "target-constant-availability") || !strings.Contains(out, "discarded-resource-handle") || !strings.Contains(out, "mismatched-resource-handle") || !strings.Contains(out, "unreleased-resource-handle") || !strings.Contains(out, "overwritten-resource-handle") {
		t.Fatalf("target API rules did not run: code=%d output=%q stderr=%q", code, out, stderr)
	}
}

func TestCLIStrictEnablesUnusedLocal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "unused.pwn")
	if err := os.WriteFile(p, []byte("main() { new unused; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--profile", "strict", "--format", "compact", p}, "")
	if code != 0 {
		t.Fatalf("unused warning code=%d", code)
	}
	if !strings.Contains(out, "unused-local") {
		t.Fatalf("strict semantic rule did not run: %q", out)
	}
}

func TestCLIUsesConfiguredDefines(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(cfg, []byte("defines = [\"FEATURE\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "conditional.pwn")
	src := "main() {\n#if defined FEATURE\nnew value = 1 / 0;\n#endif\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--config", cfg, "--format", "compact", p}, "")
	if code != 1 || !strings.Contains(out, "division-by-zero") {
		t.Fatalf("configured define was not used: code=%d output=%q", code, out)
	}
}

func TestCLIPathScopedOverrideAppliesOnlyToMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	cfgSrc := "profile = \"strict\"\n" +
		"[[overrides]]\n" +
		"paths = [\"testdata/**\"]\n" +
		"[overrides.rules]\n" +
		"unused-local = \"off\"\n"
	if err := os.WriteFile(cfg, []byte(cfgSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := "main() {\nnew value = 1;\n}\n"
	fixture := filepath.Join(dir, "testdata", "fixture.pwn")
	if err := os.WriteFile(fixture, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(main, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, _ := runCLI(t, []string{"--config", cfg, "--format", "compact", fixture, main}, "")
	if n := strings.Count(out, "unused-local"); n != 1 {
		t.Fatalf("expected exactly 1 unused-local finding (main.pwn only, testdata/ overridden off), got %d: %q", n, out)
	}
	if strings.Contains(out, "fixture.pwn") {
		t.Fatalf("testdata/fixture.pwn must not be flagged: %q", out)
	}
	if !strings.Contains(out, "main.pwn") {
		t.Fatalf("main.pwn should still be flagged: %q", out)
	}
}

func TestCLIWithoutVariantsMissesTheUntestedTargetBranch(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(cfg, []byte("defines = [\"OPENMP\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "conditional.pwn")
	src := "main() {\n#if defined SAMP\nnew value = 1 / 0;\n#endif\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--config", cfg, "--format", "compact", p}, "")
	if code != 0 || strings.Contains(out, "division-by-zero") {
		t.Fatalf("expected the SAMP-only bug to go undetected without variants: code=%d output=%q", code, out)
	}
}

func TestCLIVariantsAnalyzeEveryTargetBranch(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	cfgSrc := "defines = [\"OPENMP\"]\n" +
		"[[variants]]\n" +
		"name = \"openmp\"\n" +
		"defines = [\"OPENMP\"]\n" +
		"[[variants]]\n" +
		"name = \"samp\"\n" +
		"defines = [\"SAMP\"]\n"
	if err := os.WriteFile(cfg, []byte(cfgSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "conditional.pwn")
	src := "main() {\n#if defined SAMP\nnew value = 1 / 0;\n#endif\n#if defined OPENMP\nnew other = 1 / 0;\n#endif\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--config", cfg, "--format", "compact", p}, "")
	if code != 1 {
		t.Fatalf("expected findings from both variants: code=%d output=%q", code, out)
	}
	if n := strings.Count(out, "division-by-zero"); n != 2 {
		t.Fatalf("expected exactly 2 division-by-zero findings (one per branch, deduplicated across variants), got %d: %q", n, out)
	}
}

func TestCLIVariantsOnlyReportUnusedSuppressionWhenEveryVariantAgrees(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	cfgSrc := "profile = \"strict\"\n" +
		"[[variants]]\n" +
		"name = \"openmp\"\n" +
		"defines = [\"OPENMP\"]\n" +
		"[[variants]]\n" +
		"name = \"samp\"\n" +
		"defines = [\"SAMP\"]\n"
	if err := os.WriteFile(cfg, []byte(cfgSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "conditional.pwn")
	src := "main() {\n#if defined SAMP\n// pawnlint-disable-next-line division-by-zero\nnew value = 1 / 0;\n#endif\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--config", cfg, "--format", "compact", p}, "")
	if code != 0 || strings.Contains(out, "division-by-zero") || strings.Contains(out, "unused pawnlint suppression") {
		t.Fatalf("suppression is genuinely used under the samp variant; must not be reported unused just because the openmp variant never reaches that branch: code=%d output=%q", code, out)
	}
}

func TestCLIVariantsStillReportSuppressionUnusedInEveryVariant(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	cfgSrc := "profile = \"strict\"\n" +
		"[[variants]]\n" +
		"name = \"openmp\"\n" +
		"defines = [\"OPENMP\"]\n" +
		"[[variants]]\n" +
		"name = \"samp\"\n" +
		"defines = [\"SAMP\"]\n"
	if err := os.WriteFile(cfg, []byte(cfgSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "conditional.pwn")
	src := "main() {\n// pawnlint-disable-next-line division-by-zero\nnew value = 1;\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, _ := runCLI(t, []string{"--config", cfg, "--format", "compact", p}, "")
	if !strings.Contains(out, "unused pawnlint suppression") {
		t.Fatalf("a suppression unused in every variant must still be reported: output=%q", out)
	}
}

func TestCLIVariantsRequireNonEmptyUniqueNames(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	cfgSrc := "[[variants]]\nname = \"a\"\ndefines = [\"X\"]\n[[variants]]\nname = \"a\"\ndefines = [\"Y\"]\n"
	if err := os.WriteFile(cfg, []byte(cfgSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(p, []byte("main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, code := runCLI(t, []string{"--config", cfg, p}, "")
	if code == 0 || !strings.Contains(errOut, "duplicate variant") {
		t.Fatalf("expected duplicate variant name to be rejected: code=%d stderr=%q", code, errOut)
	}
}

func TestCLIUsesConfiguredBuildContextWithoutPaths(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	includeDir := filepath.Join(serverDir, "includes")
	dependencyDir := filepath.Join(dir, "dependencies", "library")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dependencyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(serverDir, "main.pwn")
	entrySource := "#include <shared>\n#include \"includes/local.inc\"\nShared() {}\nmain() {\n#if defined FEATURE\nnew value = 1 / 0;\n#endif\nSetPlayerAdmin(0, true);\n}\n"
	if err := os.WriteFile(entry, []byte(entrySource), 0o644); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(includeDir, "local.inc")
	if err := os.WriteFile(local, []byte("Local() { new local = 1 / 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dependency := filepath.Join(dependencyDir, "shared.inc")
	if err := os.WriteFile(dependency, []byte("Shared() { new dependency = 1 / 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "pawnlint.toml")
	configSource := `profile = "strict"

[[builds]]
name = "main"
entry = "main.pwn"
working-directory = "server"
files = ["includes/**"]
include-paths = ["../dependencies/library"]
defines = ["FEATURE"]
target = "samp"
`
	if err := os.WriteFile(configPath, []byte(configSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--config", configPath, "--format", "compact"}, "")
	if code != 1 || stderr != "" {
		t.Fatalf("code=%d stderr=%q output=%q", code, stderr, out)
	}
	if !strings.Contains(out, "duplicate-function-definition") {
		t.Fatalf("build include path was not used: %q", out)
	}
	if !strings.Contains(out, "target-native-availability") {
		t.Fatalf("build target was not used: %q", out)
	}
	if count := strings.Count(out, "division-by-zero"); count != 2 {
		t.Fatalf("expected entry and selected local include to be linted, but not dependency include; count=%d output=%q", count, out)
	}
}

func TestCLIConfiguredBuildsDeduplicateSharedDiagnostics(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(shared, []byte("Shared() { new value = 1 / 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second"} {
		entry := filepath.Join(dir, name+".pwn")
		if err := os.WriteFile(entry, []byte("#include \"shared.inc\"\nmain() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	configPath := filepath.Join(dir, "pawnlint.toml")
	configSource := `profile = "strict"

[[builds]]
name = "first"
entry = "first.pwn"
files = ["shared.inc"]

[[builds]]
name = "second"
entry = "second.pwn"
files = ["shared.inc"]
`
	if err := os.WriteFile(configPath, []byte(configSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--config", configPath, "--format", "compact"}, "")
	if code != 1 || stderr != "" {
		t.Fatalf("code=%d stderr=%q output=%q", code, stderr, out)
	}
	if count := strings.Count(out, "division-by-zero"); count != 1 {
		t.Fatalf("shared diagnostic count=%d output=%q", count, out)
	}
}

func TestCLIConfiguredBuildLintsIncludedFileInIncomingMacroContext(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.pwn")
	include := filepath.Join(dir, "feature.inc")
	if err := os.WriteFile(entry, []byte("#define FEATURE\n#include \"feature.inc\"\nmain() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	includeSource := "#if defined FEATURE\nFeature() { new value = 1 / 0; }\n#endif\n"
	if err := os.WriteFile(include, []byte(includeSource), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "pawnlint.toml")
	configSource := `profile = "strict"

[[builds]]
name = "main"
entry = "main.pwn"
files = ["feature.inc"]
`
	if err := os.WriteFile(configPath, []byte(configSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runCLI(t, []string{"--config", configPath, "--format", "compact"}, "")
	if code != 1 || stderr != "" || !strings.Contains(out, "feature.inc") || !strings.Contains(out, "division-by-zero") {
		t.Fatalf("code=%d stderr=%q output=%q", code, stderr, out)
	}
}

func TestCLIMaxDiagnosticsLimitsOutputWithoutChangingThreshold(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(cfg, []byte("profile = \"all\"\n[lint]\nmax-diagnostics = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "bad.pwn")
	if err := os.WriteFile(p, []byte("main()\n{\n a + 1;\n b + 2;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCLI(t, []string{"--config", cfg, "--format", "compact", p}, "")
	if code != 0 {
		t.Fatalf("warnings should not fail without warnings-as-errors: code=%d", code)
	}
	if lines := strings.Count(strings.TrimSpace(out), "\n") + 1; lines != 1 {
		t.Fatalf("max-diagnostics emitted %d lines: %q", lines, out)
	}
}
