package semantic_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func FuzzBuild(f *testing.F) {
	f.Add([]byte("main() { new value; value = 1; }"))
	f.Add([]byte("main( { new value value = ;"))
	f.Add([]byte("#if defined X\nnew value;\n#endif"))
	f.Fuzz(func(t *testing.T, src []byte) {
		file := parser.Parse(src)
		if file == nil {
			return
		}
		model := semantic.Build(file, walk.New("fuzz.pwn", file))
		for _, symbol := range model.Symbols {
			_ = model.References(symbol)
		}
		for _, node := range model.Walk.All() {
			_, _ = model.Eval(node)
			_ = model.ExpressionTags(node)
			_ = model.Boolean(node)
		}
	})
}
