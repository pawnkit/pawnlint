package controlflow_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func BenchmarkAliasFlow(b *testing.B) {
	var source strings.Builder
	source.WriteString("Read(value) {} main(bool:check) { new value0 = 0;")
	for index := 1; index <= 100; index++ {
		fmt.Fprintf(&source, "new value%d = value%d;", index, index-1)
		if index%10 == 0 {
			fmt.Fprintf(&source, "if (check) value%d = value0; else value%d = value0; Read(value%d);", index, index, index)
		}
	}
	source.WriteString("}")
	file := parser.Parse([]byte(source.String()))
	tree := walk.New("benchmark.pwn", file)
	semantics := semantic.Build(file, tree)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = controlflow.Build(tree, semantics)
	}
}
