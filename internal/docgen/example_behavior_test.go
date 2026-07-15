package docgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestCuratedRuleExamplesMatchRule(t *testing.T) {
	registry := rules.Default()
	known := make(map[string]struct{})
	for _, id := range registry.IDs() {
		known[id] = struct{}{}
	}
	for _, metadata := range registry.Sorted() {
		invalid, hasInvalid := curatedExample(metadata.ID, "invalid")
		valid, hasValid := curatedExample(metadata.ID, "valid")
		if !hasInvalid && !hasValid {
			continue
		}
		t.Run(metadata.ID, func(t *testing.T) {
			if hasInvalid {
				diagnostics := lintExample(t, metadata, invalid, known)
				if len(diagnostics) == 0 {
					t.Error("bad example produced no diagnostic")
				}
			}
			if hasValid {
				diagnostics := lintExample(t, metadata, valid, known)
				if len(diagnostics) != 0 {
					t.Errorf("good example produced %d diagnostics: %v", len(diagnostics), diagnostics)
				}
			}
		})
	}
}

func curatedExample(id, name string) (string, bool) {
	content, err := os.ReadFile(filepath.Join(testdataRulesDir(), id, "example-"+name+".pwn"))
	if err != nil {
		return "", false
	}
	return string(content), true
}

func lintExample(t *testing.T, metadata lint.Metadata, source string, known map[string]struct{}) []diagnostic.Diagnostic {
	t.Helper()
	dir := filepath.Join(testdataRulesDir(), metadata.ID)
	target := readText(filepath.Join(dir, "target.txt"))
	customAPI := loadExampleAPI(t, filepath.Join(dir, "api.json"))
	options := loadExampleOptions(t, metadata, filepath.Join(dir, "options.json"))
	engine := lint.NewEngine(rules.Default())
	engine.Target = target
	engine.API = customAPI
	path := filepath.Join(dir, "example-invalid.pwn")
	if metadata.AnalysisLevel == lint.ProjectAnalysis {
		_, completeErr := os.Stat(filepath.Join(dir, "defines-complete.txt"))
		model, err := project.Build([]project.Source{{Path: path, Content: []byte(source)}}, project.Options{WorkingDir: dir, IncludePaths: []string{dir}, DefinesComplete: completeErr == nil})
		if err != nil {
			t.Fatal(err)
		}
		engine.Project = model
	}
	severity := map[string]diagnostic.Severity{metadata.ID: metadata.DefaultSeverity}
	if metadata.ID == "unknown-suppression" {
		severity["discarded-expression"] = diagnostic.SeverityWarning
	}
	configured := map[string]map[string]any{metadata.ID: options}
	diagnostics := engine.LintFile(path, []byte(source), metadata.AnalysisLevel, severity, known, configured)
	filtered := diagnostics[:0]
	for _, item := range diagnostics {
		if item.RuleID == metadata.ID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func readText(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func loadExampleAPI(t *testing.T, path string) *api.Metadata {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	metadata, err := api.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return metadata
}

func loadExampleOptions(t *testing.T, metadata lint.Metadata, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		t.Fatal(err)
	}
	definitions := make(map[string]lint.Option)
	for _, option := range metadata.Options {
		definitions[option.Name] = option
	}
	result := make(map[string]any)
	for name, value := range raw {
		option, ok := definitions[name]
		if !ok {
			t.Fatalf("unknown option %q", name)
		}
		normalized, err := lint.NormalizeOption(option, value)
		if err != nil {
			t.Fatal(err)
		}
		result[name] = normalized
	}
	return result
}
