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
