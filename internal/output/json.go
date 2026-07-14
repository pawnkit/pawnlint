package output

import (
	"encoding/json"
	"io"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

type jsonDiagnostic struct {
	RuleID    string     `json:"ruleId"`
	Severity  string     `json:"severity"`
	Category  string     `json:"category"`
	Message   string     `json:"message"`
	File      string     `json:"file"`
	StartLine int        `json:"startLine"`
	StartCol  int        `json:"startCol"`
	EndLine   int        `json:"endLine"`
	EndCol    int        `json:"endCol"`
	StartOff  int        `json:"startOffset"`
	EndOff    int        `json:"endOffset"`
	Notes     []jsonNote `json:"notes,omitempty"`
	Suggested string     `json:"suggested,omitempty"`
	Fix       *jsonFix   `json:"fix,omitempty"`
}

type jsonNote struct {
	Message   string `json:"message"`
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startCol"`
	StartOff  int    `json:"startOffset"`
	EndOff    int    `json:"endOffset"`
}

type jsonFix struct {
	Description string     `json:"description"`
	Edits       []jsonEdit `json:"edits"`
}

type jsonEdit struct {
	StartOff int    `json:"startOffset"`
	EndOff   int    `json:"endOffset"`
	NewText  string `json:"newText"`
}

func writeJSON(w io.Writer, diags []diagnostic.Diagnostic, line bool) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if line {
		enc.SetIndent("", "")
		for _, d := range diags {
			if err := enc.Encode(toJSON(d)); err != nil {
				return err
			}
		}
		return nil
	}
	out := make([]jsonDiagnostic, 0, len(diags))
	for _, d := range diags {
		out = append(out, toJSON(d))
	}
	return enc.Encode(out)
}

func toJSON(d diagnostic.Diagnostic) jsonDiagnostic {
	j := jsonDiagnostic{
		RuleID:    d.RuleID,
		Severity:  d.Severity.String(),
		Category:  d.Category.String(),
		Message:   d.Message,
		File:      d.Filename,
		StartLine: d.Range.Start.Line,
		StartCol:  d.Range.Start.Col,
		EndLine:   d.Range.End.Line,
		EndCol:    d.Range.End.Col,
		StartOff:  d.Range.Start.Offset,
		EndOff:    d.Range.End.Offset,
		Suggested: d.Suggested,
	}
	for _, n := range d.Notes {
		j.Notes = append(j.Notes, jsonNote{
			Message:   n.Message,
			StartLine: n.Range.Start.Line,
			StartCol:  n.Range.Start.Col,
			StartOff:  n.Range.Start.Offset,
			EndOff:    n.Range.End.Offset,
		})
	}
	if d.Fix != nil {
		jf := &jsonFix{Description: d.Fix.Description}
		for _, e := range d.Fix.Edits {
			jf.Edits = append(jf.Edits, jsonEdit{
				StartOff: e.Range.Start.Offset,
				EndOff:   e.Range.End.Offset,
				NewText:  e.NewText,
			})
		}
		j.Fix = jf
	}
	return j
}
