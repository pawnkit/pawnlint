package walk

import (
	"testing"

	parser "github.com/pawnkit/pawn-parser"
)

func TestReferencesByAmpersand(t *testing.T) {
	parsed := parser.Parse([]byte("forward Copy(const source[], &target, size);\n"))
	tree := New("test.pwn", parsed)
	parameters := tree.OfKind(parser.KindParameter)
	if len(parameters) != 3 {
		t.Fatalf("parameters = %d", len(parameters))
	}
	if ReferencesByAmpersand(parsed.Tokens, parameters[0]) {
		t.Fatal("array parameter reported as reference")
	}
	if !ReferencesByAmpersand(parsed.Tokens, parameters[1]) {
		t.Fatal("reference parameter was not detected")
	}
	if ReferencesByAmpersand(parsed.Tokens, parameters[2]) {
		t.Fatal("value parameter reported as reference")
	}
}
