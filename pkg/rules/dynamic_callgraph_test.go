package rules_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestTimerTargetIsUsedFunction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`main()
{
	SetTimer("Hidden", 1000, false);
}

static Hidden()
{
}

static Dead()
{
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, nil, path, "unused-function")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, `"Dead"`) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}
