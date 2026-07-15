package baseline_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pawnkit/pawnlint/internal/baseline"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func TestGenerateApplyAndPrune(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	content := []byte("main() { value = 1 / 0; other = 1 / 0; }\n")
	sources := map[string][]byte{path: content}
	table := source.NewLineTable(content)
	diagnostics := []diagnostic.Diagnostic{
		{RuleID: "division-by-zero", Message: "division by zero", Filename: path, Range: table.Range(17, 22)},
		{RuleID: "division-by-zero", Message: "division by zero", Filename: path, Range: table.Range(32, 37)},
		{RuleID: "parse-error", Message: "expected expression", Filename: path, Range: table.Range(0, 4)},
		{RuleID: "deprecated-rule-id", Message: "deprecated rule", Filename: path, Range: table.Range(0, 4)},
	}
	generated := baseline.Generate(diagnostics, sources, dir)
	if len(generated.Entries) != 2 || generated.Entries[0].Fingerprint == generated.Entries[1].Fingerprint || generated.Entries[0].Path != "main.pwn" {
		t.Fatalf("generated = %#v", generated)
	}
	match := baseline.Apply(generated, diagnostics[:2], sources, dir)
	if len(match.Remaining) != 0 || match.Stale != 0 || !reflect.DeepEqual(match.Current, generated) {
		t.Fatalf("match = %#v", match)
	}
	match = baseline.Apply(generated, diagnostics[:1], sources, dir)
	if len(match.Remaining) != 0 || match.Stale != 1 || len(match.Current.Entries) != 1 {
		t.Fatalf("pruned match = %#v", match)
	}
}

func TestFingerprintSurvivesLineShift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	first := []byte("main() { value = 1 / 0; }\n")
	second := []byte("\n\nmain() { value = 1 / 0; }\n")
	firstDiagnostic := diagnostic.Diagnostic{RuleID: "division-by-zero", Message: "division by zero", Filename: path, Range: source.NewLineTable(first).Range(17, 22)}
	secondDiagnostic := diagnostic.Diagnostic{RuleID: "division-by-zero", Message: "division by zero", Filename: path, Range: source.NewLineTable(second).Range(19, 24)}
	file := baseline.Generate([]diagnostic.Diagnostic{firstDiagnostic}, map[string][]byte{path: first}, dir)
	match := baseline.Apply(file, []diagnostic.Diagnostic{secondDiagnostic}, map[string][]byte{path: second}, dir)
	if len(match.Remaining) != 0 || match.Stale != 0 || match.Current.Entries[0].Line != 3 {
		t.Fatalf("match = %#v", match)
	}
}

func TestLoadRejectsInvalidBaseline(t *testing.T) {
	for _, content := range []string{
		`{"version":2,"entries":[]}`,
		`{"version":1,"entries":[{"fingerprint":"bad","ruleId":"rule","path":"main.pwn","message":"message","line":1}]}`,
		`{"version":1,"entries":[],"unknown":true}`,
	} {
		path := filepath.Join(t.TempDir(), "baseline.json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := baseline.Load(path); err == nil {
			t.Fatalf("invalid baseline accepted: %s", content)
		}
	}
}

func TestWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "baseline.json")
	file := baseline.File{Version: baseline.Version, Entries: []baseline.Entry{{
		Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RuleID:      "rule",
		Path:        "main.pwn",
		Message:     "message",
		Line:        1,
	}}}
	if err := baseline.Write(path, file); err != nil {
		t.Fatal(err)
	}
	loaded, err := baseline.Load(path)
	if err != nil || !reflect.DeepEqual(loaded, file) {
		t.Fatalf("loaded = %#v, err = %v", loaded, err)
	}
}
