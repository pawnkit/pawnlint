package lint

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func parseErrorDiagnostics(path string, pf *parser.File, lt *source.LineTable) []diagnostic.Diagnostic {
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
