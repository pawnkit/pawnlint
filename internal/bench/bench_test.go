package bench_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

const benchSrc = `main()
{
    if (cond);
    {
        DoWork();
    }
    while (x);
    {
    }
    playerid + 1;
    if (!flags & FLAG_ADMIN)
    {
    }
    return Function1(), Function2(), 3;
    if (0 < value < 10)
    {
    }
    if (playerid = GetPlayer())
    {
    }
}
`

func benchmarkSetup(b *testing.B) (*lint.Engine, map[string]diagnostic.Severity, map[string]struct{}) {
	reg := rules.Default()
	engine := lint.NewEngine(reg)
	known := map[string]struct{}{}
	ruleSet := map[string]diagnostic.Severity{}
	for _, id := range reg.IDs() {
		known[id] = struct{}{}
		m, _ := reg.Lookup(id)
		ruleSet[id] = m.DefaultSeverity
	}
	return engine, ruleSet, known
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = parser.Parse([]byte(benchSrc))
	}
}

func BenchmarkCompactParse(b *testing.B) {
	source := []byte(benchSrc)
	b.ReportAllocs()
	for range b.N {
		_ = parser.ParseWithProfile(source, parser.ProfileAnalysis)
	}
}

func BenchmarkLosslessParse(b *testing.B) {
	source := []byte(benchSrc)
	b.ReportAllocs()
	for range b.N {
		_ = parser.ParseWithProfile(source, parser.ProfileLossless)
	}
}

func BenchmarkParseLarge(b *testing.B) {
	source := make([]byte, 0, len(benchSrc)*200)
	for range 200 {
		source = append(source, benchSrc...)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = parser.Parse(source)
	}
}

func BenchmarkCompactParseLarge(b *testing.B) {
	source := make([]byte, 0, len(benchSrc)*200)
	for range 200 {
		source = append(source, benchSrc...)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = parser.ParseWithProfile(source, parser.ProfileAnalysis)
	}
}

func BenchmarkLosslessParseLarge(b *testing.B) {
	source := make([]byte, 0, len(benchSrc)*200)
	for range 200 {
		source = append(source, benchSrc...)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = parser.ParseWithProfile(source, parser.ProfileLossless)
	}
}

func BenchmarkWalkIndex(b *testing.B) {
	pf := parser.Parse([]byte(benchSrc))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		walk.New("x.pwn", pf)
	}
}

func BenchmarkCompactWalkIndex(b *testing.B) {
	file := parser.ParseCompact([]byte(benchSrc), parser.ParseOptions{})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		walk.NewCompact("x.pwn", file)
	}
}

func BenchmarkSemanticModel(b *testing.B) {
	file := parser.Parse([]byte(benchSrc))
	tree := walk.New("x.pwn", file)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		semantic.Build(file, tree)
	}
}

func BenchmarkCompactSemanticModel(b *testing.B) {
	file := parser.ParseForLinter([]byte(benchSrc))
	tree := walk.NewCompact("x.pwn", file)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		semantic.BuildCompact(file, tree)
	}
}

func BenchmarkSemanticModelLarge(b *testing.B) {
	source := make([]byte, 0, len(benchSrc)*200)
	for range 200 {
		source = append(source, benchSrc...)
	}
	file := parser.Parse(source)
	tree := walk.New("x.pwn", file)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		semantic.Build(file, tree)
	}
}

func BenchmarkCompactSemanticModelLarge(b *testing.B) {
	source := make([]byte, 0, len(benchSrc)*200)
	for range 200 {
		source = append(source, benchSrc...)
	}
	file := parser.ParseForLinter(source)
	tree := walk.NewCompact("x.pwn", file)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		semantic.BuildCompact(file, tree)
	}
}

func BenchmarkEngineLintFile(b *testing.B) {
	engine, ruleSet, known := benchmarkSetup(b)
	src := []byte(benchSrc)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = engine.LintFile("x.pwn", src, lint.SyntaxAnalysis, ruleSet, known, nil)
	}
}

func BenchmarkEngineLintFileLarge(b *testing.B) {
	engine, ruleSet, known := benchmarkSetup(b)
	large := make([]byte, 0, len(benchSrc)*200)
	for i := 0; i < 200; i++ {
		large = append(large, []byte(benchSrc)...)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = engine.LintFile("x.pwn", large, lint.SyntaxAnalysis, ruleSet, known, nil)
	}
}
