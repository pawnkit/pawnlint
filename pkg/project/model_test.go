package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
)

func TestBuildResolvesIncludesAndIndexesDeclarations(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includePath := filepath.Join(includeDir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("new shared_value;\nstock Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include <shared>\nmain() {}\n")

	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir, IncludePaths: []string{"include"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(model.Files))
	}
	root := model.File(rootPath)
	if root == nil || len(root.Includes) != 1 || root.Includes[0].Resolved == nil {
		t.Fatalf("include was not resolved: %#v", root)
	}
	if root.Includes[0].Resolved.Provided {
		t.Fatal("loaded include marked as provided")
	}
	if len(model.Units) != 1 || len(model.Units[0].Files) != 2 {
		t.Fatalf("units = %#v", model.Units)
	}
	if !hasDeclaration(model, "Shared", semantic.SymbolFunction) {
		t.Fatal("Shared function was not indexed")
	}
	if !hasDeclaration(model, "shared_value", semantic.SymbolGlobal) {
		t.Fatal("shared_value was not indexed")
	}
}

func TestBuildResolvesIncludeWhenExtensionlessPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.MkdirAll(filepath.Join(includeDir, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	includePath := filepath.Join(includeDir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("stock Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include <shared>\nmain() {}\n")

	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir, IncludePaths: []string{"include"}})
	if err != nil {
		t.Fatal(err)
	}
	root := model.File(rootPath)
	if root == nil || len(root.Includes) != 1 || root.Includes[0].Resolved == nil {
		t.Fatalf("include was not resolved: %#v", root)
	}
	if root.Includes[0].Resolved.canonical != includePath {
		t.Fatalf("resolved = %q, want %q", root.Includes[0].Resolved.canonical, includePath)
	}
}

func TestDuplicateFunctionsStayWithinTranslationUnit(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "one.pwn")
	two := filepath.Join(dir, "two.pwn")
	source := []byte("Shared() {}\n")
	model, err := Build([]Source{{Path: one, Content: source}, {Path: two, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestDuplicateFunctionsAcrossIncludes(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	firstPath := filepath.Join(dir, "first.inc")
	secondPath := filepath.Join(dir, "second.inc")
	if err := os.WriteFile(firstPath, []byte("Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte("Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	duplicates := model.DuplicateFunctions()
	if len(duplicates) != 1 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
	if duplicates[0].Name != "Shared" || duplicates[0].Owner.Path != rootPath {
		t.Fatalf("duplicate = %#v", duplicates[0])
	}
}

func TestDuplicateHookFunctionsAreIgnored(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	firstPath := filepath.Join(dir, "first.inc")
	secondPath := filepath.Join(dir, "second.inc")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("hook OnPlayerConnect(playerid) {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestDuplicatePublicFunctionsAcrossIncludesAreIgnored(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	firstPath := filepath.Join(dir, "first.inc")
	secondPath := filepath.Join(dir, "second.inc")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("public OnPlayerConnect(playerid) {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestDuplicateInlineFunctionsAreIgnored(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("First() { inline Response() {} } Second() { inline Response() {} }\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestDuplicateMacroQualifiedFunctionsAreIgnored(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#define ACMD:%0(%1) forward acmd_%0(%1); public acmd_%0(%1)\nACMD:vehicle(playerid) {}\nvehicle() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestDuplicateTaggedFunctionsAreReported(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("Float:value() {} Float:value() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	duplicates := model.DuplicateFunctions()
	if len(duplicates) != 1 || duplicates[0].Name != "value" {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestGenericFunctionVariantsAreIgnored(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("FormatSpecifier<'T'>(output[]) {} FormatSpecifier<'M'>(output[]) {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestDuplicateGlobalsAcrossIncludes(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	includePath := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("new shared_value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := []byte("#include \"shared.inc\"\nnew shared_value;\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	duplicates := model.DuplicateGlobals()
	if len(duplicates) != 1 || duplicates[0].Name != "shared_value" || duplicates[0].Owner.Path != rootPath {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestMacroFunctionDoesNotCreateDuplicateGlobal(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	firstPath := filepath.Join(dir, "first.inc")
	secondPath := filepath.Join(dir, "second.inc")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("timer Delayed[1000](playerid) {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateGlobals(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestStaticGlobalsAcrossIncludesAreIgnored(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	firstPath := filepath.Join(dir, "first.inc")
	secondPath := filepath.Join(dir, "second.inc")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("static shared_value;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateGlobals(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestNumericSeparatorsDoNotCreateDuplicateGlobals(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("const First = 1_000;\nconst Second = 2_000;\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateGlobals(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestRepeatedIncludeDoesNotDuplicateFile(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\n#include \"shared.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Files) != 2 || len(model.Units[0].Files) != 2 {
		t.Fatalf("files = %d, unit files = %d", len(model.Files), len(model.Units[0].Files))
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("duplicates = %#v", duplicates)
	}
}

func TestBuildResolvesCrossFileReferences(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	includeSource := []byte("new shared_value;\nShared() {}\n")
	if err := os.WriteFile(includePath, includeSource, 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nmain() { Shared(); shared_value = 1; }\n")
	model, err := Build([]Source{{Path: rootPath, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	function := onlyDeclaration(t, model, "Shared", semantic.SymbolFunction)
	global := onlyDeclaration(t, model, "shared_value", semantic.SymbolGlobal)
	if references := model.References(function); len(references) != 1 || references[0].Kind != semantic.ReferenceCall {
		t.Fatalf("function references = %#v", references)
	}
	if references := model.References(global); len(references) != 1 || references[0].Kind != semantic.ReferenceWrite {
		t.Fatalf("global references = %#v", references)
	}
	root := model.File(rootPath)
	resolved := 0
	for _, node := range root.Walk.OfKind(parser.KindIdentifier) {
		if _, ok := model.Resolve(root, node); ok {
			resolved++
		}
	}
	if resolved != 2 {
		t.Fatalf("resolved cross-file nodes = %d, want 2", resolved)
	}
}

func hasDeclaration(model *Model, name string, kind semantic.SymbolKind) bool {
	for _, declaration := range model.Declarations[name] {
		if declaration.Kind == kind {
			return true
		}
	}
	return false
}

func onlyDeclaration(t *testing.T, model *Model, name string, kind semantic.SymbolKind) Declaration {
	t.Helper()
	var matches []Declaration
	for _, declaration := range model.Declarations[name] {
		if declaration.Kind == kind {
			matches = append(matches, declaration)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("%s declarations = %#v", name, matches)
	}
	return matches[0]
}
