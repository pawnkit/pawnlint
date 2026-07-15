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

func TestCallGraphAddsCallbackEntriesAndTimerEdges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`public OnGameModeInit()
{
	SetTimer("Tick", 1000, false);
	return 1;
}

public Tick()
{
	SetTimer("Tick", 1000, false);
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	graph := model.CallGraph
	if len(graph.EntryPoints) != 2 || len(graph.AsyncCalls) != 2 {
		t.Fatalf("entries=%#v async=%#v", graph.EntryPoints, graph.AsyncCalls)
	}
	for _, call := range graph.AsyncCalls {
		if call.Kind != project.CallTimer || call.Callee.Name != "Tick" {
			t.Fatalf("call = %#v", call)
		}
	}
	if len(graph.RecursiveComponents()) != 0 {
		t.Fatalf("timer scheduling was treated as recursion: %#v", graph.RecursiveComponents())
	}
	var tick project.Declaration
	for _, function := range graph.Functions {
		if function.Name == "Tick" {
			tick = function
		}
	}
	if outgoing := graph.AsyncOutgoing(tick); len(outgoing) != 1 || outgoing[0].Callee.Name != "Tick" {
		t.Fatalf("outgoing = %#v", outgoing)
	}
}

func TestCallGraphResolvesMacroTimerCallback(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "timer.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(includePath, []byte("#define TIMER_CALLBACK \"Tick\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := []byte(`#include "timer.inc"
main()
{
	SetTimer(TIMER_CALLBACK, 1000, false);
}

public Tick()
{
}
`)
	model, err := project.Build([]project.Source{{Path: mainPath, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.CallGraph.AsyncCalls) != 1 || model.CallGraph.AsyncCalls[0].Callee.Name != "Tick" {
		t.Fatalf("async calls = %#v", model.CallGraph.AsyncCalls)
	}
	origins := model.ExpansionOrigins(model.File(mainPath), model.CallGraph.AsyncCalls[0].Node)
	if len(origins) == 0 {
		t.Fatal("timer edge lost macro origin")
	}
}

func TestCallGraphRetainsMacroFactsAfterReleasingExpandedTrees(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`#define TIMER_CALLBACK "Tick"
main()
{
	SetTimer(TIMER_CALLBACK, 1000, false);
}

public Tick()
{
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true})
	if err != nil {
		t.Fatal(err)
	}
	file := model.File(path)
	if file.ExpandedSource != nil || file.ExpandedParsed != nil || file.ExpandedWalk != nil || file.ExpandedSemantic != nil {
		t.Fatal("expanded trees were retained")
	}
	if len(model.CallGraph.AsyncCalls) != 1 || model.CallGraph.AsyncCalls[0].Callee.Name != "Tick" {
		t.Fatalf("async calls = %#v", model.CallGraph.AsyncCalls)
	}
	if origins := model.ExpansionOrigins(file, model.CallGraph.AsyncCalls[0].Node); len(origins) == 0 {
		t.Fatalf("origins = %#v", origins)
	}
}

func TestCallGraphDetachesGeneratedRuntimeCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`#define START_TIMER() SetTimer("Tick", 1000, false)
main()
{
	START_TIMER();
}

public Tick()
{
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.CallGraph.AsyncCalls) != 1 {
		t.Fatalf("async calls = %#v", model.CallGraph.AsyncCalls)
	}
	call := model.CallGraph.AsyncCalls[0]
	if call.Node == nil || call.Node.Start < 0 || call.Node.End > len(source) {
		t.Fatalf("call node = %#v", call.Node)
	}
	if origins := model.ExpansionOrigins(model.File(path), call.Node); len(origins) == 0 {
		t.Fatal("generated call lost its origin")
	}
}

func TestCallGraphResolvesDynamicFunctionCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`main()
{
	CallLocalFunction("Dispatch", "i", 1);
}

public Dispatch(value)
{
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	var mainFunction project.Declaration
	for _, function := range model.CallGraph.Functions {
		if function.Name == "main" {
			mainFunction = function
		}
	}
	calls := model.CallGraph.Outgoing(mainFunction)
	if len(calls) != 1 || calls[0].Kind != project.CallDynamic || calls[0].Callee.Name != "Dispatch" || calls[0].ArgumentOffset != 2 {
		t.Fatalf("calls = %#v", calls)
	}
}
