package maintainability

import (
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestDialogHandlerHasExternalSignature(t *testing.T) {
	t.Parallel()

	source := []byte("Dialog:Menu(playerid, response, listitem, inputtext[]) {}\n")
	parsed := parser.Parse(source)
	tree := walk.New("test.pwn", parsed)
	functions := tree.OfKind(parser.KindFunctionDefinition)
	if len(functions) != 1 {
		t.Fatalf("functions = %d, want 1", len(functions))
	}
	if !hasExternalSignature(&lint.Context{Walk: tree}, functions[0]) {
		t.Fatal("dialog handler was treated as an internal function")
	}
}
