package project

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

type ExpansionOrigin struct {
	File  *File
	Span  token.Span
	Macro string
}

func (m *Model) ExpansionOrigins(file *File, node *parser.Node) []ExpansionOrigin {
	if m == nil || file == nil || file.ExpandedParsed == nil || node == nil {
		return nil
	}
	for _, current := range file.ExpandedParsed.Tokens {
		if current.Kind == token.EOF || current.End.Offset <= node.Start || current.Start.Offset >= node.End || current.Origin == nil {
			continue
		}
		var result []ExpansionOrigin
		for origin := current.Origin; origin != nil; origin = origin.Parent {
			result = append(result, ExpansionOrigin{File: m.sourceFiles[origin.Span.File], Span: origin.Span, Macro: origin.Macro})
		}
		return result
	}
	return nil
}
