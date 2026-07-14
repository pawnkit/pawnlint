package bench_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
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

func BenchmarkWalkIndex(b *testing.B) {
	pf := parser.Parse([]byte(benchSrc))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		walk.New("x.pwn", pf)
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
