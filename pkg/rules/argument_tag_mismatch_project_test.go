package rules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestArgumentTagMismatchAcrossInclude(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "units.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	include := "stock SetDistance(Float:value) { return floatround(value); }\n"
	main := "#include \"units.inc\"\nmain() { SetDistance(1); }\n"
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
	diagnostics := lintProjectRule(t, model, nil, mainPath, "argument-tag-mismatch")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "expects tag Float, but has no tag") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestArgumentTagMismatchExpandsTagAliases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	include := `#define ConstString String@Const
#define StringTags String
#define ConstStringTags ConstString,StringTags
#define ConstStringTag {ConstStringTags}
native str_get(ConstStringTag:value, output[]);
`
	if err := os.WriteFile(filepath.Join(dir, "types.inc"), []byte(include), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `#include "types.inc"
new ConstString:commandName;
new command[65];
main() { str_get(commandName, command); }
`
	model, err := project.Build([]project.Source{{Path: path, Content: []byte(source)}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := lintProjectRule(t, model, nil, path, "argument-tag-mismatch"); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}
