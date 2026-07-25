package project_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pawnkit/pawn-parser/lexer"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestParseCacheReusesUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	cache := project.NewParseCache()
	source := project.Source{Path: path, Content: []byte("main() {}\n")}
	parseEvents := 0
	options := project.Options{WorkingDir: dir, ParseCache: cache, ObserveTiming: func(event project.TimingEvent) {
		if event.Stage == project.TimingParse {
			parseEvents++
		}
	}}
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 1 {
		t.Fatalf("first parse events = %d", parseEvents)
	}
	parseEvents = 0
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 0 {
		t.Fatalf("cached parse events = %d", parseEvents)
	}
	source.Content = append(source.Content, '\n')
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 1 {
		t.Fatalf("changed parse events = %d", parseEvents)
	}
}

func TestParseCacheReusesAnalysisForUnchangedIncludes(t *testing.T) {
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

	cache := project.NewParseCache()
	semanticEvents := 0
	options := project.Options{WorkingDir: dir, IncludePaths: []string{"include"}, ParseCache: cache, ObserveTiming: func(event project.TimingEvent) {
		if event.Stage == project.TimingSemantic {
			semanticEvents++
		}
	}}

	if _, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, options); err != nil {
		t.Fatal(err)
	}
	if semanticEvents != 2 {
		t.Fatalf("first build semantic events = %d, want 2 (root + include)", semanticEvents)
	}

	semanticEvents = 0
	edited := []byte("#include <shared>\nmain() { print(\"x\"); }\n")
	model, err := project.Build([]project.Source{{Path: rootPath, Content: edited}}, options)
	if err != nil {
		t.Fatal(err)
	}
	if semanticEvents != 1 {
		t.Fatalf("second build semantic events = %d, want 1 (root only)", semanticEvents)
	}
	if len(model.Declarations["Shared"]) != 1 {
		t.Fatalf("Shared declaration missing after cached include reuse: %#v", model.Declarations["Shared"])
	}
}

func TestParseCacheInvalidatesAnalysisOnDefineChange(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includePath := filepath.Join(includeDir, "shared.inc")
	body := "#if defined FEATURE_X\nstock FeatureFunc() {}\n#endif\n"
	if err := os.WriteFile(includePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")

	cache := project.NewParseCache()
	options := project.Options{WorkingDir: dir, IncludePaths: []string{"include"}, ParseCache: cache}

	without, err := project.Build([]project.Source{{Path: rootPath, Content: []byte("#include <shared>\nmain() {}\n")}}, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(without.Declarations["FeatureFunc"]) != 0 {
		t.Fatal("FeatureFunc should not be declared without FEATURE_X defined")
	}

	with, err := project.Build([]project.Source{{Path: rootPath, Content: []byte("#define FEATURE_X\n#include <shared>\nmain() {}\n")}}, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(with.Declarations["FeatureFunc"]) != 1 {
		t.Fatalf("FeatureFunc should be declared once FEATURE_X is defined, got %#v", with.Declarations["FeatureFunc"])
	}
}

func TestParseCacheReusesFinalWalkAfterMidFileRedefinition(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includePath := filepath.Join(includeDir, "redefine.inc")
	if err := os.WriteFile(includePath, []byte("#define VALUE 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#define VALUE 1\nnew before = VALUE;\n#include <redefine>\nnew after = VALUE;\nmain() {}\n")

	cache := project.NewParseCache()
	options := project.Options{WorkingDir: dir, IncludePaths: []string{"include"}, ParseCache: cache}

	first, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, options)
	if err != nil {
		t.Fatal(err)
	}
	firstRoot := first.File(rootPath)
	secondRoot := second.File(rootPath)
	if len(firstRoot.Semantic.Symbols) != len(secondRoot.Semantic.Symbols) {
		t.Fatalf("symbol count changed across cached builds: %d vs %d", len(firstRoot.Semantic.Symbols), len(secondRoot.Semantic.Symbols))
	}
}

func TestParseCacheSeparatesTriviaProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	cache := project.NewParseCache()
	source := project.Source{Path: path, Content: []byte("// documentation\nmain() {}\n")}
	parseEvents := 0
	features := project.Features(0)
	options := project.Options{WorkingDir: dir, ParseCache: cache, Features: &features, ObserveTiming: func(event project.TimingEvent) {
		if event.Stage == project.TimingParse {
			parseEvents++
		}
	}}
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	features = project.NewFeatures(project.FeatureTrivia)
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 2 {
		t.Fatalf("parse events = %d, want 2", parseEvents)
	}
}

func TestRootTokensProduceIdenticalParseToNormalPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	content := []byte("#define MAX 10\nnew values[MAX];\nmain() { new x = values[0]; }\n")
	source := project.Source{Path: path, Content: content}

	without, err := project.Build([]project.Source{source}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	with, err := project.Build([]project.Source{source}, project.Options{
		WorkingDir: dir, RootTokens: lexer.Tokenize(content),
	})
	if err != nil {
		t.Fatal(err)
	}

	got := with.File(path).Parsed
	want := without.File(path).Parsed
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parse with RootTokens diverged from the normal path:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestRootTokensIgnoredForIncludes(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includeContent := []byte("stock Shared() {}\n")
	if err := os.WriteFile(filepath.Join(includeDir, "shared.inc"), includeContent, 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	rootContent := []byte("#include <shared>\nmain() { Shared(); }\n")

	model, err := project.Build([]project.Source{{Path: rootPath, Content: rootContent}}, project.Options{
		WorkingDir: dir, IncludePaths: []string{"include"}, RootTokens: lexer.Tokenize(rootContent),
	})
	if err != nil {
		t.Fatal(err)
	}
	root := model.File(rootPath)
	if root == nil || len(root.Includes) != 1 || root.Includes[0].Resolved == nil {
		t.Fatalf("include was not resolved: %#v", root)
	}
	includeFile := root.Includes[0].Resolved
	want := project.Options{WorkingDir: dir, IncludePaths: []string{"include"}}
	reference, err := project.Build([]project.Source{{Path: rootPath, Content: rootContent}}, want)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(includeFile.Parsed, reference.File(rootPath).Includes[0].Resolved.Parsed) {
		t.Fatal("RootTokens incorrectly affected the include's own parse")
	}
}
