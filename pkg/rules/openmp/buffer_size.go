package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type BufferSize struct{}

func (BufferSize) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "buffer-size",
		Name:            "Buffer size",
		Summary:         "Reports native size arguments larger than a declared buffer",
		Explanation:     "Official native declarations link output arrays to capacity parameters with defaults such as sizeof(buffer). The rule reports only direct array arguments with one known dimension and a definite oversized value.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"buffer", "arrays", "native", "api"},
	}
}

func (BufferSize) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		native, name, ok := calledNative(ctx, node)
		if !ok || len(native.Buffers) == 0 {
			return
		}
		arguments := node.Field("arguments")
		if _, ok := argumentCount(ctx, arguments); !ok || hasNamedArgument(arguments) {
			return
		}
		for _, relation := range native.Buffers {
			bufferIndex := relation.Parameter - 1
			sizeIndex := relation.SizeParameter - 1
			if bufferIndex < 0 || sizeIndex < 0 || bufferIndex >= len(arguments.Children) || sizeIndex >= len(arguments.Children) {
				continue
			}
			bufferNode := unwrapParentheses(arguments.Children[bufferIndex])
			if bufferNode == nil || bufferNode.Kind != parser.KindIdentifier {
				continue
			}
			symbol := ctx.Semantic.Resolve(bufferNode)
			capacity, ok := arrayCapacity(ctx, symbol)
			if !ok {
				continue
			}
			sizeNode := arguments.Children[sizeIndex]
			size, ok := ctx.Eval(sizeNode)
			if !ok || size <= capacity {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("size %d exceeds the declared capacity %d of %q in native %q", size, capacity, symbol.Name, name),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(sizeNode),
			})
		}
	})
}

func arrayCapacity(ctx *lint.Context, symbol *semantic.Symbol) (int64, bool) {
	if symbol == nil || symbol.Ambiguous || symbol.Decl == nil {
		return 0, false
	}
	var dimension *parser.Node
	count := 0
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			dimension = child
			count++
		}
	}
	if count != 1 || dimension.Field("packed") != nil {
		return 0, false
	}
	capacity, ok := ctx.Semantic.Eval(dimension.Field("size"))
	return capacity, ok && capacity > 0
}
