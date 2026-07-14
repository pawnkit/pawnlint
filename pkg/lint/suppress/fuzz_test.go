package suppress_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
)

func FuzzSuppression(f *testing.F) {
	f.Add([]byte("// pawnlint-disable-next-line foo\nx;\n"))
	f.Add([]byte("/* pawnlint-disable all\nx;\npawnlint-enable all */\n"))
	f.Add([]byte("// pawnlint-disable-next-line\n"))
	f.Add([]byte("// pawnlint-disable"))
	f.Fuzz(func(t *testing.T, src []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		pf := parser.Parse(src)
		if pf == nil {
			return
		}
		ds := suppress.FromFile("fuzz.pwn", src, pf)
		m := suppress.NewMatcher(ds)
		used := make([]bool, len(ds))
		for _, d := range ds {
			m.IsSuppressed(used, "x", d.Line+1)
		}
	})
}
