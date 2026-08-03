package performance

import (
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type loopCallGroup struct {
	loop  *parser.Node
	calls []*parser.Node
}

func loopInvariantCallGroups(ctx *lint.Context) []loopCallGroup {
	callsByLoop := ctx.LoopCalls()
	loops := make([]*parser.Node, 0, len(callsByLoop))
	for _, kind := range []parser.Kind{parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement} {
		for _, loop := range ctx.Walk.OfKind(kind) {
			if _, ok := callsByLoop[loop]; ok {
				loops = append(loops, loop)
			}
		}
	}
	sort.Slice(loops, func(left, right int) bool { return loops[left].Start < loops[right].Start })
	groups := make([]loopCallGroup, 0, len(loops))
	for _, loop := range loops {
		groups = append(groups, loopCallGroup{loop: loop, calls: callsByLoop[loop]})
	}
	return groups
}
