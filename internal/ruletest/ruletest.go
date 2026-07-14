package ruletest

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

var update = flag.Bool("update", false, "regenerate expected.json golden files")

type SnapshotEntry struct {
	RuleID    string         `json:"ruleId"`
	Severity  string         `json:"severity"`
	Message   string         `json:"message"`
	StartLine int            `json:"startLine"`
	StartCol  int            `json:"startCol"`
	EndLine   int            `json:"endLine"`
	EndCol    int            `json:"endCol"`
	Notes     []SnapshotNote `json:"notes,omitempty"`
	Suggested string         `json:"suggested,omitempty"`
	Fix       *SnapshotFix   `json:"fix,omitempty"`
}

type SnapshotNote struct {
	Message   string `json:"message"`
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startCol"`
	EndLine   int    `json:"endLine"`
	EndCol    int    `json:"endCol"`
}

type SnapshotFix struct {
	Description string         `json:"description"`
	Edits       []SnapshotEdit `json:"edits"`
}

type SnapshotEdit struct {
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	NewText     string `json:"newText"`
}

func SilentRegistrar() *lint.Registrar {
	return rules.Default()
}

func RunRule(t *testing.T, path string, ruleID string, src []byte) []diagnostic.Diagnostic {
	return runRule(t, path, ruleID, src, "")
}

func runRule(t *testing.T, path string, ruleID string, src []byte, target string) []diagnostic.Diagnostic {
	t.Helper()
	reg := rules.Default()
	m, ok := reg.Lookup(ruleID)
	if !ok {
		t.Fatalf("unknown rule %q", ruleID)
	}
	ruleSet := map[string]diagnostic.Severity{ruleID: m.DefaultSeverity}
	known := map[string]struct{}{}
	for _, id := range reg.IDs() {
		known[id] = struct{}{}
	}
	engine := lint.NewEngine(reg)
	engine.Target = target
	if m.AnalysisLevel == lint.ProjectAnalysis {
		model, err := project.Build([]project.Source{{Path: path, Content: src}}, project.Options{})
		if err != nil {
			t.Fatalf("build project: %v", err)
		}
		engine.Project = model
	}
	return engine.LintFile(path, src, m.AnalysisLevel, ruleSet, known, nil)
}

func RunSnapshot(t *testing.T, ruleID, fixtureDir string) {
	t.Helper()
	dir := fixtureDir
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no fixtures for %s at %s", ruleID, dir)
	}
	fixtureNames := []string{"valid.pwn", "invalid.pwn", "edge-cases.pwn"}
	target := ""
	if value, err := os.ReadFile(filepath.Join(dir, "target.txt")); err == nil {
		target = strings.TrimSpace(string(value))
	} else if !os.IsNotExist(err) {
		t.Fatalf("read target: %v", err)
	}
	got := map[string][]SnapshotEntry{}
	for _, name := range fixtureNames {
		full := filepath.Join(dir, name)
		if _, err := os.Stat(full); err != nil {
			continue
		}
		src, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("read %s: %v", full, err)
		}
		diags := runRule(t, "in/"+ruleID+"/"+name, ruleID, src, target)
		entries := make([]SnapshotEntry, 0, len(diags))
		for _, d := range diags {
			entry := SnapshotEntry{
				RuleID:    d.RuleID,
				Severity:  d.Severity.String(),
				Message:   d.Message,
				StartLine: d.Range.Start.Line,
				StartCol:  d.Range.Start.Col,
				EndLine:   d.Range.End.Line,
				EndCol:    d.Range.End.Col,
				Suggested: d.Suggested,
			}
			for _, note := range d.Notes {
				entry.Notes = append(entry.Notes, SnapshotNote{
					Message:   note.Message,
					StartLine: note.Range.Start.Line,
					StartCol:  note.Range.Start.Col,
					EndLine:   note.Range.End.Line,
					EndCol:    note.Range.End.Col,
				})
			}
			if d.Fix != nil {
				entry.Fix = &SnapshotFix{Description: d.Fix.Description}
				for _, edit := range d.Fix.Edits {
					entry.Fix.Edits = append(entry.Fix.Edits, SnapshotEdit{
						StartOffset: edit.Range.Start.Offset,
						EndOffset:   edit.Range.End.Offset,
						NewText:     edit.NewText,
					})
				}
			}
			entries = append(entries, entry)
		}
		got[name] = entries
	}
	expPath := filepath.Join(dir, "expected.json")
	if *update {
		b, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		if err := os.WriteFile(expPath, b, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", expPath)
		return
	}
	exp, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatalf("read expected.json: %v (run with -update to generate)", err)
	}
	var want map[string][]SnapshotEntry
	if err := json.Unmarshal(exp, &want); err != nil {
		t.Fatalf("parse expected.json: %v", err)
	}
	compareSnapshot(t, want, got)
}

func compareSnapshot(t *testing.T, want, got map[string][]SnapshotEntry) {
	t.Helper()
	for file := range got {
		if _, ok := want[file]; !ok {
			want[file] = nil
		}
	}
	for file, wents := range want {
		gents := got[file]
		if len(wents) != len(gents) {
			t.Errorf("%s: %d diagnostics, want %d\nGOT:\n%s\nWANT:\n%s", file, len(gents), len(wents), pretty(gents), pretty(wents))
			continue
		}
		for i := range wents {
			if !reflect.DeepEqual(wents[i], gents[i]) {
				t.Errorf("%s diagnostic %d:\n GOT: %+v\nWANT: %+v", file, i, gents[i], wents[i])
			}
		}
	}
}

func pretty(s []SnapshotEntry) string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}
