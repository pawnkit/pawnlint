package performance

import (
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestLoopInvariantCallGroupsAreOrdered(t *testing.T) {
	file := parser.Parse([]byte("main() { for (new i = 0; i < 2; i++) { Work(); } while (ready) { Work(); } }\n"))
	ctx := &lint.Context{Walk: walk.New("loops.pwn", file)}
	groups := loopInvariantCallGroups(ctx)
	if len(groups) != 2 || len(groups[0].calls) != 1 || len(groups[1].calls) != 1 {
		t.Fatalf("loop call groups = %#v", groups)
	}
	if groups[0].loop.Start >= groups[1].loop.Start {
		t.Fatal("loop call groups are not ordered by source position")
	}
}
