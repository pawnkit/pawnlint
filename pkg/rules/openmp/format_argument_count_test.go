package openmp

import (
	"reflect"
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestFormatArgumentCount(t *testing.T) {
	tests := []struct {
		format string
		count  int
		ok     bool
	}{
		{format: "plain", count: 0, ok: true},
		{format: "%d %s %.2f", count: 3, ok: true},
		{format: "%05d %10s %% %q", count: 3, ok: true},
		{format: "%%%d", count: 1, ok: true},
		{format: "%e", ok: false},
		{format: "%", ok: false},
	}
	for _, test := range tests {
		count, ok := formatArgumentCount(test.format)
		if count != test.count || ok != test.ok {
			t.Errorf("formatArgumentCount(%q) = %d, %t", test.format, count, ok)
		}
	}
}

func TestFormatSpecifiers(t *testing.T) {
	specifiers, ok := formatSpecifiers("%05d %.2f %% %q")
	want := []byte{'d', 'f', 'q'}
	if !ok || !reflect.DeepEqual(specifiers, want) {
		t.Fatalf("format specifiers = %q, %t, want %q", specifiers, ok, want)
	}
}

func TestLiteralContinues(t *testing.T) {
	for _, test := range []struct {
		source string
		want   bool
	}{
		{source: `main() { print("value", suffix); }`},
		{source: `main() { print("value" suffix); }`, want: true},
		{source: `main() { print("value" "next"); }`, want: true},
	} {
		parsed := parser.Parse([]byte(test.source))
		var literal token.Token
		for _, current := range parsed.Tokens {
			if current.Kind == token.StringLiteral {
				literal = current
				break
			}
		}
		ctx := &lint.Context{File: &lint.File{Parsed: parsed}}
		if got := literalContinues(ctx, &parser.Node{Tok: literal}); got != test.want {
			t.Errorf("literalContinues(%q) = %t, want %t", test.source, got, test.want)
		}
	}
}
