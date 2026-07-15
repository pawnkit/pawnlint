package rules_test

import (
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

func TestProjectOwnershipSummaries(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "resource.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	include := `stock File:OpenResource()
{
	return File:1;
}

stock CloseResource(File:resource)
{
	return _:resource;
}

stock InspectResource(File:resource)
{
	return _:resource;
}
`
	main := `#include "resource.inc"

main()
{
	OpenResource();
	new File:leaked = OpenResource();
	InspectResource(leaked);
	new File:closed = OpenResource();
	CloseResource(closed);
	InspectResource(closed);
	new DB:wrong;
	CloseResource(wrong);
}
`
	if err := os.WriteFile(includePath, []byte(include), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	model, err := project.Build([]project.Source{{Path: mainPath, Content: []byte(main)}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := api.Merge("openmp", &api.Metadata{Functions: map[string]api.Function{
		"OpenResource":    {ReturnTag: "File", Release: "CloseResource"},
		"CloseResource":   {Parameters: []api.Parameter{{Name: "resource", Tag: "File", Ownership: "transferred"}}},
		"InspectResource": {Parameters: []api.Parameter{{Name: "resource", Tag: "File", Ownership: "borrowed"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"discarded-resource-handle":  "returned by \"OpenResource\" is discarded",
		"unreleased-resource-handle": "resource handle \"leaked\"",
		"read-after-release":         "resource handle \"closed\" is used after release",
		"mismatched-resource-handle": "\"CloseResource\" releases File handles, but this argument has tag DB",
	}
	for ruleID, message := range want {
		t.Run(ruleID, func(t *testing.T) {
			diagnostics := lintProjectRule(t, model, metadata, mainPath, ruleID)
			if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, message) {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
		})
	}
}

func lintProjectRule(t *testing.T, model *project.Model, metadata *api.Metadata, path, ruleID string) []diagnostic.Diagnostic {
	t.Helper()
	registry := rules.Default()
	rule, ok := registry.Lookup(ruleID)
	if !ok {
		t.Fatalf("unknown rule %s", ruleID)
	}
	known := make(map[string]struct{})
	for _, id := range registry.IDs() {
		known[id] = struct{}{}
	}
	engine := lint.NewEngine(registry)
	engine.Project = model
	engine.API = metadata
	diagnostics := engine.LintProjectFile(model.File(path), rule.AnalysisLevel, map[string]diagnostic.Severity{ruleID: rule.DefaultSeverity}, known, nil)
	var result []diagnostic.Diagnostic
	for _, item := range diagnostics {
		if item.RuleID == ruleID {
			result = append(result, item)
		}
	}
	return result
}
