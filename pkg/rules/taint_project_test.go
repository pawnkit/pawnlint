package rules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestTaintedDataCrossesProjectFunction(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "query.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	include := `ForwardInput(const value[])
{
    SQL_Query(value);
}
`
	main := `#include "query.inc"

public OnPluginInput(playerid, const text[])
{
    ForwardInput(text);
    return playerid;
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
	metadata, err := api.Merge("openmp", &api.Metadata{
		Callbacks: map[string]api.Callback{
			"OnPluginInput": {Parameters: []api.Parameter{{Name: "playerid"}, {Name: "text", ArrayRank: 1, Const: true, TaintSource: "player-input"}}},
		},
		Natives: map[string]api.Native{
			"SQL_Query": {Parameters: []api.Parameter{{Name: "query", ArrayRank: 1, Const: true, TaintSink: "sql"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, metadata, mainPath, "tainted-data-to-sink")
	if len(diagnostics) != 1 || diagnostics[0].Filename != includePath || !strings.Contains(diagnostics[0].Message, `"player-input" reaches "sql"`) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestTaintedDataReturnsAndWritesAcrossProjectFunctions(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "transform.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	include := `Identity(value)
{
	return value;
}

Forward(value)
{
	return Identity(value);
}

CopyOutput(&output, value)
{
	output = Identity(value);
}

ReadAndReturn()
{
	new value;
	ReadSource(value);
	return value;
}
`
	main := `#include "transform.inc"

public OnPluginInput(playerid, value)
{
	Dangerous(Forward(0));
	Dangerous(Forward(value));
	new output;
	CopyOutput(output, value);
	Dangerous(output);
	Dangerous(ReadAndReturn());
	return playerid;
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
	metadata, err := api.Merge("openmp", &api.Metadata{
		Callbacks: map[string]api.Callback{
			"OnPluginInput": {Parameters: []api.Parameter{{Name: "playerid"}, {Name: "value", TaintSource: "player-input"}}},
		},
		Natives: map[string]api.Native{
			"Dangerous":  {Parameters: []api.Parameter{{Name: "value", TaintSink: "command"}}},
			"ReadSource": {Parameters: []api.Parameter{{Name: "value", Reference: true, Output: true, TaintSource: "network-input"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, metadata, mainPath, "tainted-data-to-sink")
	if len(diagnostics) != 3 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	counts := map[string]int{"player-input": 0, "network-input": 0}
	for _, item := range diagnostics {
		if item.Filename != mainPath || !strings.Contains(item.Message, `reaches "command"`) {
			t.Fatalf("diagnostic = %#v", item)
		}
		for source := range counts {
			if strings.Contains(item.Message, `"`+source+`"`) {
				counts[source]++
			}
		}
	}
	if counts["player-input"] != 2 || counts["network-input"] != 1 {
		t.Fatalf("source counts = %#v", counts)
	}
}

func TestTaintedDataCrossesTimerEdge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`public OnPluginInput(value)
{
	SetTimerEx("Delayed", 1000, false, "i", value);
}

public Delayed(value)
{
	Dangerous(value);
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := api.Merge("openmp", &api.Metadata{
		Callbacks: map[string]api.Callback{
			"OnPluginInput": {Parameters: []api.Parameter{{Name: "value", TaintSource: "player-input"}}},
		},
		Natives: map[string]api.Native{
			"Dangerous": {Parameters: []api.Parameter{{Name: "value", TaintSink: "command"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, metadata, path, "tainted-data-to-sink")
	if len(diagnostics) != 1 || diagnostics[0].Filename != path || !strings.Contains(diagnostics[0].Message, `"player-input" reaches "command"`) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestTaintedDataCrossesDynamicFunctionCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`public OnPluginInput(value)
{
	CallLocalFunction("Dispatch", "i", value);
}

public Dispatch(value)
{
	Dangerous(value);
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := api.Merge("openmp", &api.Metadata{
		Callbacks: map[string]api.Callback{
			"OnPluginInput": {Parameters: []api.Parameter{{Name: "value", TaintSource: "player-input"}}},
		},
		Natives: map[string]api.Native{
			"Dangerous": {Parameters: []api.Parameter{{Name: "value", TaintSink: "command"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, metadata, path, "tainted-data-to-sink")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, `"player-input" reaches "command"`) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}
