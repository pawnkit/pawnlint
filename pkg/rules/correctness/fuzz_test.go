package correctness_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/ruletest"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func FuzzMalformed(f *testing.F) {
	f.Add([]byte("main()\n{\n}\n"))
	f.Add([]byte("if (a);;{{"))
	f.Add([]byte("#if defined X\nx;\n#endif\n"))
	f.Add([]byte("main(){ return a, b, c }"))
	f.Add([]byte("new a[][] = {}; main() { a[0] = ! /* */ & ; }"))
	f.Fuzz(func(t *testing.T, src []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on malformed source: %v", r)
			}
		}()
		reg := ruletest.SilentRegistrar()
		known := map[string]struct{}{}
		for _, id := range reg.IDs() {
			known[id] = struct{}{}
		}
		ruleSet := map[string]diagnostic.Severity{}
		for _, id := range reg.IDs() {
			m, _ := reg.Lookup(id)
			ruleSet[id] = m.DefaultSeverity
		}
		engine := lint.NewEngine(reg)
		_ = engine.LintFile("fuzz.pwn", src, lint.ControlFlowAnalysis, ruleSet, known, nil)
	})
}
