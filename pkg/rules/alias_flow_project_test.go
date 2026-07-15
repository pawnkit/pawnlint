package rules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestLocalFlowUsesProjectMutationEffects(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "effects.inc")
	include := []byte("Read(value) { return value; }\nMutate(&value) { value = 2; }\nWrapper(&value) { Mutate(value); }\n")
	if err := os.WriteFile(includePath, include, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`#include "effects.inc"
main()
{
	new value = 1;
	Read(value);
	if (value == 1) {}
	Wrapper(value);
	if (value == 1) {}
}
`)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := lintProjectRule(t, model, nil, path, "constant-condition")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "always true") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}
