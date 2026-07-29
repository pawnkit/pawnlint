package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	parser "github.com/pawnkit/pawn-parser"
)

func BenchmarkBuildContextualIncludes(b *testing.B) {
	dir, entry, source := contextualIncludeBenchmark(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Build([]Source{{Path: entry, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildContextualIncludesCompact(b *testing.B) {
	dir, entry, source := contextualIncludeBenchmark(b)
	features := AllFeatures()
	cache := NewParseCache()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Build([]Source{{Path: entry, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true, Features: &features, ParseCache: cache}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildSharedCompactIncludeContexts(b *testing.B) {
	dir, sources := sharedCompactIncludeContexts(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := Build(sources, Options{WorkingDir: dir, DefinesComplete: true, ReleaseIncludes: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSharedCompactIncludeContextPerformanceBudget(t *testing.T) {
	if os.Getenv("PAWNKIT_PERFORMANCE_BUDGET") == "" {
		t.Skip()
	}
	dir, sources := sharedCompactIncludeContexts(t)
	var buildErr error
	started := time.Now()
	allocations := testing.AllocsPerRun(1, func() {
		_, buildErr = Build(sources, Options{WorkingDir: dir, DefinesComplete: true, ReleaseIncludes: true})
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	if allocations > 60_000 {
		t.Fatalf("allocations = %.0f, budget = 60000", allocations)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("elapsed = %s, budget = 500ms", elapsed)
	}
}

func sharedCompactIncludeContexts(tb testing.TB) (string, []Source) {
	tb.Helper()
	dir := tb.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	var include strings.Builder
	for index := range 100 {
		fmt.Fprintf(&include, "#if defined CONTEXT_%d\nstock Function%d() {}\n#endif\n", index, index)
	}
	if err := os.WriteFile(includePath, []byte(include.String()), 0o644); err != nil {
		tb.Fatal(err)
	}
	sources := make([]Source, 100)
	for index := range sources {
		path := filepath.Join(dir, fmt.Sprintf("root_%d.pwn", index))
		content := fmt.Appendf(nil, "#define CONTEXT_%d\n#include \"shared.inc\"\n", index)
		sources[index] = Source{Path: path, Content: content}
	}
	return dir, sources
}

func BenchmarkFunctionEffects(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "main.pwn")
	var source strings.Builder
	for index := 0; index < 2_000; index++ {
		fmt.Fprintf(&source, "Function%d(&value, other) { return value + other; }\n", index)
	}
	content := []byte(source.String())
	b.ReportAllocs()
	b.StopTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		model, err := Build([]Source{{Path: path, Content: content}}, Options{WorkingDir: dir, DefinesComplete: true})
		if err != nil {
			b.Fatal(err)
		}
		functions := model.Declarations["Function1999"]
		if len(functions) != 1 {
			b.Fatalf("Function1999 declarations = %d", len(functions))
		}
		b.StartTimer()
		if _, ok := model.FunctionEffects(functions[0]); !ok {
			b.Fatal("function effects are unavailable")
		}
		b.StopTimer()
	}
}

func BenchmarkEvalUnknown(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "main.pwn")
	model, err := Build([]Source{{Path: path, Content: []byte("main() { Missing; }\n")}}, Options{WorkingDir: dir})
	if err != nil {
		b.Fatal(err)
	}
	file := model.File(path)
	identifiers := file.Walk.OfKind(parser.KindIdentifier)
	if len(identifiers) == 0 {
		b.Fatal("missing identifier")
	}
	node := identifiers[len(identifiers)-1]
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		model.Eval(file, node)
	}
}

func BenchmarkFunctionEffectsCallChain(b *testing.B) {
	benchmarkFunctionEffectsCallChain(b, false)
}

func BenchmarkFunctionEffectsReverseCallChain(b *testing.B) {
	benchmarkFunctionEffectsCallChain(b, true)
}

func benchmarkFunctionEffectsCallChain(b *testing.B, reverse bool) {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "main.pwn")
	var source strings.Builder
	source.WriteString("new shared;\n")
	if reverse {
		source.WriteString("Function500() { return shared; }\n")
		for index := 499; index >= 0; index-- {
			fmt.Fprintf(&source, "Function%d() { return Function%d(); }\n", index, index+1)
		}
	} else {
		for index := 0; index < 500; index++ {
			fmt.Fprintf(&source, "Function%d() { return Function%d(); }\n", index, index+1)
		}
		source.WriteString("Function500() { return shared; }\n")
	}
	content := []byte(source.String())
	b.ReportAllocs()
	b.StopTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		model, err := Build([]Source{{Path: path, Content: content}}, Options{WorkingDir: dir, DefinesComplete: true})
		if err != nil {
			b.Fatal(err)
		}
		functions := model.Declarations["Function0"]
		if len(functions) != 1 {
			b.Fatalf("Function0 declarations = %d", len(functions))
		}
		b.StartTimer()
		effects, ok := model.FunctionEffects(functions[0])
		b.StopTimer()
		if !ok || len(effects.ReadsGlobals) != 1 {
			b.Fatalf("Function0 effects = %#v, %v", effects, ok)
		}
	}
}

func contextualIncludeBenchmark(b *testing.B) (string, string, []byte) {
	b.Helper()
	dir := b.TempDir()
	var root strings.Builder
	for i := 0; i < 25; i++ {
		name := fmt.Sprintf("include_%02d", i)
		path := filepath.Join(dir, name+".inc")
		source := fmt.Sprintf("#define CONTEXT_%02d\nstock Function%02d() {}\n", i, i)
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			b.Fatal(err)
		}
		fmt.Fprintf(&root, "#include \"%s.inc\"\n", name)
	}
	root.WriteString("main() {}\n")
	entry := filepath.Join(dir, "main.pwn")
	source := []byte(root.String())
	return dir, entry, source
}
