package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawn-parser"
)

func TestCrossFileConstantEvaluation(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.inc")
	valuesPath := filepath.Join(dir, "values.inc")
	rootPath := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(basePath, []byte("const Base = 4;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valuesPath, []byte("#include \"base.inc\"\nenum Value { First = Base, Second, Slots[2], After };\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := []byte("#include \"values.inc\"\nmain() { new value = After + Base; }\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	root := model.File(rootPath)
	declarators := root.Walk.OfKind(parser.KindVariableDeclarator)
	value, ok := model.Eval(root, declarators[0].Field("initializer"))
	if !ok || value != 12 {
		t.Fatalf("cross-file value = %d, %v", value, ok)
	}
}

func TestCrossFileConstantCycleIsUnknown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "constants.inc"), []byte("const A = B;\nconst B = A;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"constants.inc\"\nmain() { new value = A; }\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	root := model.File(rootPath)
	initializer := root.Walk.OfKind(parser.KindVariableDeclarator)[0].Field("initializer")
	if _, ok := model.Eval(root, initializer); ok {
		t.Fatal("cyclic constant resolved")
	}
}

func TestCrossFileExpressionTags(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shared.inc"), []byte("new Float:shared;\nFloat:GetShared() { return shared; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nmain() { new Float:value = shared; new Float:result = GetShared(); }\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	root := model.File(rootPath)
	declarators := root.Walk.OfKind(parser.KindVariableDeclarator)
	for _, declaration := range declarators {
		initializer := declaration.Field("initializer")
		if initializer == nil {
			continue
		}
		tag, ok := model.ExpressionTag(root, initializer)
		if !ok || tag != "Float" {
			t.Fatalf("tag = %q, %v", tag, ok)
		}
	}
}

func TestCrossFileStateFunctionVariants(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "idle.inc"), []byte("Float:Mode()<idle> { return Float:1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "running.inc"), []byte("Float:Mode()<running> { return Float:2; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"idle.inc\"\n#include \"running.inc\"\nmain() { new Float:value = Mode(); }\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	root := model.File(rootPath)
	call := root.Walk.OfKind(parser.KindCallExpression)[0]
	callee := call.Field("function")
	variants := model.FunctionVariants(root, callee)
	if len(variants) != 2 || variants[0].Symbol.States[0] != "idle" || variants[1].Symbol.States[0] != "running" {
		t.Fatalf("variants = %#v", variants)
	}
	if tag, ok := model.ExpressionTag(root, call); !ok || tag != "Float" {
		t.Fatalf("tag = %q, %v", tag, ok)
	}
	if len(model.CallGraph.Calls) != 2 {
		t.Fatalf("calls = %#v", model.CallGraph.Calls)
	}
	for _, variant := range variants {
		if references := model.References(variant); len(references) != 1 {
			t.Fatalf("references to %s = %#v", variant.File.Path, references)
		}
	}
}
