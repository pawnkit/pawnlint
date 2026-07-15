package external

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/externalrule"
)

func TestConvertValidatesAndNamespacesDiagnostics(t *testing.T) {
	files := []externalrule.File{{Path: "main.pwn", Content: "main() {}\n"}}
	values := []externalrule.Diagnostic{{
		RuleID: "example", Severity: "warning", Category: "style", Message: "example", Path: "main.pwn", StartOffset: 0, EndOffset: 4,
		Fix: &externalrule.Fix{Description: "rename", Edits: []externalrule.Edit{{StartOffset: 0, EndOffset: 4, NewText: "entry"}}},
	}}
	diagnostics, err := convert("custom", files, map[string]string{"main.pwn": "/project/main.pwn"}, values)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].RuleID != "external/custom/example" || diagnostics[0].Filename != "/project/main.pwn" || diagnostics[0].Range.End.Col != 5 || diagnostics[0].Fix == nil {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestConvertRejectsInvalidOutput(t *testing.T) {
	files := []externalrule.File{{Path: "main.pwn", Content: "main() {}\n"}}
	values := []externalrule.Diagnostic{{RuleID: "bad/id", Severity: "warning", Category: "style", Message: "example", Path: "main.pwn"}}
	if _, err := convert("custom", files, nil, values); err == nil {
		t.Fatal("invalid rule ID accepted")
	}
	values[0].RuleID = "example"
	values[0].EndOffset = 100
	if _, err := convert("custom", files, nil, values); err == nil {
		t.Fatal("invalid range accepted")
	}
}
