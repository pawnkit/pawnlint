package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestCallGraphFindsCrossFileRecursion(t *testing.T) {
	dir := t.TempDir()
	include := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(include, []byte("stock Second() { First(); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nmain() { First(); }\nstock First() { Second(); }\n")
	model, err := project.Build([]project.Source{{Path: mainPath, Content: source}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if model.CallGraph == nil || len(model.CallGraph.Calls) != 3 {
		t.Fatalf("call graph = %#v", model.CallGraph)
	}
	components := model.CallGraph.RecursiveComponents()
	if len(components) != 1 || len(components[0]) != 2 {
		t.Fatalf("recursive components = %#v", components)
	}
	if components[0][0].Name != "First" || components[0][1].Name != "Second" {
		t.Fatalf("component = %#v", components[0])
	}
}

func TestCallGraphFindsDirectRecursion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte("forward Recur();\nmain() { Recur(); }\nstock Recur() { Recur(); }\n")
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	components := model.CallGraph.RecursiveComponents()
	if len(components) != 1 || len(components[0]) != 1 || components[0][0].Name != "Recur" {
		t.Fatalf("recursive components = %#v", components)
	}
	components[0][0].Name = "Changed"
	if next := model.CallGraph.RecursiveComponents(); next[0][0].Name != "Recur" {
		t.Fatalf("cached components were mutated: %#v", next)
	}
}

func TestCallGraphKeepsSharedCSTContextsIndependent(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	includeSource := []byte("Helper() {}\nShared() { Helper(); }\n")
	if err := os.WriteFile(includePath, includeSource, 0o644); err != nil {
		t.Fatal(err)
	}
	onePath := filepath.Join(dir, "one.pwn")
	twoPath := filepath.Join(dir, "two.pwn")
	oneSource := []byte("#define ONE\n#include \"shared.inc\"\n")
	twoSource := []byte("#define TWO\n#include \"shared.inc\"\n")
	model, err := project.Build([]project.Source{{Path: onePath, Content: oneSource}, {Path: twoPath, Content: twoSource}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.CallGraph.Calls) != 2 {
		t.Fatalf("calls = %#v", model.CallGraph.Calls)
	}
	for _, call := range model.CallGraph.Calls {
		if call.File != call.Caller.File || call.File != call.Callee.File {
			t.Fatalf("call crossed semantic contexts: %#v", call)
		}
	}
}
