package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/output"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func diags() []diagnostic.Diagnostic {
	lt := source.NewLineTable([]byte("main()\n{\n playerid + 1;\n}\n"))
	r := lt.Range(10, 22)
	return []diagnostic.Diagnostic{{
		RuleID:   "discarded-expression",
		Severity: diagnostic.SeverityWarning,
		Category: diagnostic.CategorySuspicious,
		Message:  "expression has no effect",
		Filename: "x.pwn",
		Range:    r,
	}}
}

func TestText(t *testing.T) {
	var b bytes.Buffer
	err := output.Write(&b, output.FormatText, diags(), output.SourceSet{"x.pwn": []byte("main()\n{\n playerid + 1;\n}\n")}, false)
	if err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if !strings.Contains(s, "x.pwn:3:2:") {
		t.Errorf("missing header: %q", s)
	}
	if !strings.Contains(s, "expression has no effect") {
		t.Errorf("missing message")
	}
	if !strings.Contains(s, "^") {
		t.Errorf("missing caret")
	}
}

func TestTextClipsMultilineCaret(t *testing.T) {
	src := []byte("abc\ndef\n")
	lt := source.NewLineTable(src)
	d := diags()[0]
	d.Filename = "x.pwn"
	d.Range = lt.Range(1, 7)
	var b bytes.Buffer
	if err := output.Write(&b, output.FormatText, []diagnostic.Diagnostic{d}, output.SourceSet{"x.pwn": src}, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "^^^^^^") || !strings.Contains(b.String(), "^^") {
		t.Fatalf("caret was not clipped to the first line: %q", b.String())
	}
}

func TestTextCaretMirrorsTabIndent(t *testing.T) {
	src := []byte("main()\n{\n\t\tif(playerid + 1)\n\t\t{\n\t\t}\n}\n")
	lt := source.NewLineTable(src)
	d := diags()[0]
	d.Filename = "x.pwn"
	d.Range = lt.Range(12, 25) // "playerid + 1"
	var b bytes.Buffer
	if err := output.Write(&b, output.FormatText, []diagnostic.Diagnostic{d}, output.SourceSet{"x.pwn": src}, false); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(b.String(), "\n")
	var srcLine, caretLine string
	for i, l := range lines {
		if strings.HasPrefix(l, "3 | ") {
			srcLine = l
			caretLine = lines[i+1]
		}
	}
	if srcLine == "" {
		t.Fatalf("missing source line: %q", b.String())
	}
	prefixLen := len("3 | ")
	wantIndent := srcLine[prefixLen : prefixLen+2]
	if wantIndent != "\t\t" {
		t.Fatalf("test setup: expected tab indent, got %q", wantIndent)
	}
	if !strings.HasPrefix(caretLine, strings.Repeat(" ", prefixLen)+"\t\t") {
		t.Fatalf("caret line does not mirror source tabs: %q", caretLine)
	}
}

func TestCompact(t *testing.T) {
	var b bytes.Buffer
	if err := output.Write(&b, output.FormatCompact, diags(), nil, false); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(b.String(), "x.pwn:") || !strings.Contains(b.String(), "discarded-expression") {
		t.Errorf("compact: %q", b.String())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	var b bytes.Buffer
	if err := output.Write(&b, output.FormatJSON, diags(), nil, false); err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(b.Bytes(), &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 || arr[0]["ruleId"] != "discarded-expression" {
		t.Errorf("json: %v", arr)
	}
}

func TestJSONL(t *testing.T) {
	var b bytes.Buffer
	if err := output.Write(&b, output.FormatJSONL, diags(), nil, false); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("jsonl got %d lines", len(lines))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatal(err)
	}
	if m["ruleId"] != "discarded-expression" {
		t.Errorf("jsonl: %v", m)
	}
}

func TestSARIFAndGitHub(t *testing.T) {
	var s, g bytes.Buffer
	if err := output.Write(&s, output.FormatSARIF, diags(), nil, false); err != nil {
		t.Fatal(err)
	}
	var log struct {
		Runs []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(s.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if len(log.Runs) != 1 || len(log.Runs[0].Results) != 1 || log.Runs[0].Results[0].RuleID != "discarded-expression" {
		t.Errorf("sarif dropped diagnostics: %s", s.String())
	}
	if err := output.Write(&g, output.FormatGitHub, diags(), nil, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(g.String(), "::warning") {
		t.Errorf("github: %q", g.String())
	}
}

func TestGitHubEscapesCommands(t *testing.T) {
	d := diags()[0]
	d.Filename = "a,b:pwn"
	d.Message = "first%line\n::error::injected"
	var b bytes.Buffer
	if err := output.Write(&b, output.FormatGitHub, []diagnostic.Diagnostic{d}, nil, false); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("annotation contains an unescaped newline: %q", got)
	}
	if !strings.Contains(got, "file=a%2Cb%3Apwn") || !strings.Contains(got, "first%25line%0A::error::injected") {
		t.Fatalf("annotation was not escaped: %q", got)
	}
}

func TestAllowedFormats(t *testing.T) {
	if !output.AllowedFormat("json") {
		t.Error("json allowed")
	}
	if output.AllowedFormat("xml") {
		t.Error("xml not allowed")
	}
}
