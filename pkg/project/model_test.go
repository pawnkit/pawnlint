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

func TestBuildResolvesDottedIncludeWithIncSuffix(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includePath := filepath.Join(includeDir, "open.mp.inc")
	if err := os.WriteFile(includePath, []byte("stock OpenMP() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include <open.mp>\nmain() {}\n")

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

func TestBuildIndexesDefinedNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte("#define ACTIVE 1\n#if 0\n#define INACTIVE 1\n#endif\nmain() {}\n")
	model, err := Build([]Source{{Path: path, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if !model.DefinesName("ACTIVE") {
		t.Fatal("active define was not indexed")
	}
	if model.DefinesName("INACTIVE") {
		t.Fatal("inactive define was indexed")
	}
}

func TestBuildReportsTimings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	var events []TimingEvent
	_, err := Build([]Source{{Path: path, Content: []byte("main() {}\n")}}, Options{
		WorkingDir: dir,
		ObserveTiming: func(event TimingEvent) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[TimingStage]bool{TimingParse: false, TimingPreprocess: false, TimingSemantic: false}
	for _, event := range events {
		if event.Duration < 0 {
			t.Fatalf("negative duration: %+v", event)
		}
		wanted[event.Stage] = true
	}
	for stage, found := range wanted {
		if !found {
			t.Errorf("missing %s event: %+v", stage, events)
		}
	}
}

func TestBuildCreatesExpandedMacroView(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte("#define DOUBLE(%0) ((%0) + (%0))\nnew value = DOUBLE(3);\n")
	model, err := Build([]Source{{Path: path, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	file := model.File(path)
	if file == nil || !file.ExpansionComplete || file.ExpandedParsed == nil || file.ExpandedWalk == nil || file.ExpandedSemantic == nil {
		t.Fatalf("expanded file = %#v", file)
	}
	declarators := file.ExpandedWalk.OfKind(parser.KindVariableDeclarator)
	if len(declarators) != 1 {
		t.Fatalf("declarators = %d", len(declarators))
	}
	if value, ok := file.ExpandedSemantic.Eval(declarators[0].Field("initializer")); !ok || value != 6 {
		t.Fatalf("value = %d, %v", value, ok)
	}
}

func TestBuildExpandsMacroExportedByInclude(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "macros.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	if err := os.WriteFile(includePath, []byte("#define DOUBLE(%0) ((%0) + (%0))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := []byte("#include \"macros.inc\"\nnew value = DOUBLE(4);\n")
	model, err := Build([]Source{{Path: mainPath, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	file := model.File(mainPath)
	declarators := file.ExpandedWalk.OfKind(parser.KindVariableDeclarator)
	if !file.ExpansionComplete || len(declarators) != 1 {
		t.Fatalf("complete=%v declarators=%d source=%s", file.ExpansionComplete, len(declarators), file.ExpandedSource)
	}
	if value, ok := file.ExpandedSemantic.Eval(declarators[0].Field("initializer")); !ok || value != 8 {
		t.Fatalf("value = %d, %v", value, ok)
	}
	origins := model.ExpansionOrigins(file, declarators[0].Field("initializer"))
	files := make(map[string]bool)
	for _, origin := range origins {
		if origin.File != nil {
			files[origin.File.Path] = true
		}
		if origin.Macro != "DOUBLE" {
			t.Fatalf("origin = %#v", origin)
		}
	}
	if !files[includePath] || !files[mainPath] {
		t.Fatalf("origin files = %#v", files)
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

func TestContextualIncludeDoesNotDuplicatePhysicalDefinitions(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	shared := "#if defined SHARED_INC\n#endinput\n#endif\n#define SHARED_INC\nShared() {}\nnew shared_value;\n"
	if err := os.WriteFile(includePath, []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	leftPath := filepath.Join(dir, "left.inc")
	rightPath := filepath.Join(dir, "right.inc")
	if err := os.WriteFile(leftPath, []byte("#define FIRST\n#include \"shared.inc\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rightPath, []byte("#define SECOND\n#include \"shared.inc\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	root := []byte("#include \"left.inc\"\n#include \"right.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: root}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if duplicates := model.DuplicateFunctions(); len(duplicates) != 0 {
		t.Fatalf("functions = %#v", duplicates)
	}
	if duplicates := model.DuplicateGlobals(); len(duplicates) != 0 {
		t.Fatalf("globals = %#v", duplicates)
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

func TestIncludeReceivesDefinesEstablishedByParent(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "feature.inc")
	includeSource := []byte("#if defined FEATURE\nFeatureEnabled() {}\n#endif\n")
	if err := os.WriteFile(includePath, includeSource, 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	rootSource := []byte("#define FEATURE\n#include \"feature.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: rootSource}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDeclaration(model, "FeatureEnabled", semantic.SymbolFunction) {
		t.Fatal("include did not receive the parent's define environment")
	}
}

func TestIncludeExportsDefinesToLaterParentCode(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "export.inc")
	if err := os.WriteFile(includePath, []byte("#define EXPORTED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	rootSource := []byte("#include \"export.inc\"\n#if defined EXPORTED\nAfterInclude() {}\n#endif\n")
	model, err := Build([]Source{{Path: rootPath, Content: rootSource}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDeclaration(model, "AfterInclude", semantic.SymbolFunction) {
		t.Fatal("include define did not affect later parent code")
	}
}

func TestRepeatedIncludeUsesUpdatedGuardContext(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "guarded.inc")
	includeSource := []byte("#if !defined GUARDED_INCLUDED\n#define GUARDED_INCLUDED\nGuarded() {}\n#endif\n")
	if err := os.WriteFile(includePath, includeSource, 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	rootSource := []byte("#include \"guarded.inc\"\n#include \"guarded.inc\"\nmain() {}\n")
	model, err := Build([]Source{{Path: rootPath, Content: rootSource}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if declarations := model.Declarations["Guarded"]; len(declarations) != 1 {
		t.Fatalf("Guarded declarations = %d, want 1", len(declarations))
	}
	if len(model.Files) != 3 || len(model.Units) != 1 || len(model.Units[0].Files) != 3 {
		t.Fatalf("files = %d, units = %#v", len(model.Files), model.Units)
	}
}

func TestRootsUseIndependentIncludeContexts(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "context.inc")
	includeSource := []byte("#if defined ONE\nFromOne() {}\n#endif\n#if defined TWO\nFromTwo() {}\n#endif\n")
	if err := os.WriteFile(includePath, includeSource, 0o644); err != nil {
		t.Fatal(err)
	}
	onePath := filepath.Join(dir, "one.pwn")
	twoPath := filepath.Join(dir, "two.pwn")
	oneSource := []byte("#define ONE\n#include \"context.inc\"\n")
	twoSource := []byte("#define TWO\n#include \"context.inc\"\n")
	model, err := Build([]Source{{Path: onePath, Content: oneSource}, {Path: twoPath, Content: twoSource}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDeclaration(model, "FromOne", semantic.SymbolFunction) || !hasDeclaration(model, "FromTwo", semantic.SymbolFunction) {
		t.Fatalf("contextual declarations = %#v", model.Declarations)
	}
	if len(model.Files) != 4 || len(model.Units) != 2 {
		t.Fatalf("files = %d, units = %d", len(model.Files), len(model.Units))
	}
	for _, unit := range model.Units {
		want := "FromOne"
		unwanted := "FromTwo"
		if unit.Root.Path == twoPath {
			want, unwanted = unwanted, want
		}
		if !unitHasDeclaration(unit, want) || unitHasDeclaration(unit, unwanted) {
			t.Fatalf("unit %q has incorrect context", unit.Root.Path)
		}
	}
}

func TestContextualDeclarationsKeepIndependentReferences(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	onePath := filepath.Join(dir, "one.pwn")
	twoPath := filepath.Join(dir, "two.pwn")
	oneSource := []byte("#define ONE\n#include \"shared.inc\"\nmain() { Shared(); }\n")
	twoSource := []byte("#define TWO\n#include \"shared.inc\"\nmain() { Shared(); }\n")
	model, err := Build([]Source{{Path: onePath, Content: oneSource}, {Path: twoPath, Content: twoSource}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	declarations := model.Declarations["Shared"]
	if len(declarations) != 2 {
		t.Fatalf("Shared declarations = %d", len(declarations))
	}
	for _, declaration := range declarations {
		if references := model.References(declaration); len(references) != 1 || references[0].Kind != semantic.ReferenceCall {
			t.Fatalf("references = %#v", references)
		}
	}
}

func unitHasDeclaration(unit *Unit, name string) bool {
	for _, file := range unit.Files {
		for _, symbol := range file.Semantic.Symbols {
			if symbol.Name == name {
				return true
			}
		}
	}
	return false
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
