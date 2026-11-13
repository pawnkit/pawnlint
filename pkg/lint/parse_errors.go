package lint

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-parser"
	corediagnostic "github.com/pawnkit/pawnkit-core/diagnostic"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

// Shared analysis reconciles raw and expanded syntax, including diagnostics
// whose original spelling is a valid macro invocation. Use it for the same
// file and revision while continuing to run ordinary lint rules on the CST.
func sharedParseErrors(path string, content []byte, lines *source.LineTable, shared *analysis.Result) ([]diagnostic.Diagnostic, bool) {
	if shared == nil || shared.Parse == nil || shared.Parse.Broken || shared.Registry == nil ||
		!bytes.Equal(shared.Parse.Source, content) {
		return nil, false
	}
	uri, ok := shared.Registry.URI(shared.File)
	if !ok || uri != coresource.FileURI(path) {
		return nil, false
	}
	var findings []diagnostic.Diagnostic
	for _, item := range shared.Diagnostics {
		if item.Primary.File != shared.File || item.Source != "pawn-parser" {
			continue
		}
		start, end := boundedParserRange(int(item.Primary.Start), int(item.Primary.End), len(content))
		findings = append(findings, diagnostic.Diagnostic{
			RuleID: ParseErrorID, Code: strings.TrimPrefix(item.Code, "pawn-parser:"),
			Severity: sharedSeverity(item.Severity), Category: diagnostic.CategoryCorrectness,
			Message: item.Message, Filename: path, Range: lines.Range(start, end),
			Fix: sharedParserFix(item.SafeFixes, shared.File, len(content), lines),
		})
	}
	return findings, true
}

func sharedParserFix(fixes []corediagnostic.Fix, file coresource.FileID, size int, lines *source.LineTable) *diagnostic.Fix {
	for _, fix := range fixes {
		if len(fix.Edit.Documents) != 1 || fix.Edit.Documents[0].File != file {
			continue
		}
		edits := fix.Edit.Documents[0].Edits
		result := &diagnostic.Fix{Description: fix.Message}
		for _, edit := range edits {
			start, end := int(edit.Span.Start), int(edit.Span.End)
			if edit.Span.File != file || start < 0 || end < start || end > size {
				break
			}
			result.Edits = append(result.Edits, diagnostic.Edit{
				Range: lines.Range(start, end), NewText: edit.NewText,
			})
		}
		if len(result.Edits) != 0 && len(result.Edits) == len(edits) {
			return result
		}
	}
	return nil
}

func parseErrorDiagnostics(path string, pf *parser.File, lt *source.LineTable) []diagnostic.Diagnostic {
	if pf != nil && len(pf.Diagnostics) != 0 {
		out := make([]diagnostic.Diagnostic, 0, len(pf.Diagnostics))
		for _, item := range pf.Diagnostics {
			start, end := boundedParserRange(item.Range.Start, item.Range.End, len(pf.Source))
			d := diagnostic.Diagnostic{
				RuleID:   ParseErrorID,
				Code:     string(item.Code),
				Severity: diagnostic.SeverityError,
				Category: diagnostic.CategoryCorrectness,
				Message:  item.Message,
				Filename: path,
				Range:    lt.Range(start, end),
			}
			if fix := parserRecoveryFix(item.Recovery, len(pf.Source), lt); fix != nil {
				d.Fix = fix
			}
			out = append(out, d)
		}
		return out
	}
	return fallbackParseErrorDiagnostics(path, pf, lt)
}

func parserRecoveryFix(recovery parser.Recovery, sourceSize int, lt *source.LineTable) *diagnostic.Fix {
	if recovery.Confidence != parser.RecoveryExact || recovery.Kind == parser.RecoveryNone {
		return nil
	}
	start, end := boundedParserRange(recovery.Range.Start, recovery.Range.End, sourceSize)
	if start != recovery.Range.Start || end != recovery.Range.End {
		return nil
	}
	var description string
	switch recovery.Kind {
	case parser.RecoveryInsert:
		if start != end || recovery.Replacement == "" {
			return nil
		}
		description = fmt.Sprintf("insert %q", recovery.Replacement)
	case parser.RecoveryRemove:
		if start == end || recovery.Replacement != "" {
			return nil
		}
		description = "remove the unexpected token"
	case parser.RecoveryReplace:
		if start == end || recovery.Replacement == "" {
			return nil
		}
		description = fmt.Sprintf("replace the unexpected syntax with %q", recovery.Replacement)
	default:
		return nil
	}
	return &diagnostic.Fix{
		Description: description,
		Edits: []diagnostic.Edit{{
			Range:   lt.Range(start, end),
			NewText: recovery.Replacement,
		}},
	}
}

func boundedParserRange(start, end, sourceSize int) (int, int) {
	if start < 0 {
		start = 0
	}
	if start > sourceSize {
		start = sourceSize
	}
	if end < start {
		end = start
	}
	if end > sourceSize {
		end = sourceSize
	}
	return start, end
}

func fallbackParseErrorDiagnostics(path string, pf *parser.File, lt *source.LineTable) []diagnostic.Diagnostic {
	if pf == nil {
		return nil
	}
	var nodes []*parser.Node
	var visit func(*parser.Node) bool
	visit = func(n *parser.Node) bool {
		if n == nil {
			return false
		}
		childError := false
		for _, child := range n.Children {
			if visit(child) {
				childError = true
			}
		}
		if n.HasError && !childError {
			nodes = append(nodes, n)
		}
		return n.HasError || childError
	}
	visit(pf.Root)
	if !pf.HasParseErrors() {
		return nil
	}
	if len(nodes) == 0 {
		nodes = append(nodes, pf.Root)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Start < nodes[j].Start
	})
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, len(nodes))
	for _, n := range nodes {
		start, end := 0, min(1, len(pf.Source))
		if n != nil {
			start, end = n.Start, n.End
			if end <= start {
				end = start + 1
			}
		}
		if len(spans) != 0 {
			last := &spans[len(spans)-1]
			if start == last.start {
				if end > last.end {
					last.end = end
				}
				continue
			}
			if lt.Lookup(start).Line <= lt.Lookup(last.end).Line+1 {
				if end > last.end {
					last.end = end
				}
				continue
			}
		}
		spans = append(spans, span{start: start, end: end})
	}
	out := make([]diagnostic.Diagnostic, 0, len(spans))
	for _, span := range spans {
		out = append(out, diagnostic.Diagnostic{
			RuleID:   ParseErrorID,
			Severity: diagnostic.SeverityError,
			Category: diagnostic.CategoryCorrectness,
			Message:  "source contains syntax the parser could not understand",
			Filename: path,
			Range:    lt.Range(span.start, span.end),
		})
	}
	return out
}
