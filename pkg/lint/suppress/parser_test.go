package suppress_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
)

func parseSupp(t *testing.T, path, src string) []suppress.Directive {
	t.Helper()
	f := parser.Parse([]byte(src))
	if f == nil || f.Root == nil {
		t.Fatalf("parse returned nil root for %q", src)
	}
	return suppress.FromFile(path, []byte(src), f)
}

func TestFromFileNextLine(t *testing.T) {
	src := "// pawnlint-disable-next-line float-equality\nif (value == 0.0)\n{\n}\n"
	ds := parseSupp(t, "x.pwn", src)
	if len(ds) != 1 {
		t.Fatalf("got %d directives want 1: %+v", len(ds), ds)
	}
	d := ds[0]
	if d.Kind != suppress.KindDisableNextLine || !d.MatchesRule("float-equality") {
		t.Errorf("wrong directive: %+v", d)
	}
	if d.Reason != "" {
		t.Errorf("unexpected reason %q", d.Reason)
	}
}

func TestFromFileBlockAndReason(t *testing.T) {
	src := "// pawnlint-disable unused-parameter\npublic OnPlayerUpdate(playerid)\n{\n return 1;\n}\n// pawnlint-enable unused-parameter\n"
	ds := parseSupp(t, "x.pwn", src)
	if len(ds) != 2 {
		t.Fatalf("got %d want 2: %+v", len(ds), ds)
	}
	if ds[0].Kind != suppress.KindDisable || ds[1].Kind != suppress.KindEnable {
		t.Errorf("kinds wrong: %+v", ds)
	}
	if !ds[0].MatchesRule("unused-parameter") {
		t.Errorf("matches wrong")
	}
}

func TestReasonTail(t *testing.T) {
	src := "// pawnlint-disable-next-line unused-result -- best-effort notification\nSendNotification(playerid);\n"
	ds := parseSupp(t, "x.pwn", src)
	if len(ds) != 1 {
		t.Fatalf("got %d want 1", len(ds))
	}
	if ds[0].Reason != "best-effort notification" {
		t.Errorf("reason %q", ds[0].Reason)
	}
	if !ds[0].MatchesRule("unused-result") {
		t.Errorf("should match unused-result")
	}
}

func TestAllAndMalformed(t *testing.T) {
	src := "// pawnlint-disable-next-line all\nx;\n// pawnlint-disable-next-line\ny;\n"
	ds := parseSupp(t, "x.pwn", src)
	if len(ds) != 2 {
		t.Fatalf("got %d want 2", len(ds))
	}
	if !ds[0].All {
		t.Errorf("first should be all")
	}
}

func TestBlockSuppression(t *testing.T) {
	src := `// pawnlint-disable all
a;
b;
// pawnlint-enable all
c;
`
	ds := parseSupp(t, "x.pwn", src)
	m := suppress.NewMatcher(ds)
	used := make([]bool, len(ds))
	if !m.IsSuppressed(used, "any-rule", 2) {
		t.Errorf("line2 should be suppressed")
	}
	if !m.IsSuppressed(used, "any-rule", 3) {
		t.Errorf("line3 should be suppressed")
	}
	if m.IsSuppressed(used, "any-rule", 5) {
		t.Errorf("line5 should NOT be suppressed")
	}
	if !used[0] {
		t.Errorf("disable used should be marked")
	}
}

func TestUnmatchedEnable(t *testing.T) {
	src := `// pawnlint-enable foo
x;
`
	ds := parseSupp(t, "x.pwn", src)
	m := suppress.NewMatcher(ds)
	used := make([]bool, len(ds))
	if m.IsSuppressed(used, "foo", 2) {
		t.Errorf("enable with no disable should not suppress")
	}
}

func TestNestedBlockSuppressionTracksOnlyActiveDirectives(t *testing.T) {
	src := `// pawnlint-disable foo
// pawnlint-disable foo
// pawnlint-enable foo
x;
`
	ds := parseSupp(t, "x.pwn", src)
	m := suppress.NewMatcher(ds)
	used := make([]bool, len(ds))
	if !m.IsSuppressed(used, "foo", 4) {
		t.Fatal("outer directive should remain active")
	}
	if !used[0] || used[1] {
		t.Fatalf("used directives = %v", used)
	}
}
