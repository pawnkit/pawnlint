package rules_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestLoopInvariantCallUsesProjectEffects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`new shared;
Pure(value) { return value * 2; }
Impure() { return shared; }
main()
{
	for (new index = 0; index < 10; index++)
	{
		new pure = Pure(4);
		new impure = Impure();
	}
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, nil, path, "loop-invariant-call")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, `pure call "Pure"`) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}
