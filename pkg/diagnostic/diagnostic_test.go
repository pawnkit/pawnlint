package diagnostic_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func TestSeverityParseRoundTrip(t *testing.T) {
	for _, s := range []diagnostic.Severity{diagnostic.SeverityOff, diagnostic.SeverityError, diagnostic.SeverityWarning, diagnostic.SeverityInfo, diagnostic.SeverityHint} {
		got, ok := diagnostic.ParseSeverity(s.String())
		if !ok || got != s {
			t.Errorf("roundtrip %v: got %v ok=%v", s, got, ok)
		}
	}
	if _, ok := diagnostic.ParseSeverity("nope"); ok {
		t.Errorf("unknown should fail")
	}
}

func TestDiagnosticCategories(t *testing.T) {
	tests := map[diagnostic.Category]string{
		diagnostic.CategoryCorrectness:     "correctness",
		diagnostic.CategorySuspicious:      "suspicious",
		diagnostic.CategoryPerformance:     "performance",
		diagnostic.CategoryMaintainability: "maintainability",
		diagnostic.CategoryOpenMP:          "openmp",
		diagnostic.CategoryStyle:           "style",
		diagnostic.CategorySecurity:        "security",
		diagnostic.CategoryPortability:     "portability",
		diagnostic.CategoryRestriction:     "restriction",
	}
	for category, want := range tests {
		if got := category.String(); got != want {
			t.Fatalf("category %d = %q, want %q", category, got, want)
		}
	}
}

func TestSortDeterministic(t *testing.T) {
	lt := source.NewLineTable([]byte("ab\ncd\n"))
	ds := []diagnostic.Diagnostic{
		{RuleID: "b", Filename: "x.pwn", Range: lt.Range(0, 1), Message: "m2"},
		{RuleID: "a", Filename: "x.pwn", Range: lt.Range(3, 4), Message: "m1"},
		{RuleID: "a", Filename: "x.pwn", Range: lt.Range(0, 1), Message: "m0"},
		{RuleID: "a", Filename: "y.pwn", Range: lt.Range(0, 1)},
	}
	diagnostic.Sort(ds)
	want := []string{"x.pwn:a:m0", "x.pwn:b:m2", "x.pwn:a:m1", "y.pwn:a:"}
	for i, w := range want {
		got := ds[i].Filename + ":" + ds[i].RuleID + ":" + ds[i].Message
		if got != w {
			t.Errorf("idx %d got %q want %q", i, got, w)
		}
	}
}
