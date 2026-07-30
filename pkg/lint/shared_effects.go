package lint

import (
	"path/filepath"
	"runtime"
	"strings"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func sharedLeafFunctionEffects(
	shared *analysis.Result,
	declaration project.Declaration,
) (project.FunctionEffects, bool) {
	if shared == nil || declaration.File == nil || declaration.Node == nil ||
		len(shared.FunctionFacts) == 0 || shared.Registry == nil || shared.Symbols == nil {
		return project.FunctionEffects{}, false
	}
	uri, ok := shared.Registry.URI(shared.File)
	if !ok {
		return project.FunctionEffects{}, false
	}
	filename, err := uri.Filename()
	if err != nil || !sameSharedPath(filename, declaration.File.Path) {
		return project.FunctionEffects{}, false
	}
	name := declaration.Node.Field("name")
	if name == nil {
		return project.FunctionEffects{}, false
	}
	for _, item := range shared.Symbols.Symbols {
		if item.Name != declaration.Name || int(item.Span.Start) != name.Start || int(item.Span.End) != name.End {
			continue
		}
		facts, found := shared.FunctionFacts[item.ID]
		if !found || !facts.Complete || len(facts.Calls) != 0 {
			return project.FunctionEffects{}, false
		}
		return project.FunctionEffects{
			Complete:          true,
			Pure:              !facts.IntrinsicImpure && len(facts.ReadsGlobals) == 0 && len(facts.WritesGlobals) == 0 && len(facts.MutatedParameters) == 0,
			MutatedParameters: append([]int(nil), facts.MutatedParameters...),
		}, true
	}
	return project.FunctionEffects{}, false
}

func sameSharedPath(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	return left == right || runtime.GOOS == "windows" && strings.EqualFold(left, right)
}
