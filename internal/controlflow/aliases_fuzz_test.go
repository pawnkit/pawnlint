package controlflow_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func FuzzAliases(f *testing.F) {
	f.Add("main() { new a = 1; new b = a; b = 2; }\n")
	f.Add("Change(&value) {} main(bool:check) { new a; new b; if (check) b = a; Change(b); }\n")
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 32*1024 {
			t.Skip()
		}
		file := parser.Parse([]byte(input))
		tree := walk.New("fuzz.pwn", file)
		semantics := semantic.Build(file, tree)
		model := controlflow.Build(tree, semantics)
		for _, identifier := range tree.OfKind(parser.KindIdentifier) {
			if symbol := semantics.Resolve(identifier); symbol != nil {
				_ = model.Aliases(identifier, symbol)
			}
		}
	})
}
