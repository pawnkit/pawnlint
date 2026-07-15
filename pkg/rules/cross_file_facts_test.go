package rules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestRulesUseCrossFileConstantsAndTags(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "facts.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	include := "const Enabled = 1;\nnew Float:distance;\n"
	main := "#include \"facts.inc\"\nUse(value) {}\nmain() { if (Enabled) {} Use(distance); }\n"
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
	conditions := lintProjectRule(t, model, nil, mainPath, "constant-condition")
	if len(conditions) != 1 || !strings.Contains(conditions[0].Message, "always true") {
		t.Fatalf("constant diagnostics = %#v", conditions)
	}
	tags := lintProjectRule(t, model, nil, mainPath, "argument-tag-mismatch")
	if len(tags) != 1 || !strings.Contains(tags[0].Message, "has tag Float") {
		t.Fatalf("tag diagnostics = %#v", tags)
	}
}
