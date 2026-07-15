package walk

import (
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func IsStatement(n *parser.Node) bool {
	if n == nil {
		return false
	}
	switch n.Kind {
	case parser.KindBlock, parser.KindIfStatement, parser.KindWhileStatement,
		parser.KindDoWhileStatement, parser.KindForStatement,
		parser.KindSwitchStatement, parser.KindCaseClause,
		parser.KindDefaultClause, parser.KindGotoStatement,
		parser.KindLabelStatement, parser.KindReturnStatement,
		parser.KindBreakStatement, parser.KindContinueStatement,
		parser.KindStateStatement, parser.KindExpressionStatement,
		parser.KindEmptyStatement, parser.KindVariableDeclaration,
		parser.KindMacroInvocationBlock:
		return true
	default:
		return false
	}
}

func HasChildToken(node *parser.Node, kind token.Kind) bool {
	if node == nil {
		return false
	}
	for _, child := range node.Children {
		if child.Tok.Kind == kind {
			return true
		}
	}
	return false
}

func HasWrapperStorageQualifier(node *parser.Node) bool {
	storage := node.Field("storage")
	return storage != nil && storage.Tok.Kind == token.Identifier
}

func ReferencesByAmpersand(tokens []token.Token, node *parser.Node) bool {
	if node == nil {
		return false
	}
	name := node.Field("name")
	end := node.End
	if name != nil {
		end = name.Start
	}
	start := sort.Search(len(tokens), func(index int) bool {
		return tokens[index].End.Offset > node.Start
	})
	for _, tok := range tokens[start:] {
		if tok.Start.Offset >= end {
			break
		}
		if tok.Start.Offset >= node.Start && tok.End.Offset <= end && tok.Kind == token.Amp {
			return true
		}
	}
	return false
}
