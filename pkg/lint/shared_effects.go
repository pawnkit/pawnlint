package lint

import (
	"path/filepath"
	"runtime"
	"strings"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func hasSharedFunctionFacts(shared *analysis.Result) bool {
	return shared != nil && len(shared.FunctionFacts) != 0
}

func sharedFunctionEffects(
	shared *analysis.Result,
	declaration project.Declaration,
) (project.FunctionEffects, bool) {
	if shared == nil || declaration.File == nil || declaration.Node == nil ||
		len(shared.FunctionFacts) == 0 || shared.Registry == nil {
		return project.FunctionEffects{}, false
	}
	table := shared.ExpandedSymbols
	if table == nil {
		table = shared.Symbols
	}
	if table == nil {
		return project.FunctionEffects{}, false
	}
	name := declaration.Node.Field("name")
	if name == nil {
		return project.FunctionEffects{}, false
	}
	matchedFile := false
	for _, item := range table.Symbols {
		if item.Name != declaration.Name {
			continue
		}
		uri, ok := shared.Registry.URI(item.Span.File)
		if !ok {
			continue
		}
		filename, err := uri.Filename()
		if err != nil || !sameSharedPath(filename, declaration.File.Path) {
			continue
		}
		matchedFile = true
		if int(item.Span.Start) != name.Start || int(item.Span.End) != name.End {
			continue
		}
		facts, found := shared.FunctionFacts[item.ID]
		if !found {
			return project.FunctionEffects{}, true
		}
		return project.FunctionEffects{
			Complete: facts.Complete,
			Pure: facts.Complete && !facts.IntrinsicImpure &&
				len(facts.ReadsGlobals) == 0 && len(facts.WritesGlobals) == 0 &&
				len(facts.MutatedParameters) == 0,
			MutatedParameters: append([]int(nil), facts.MutatedParameters...),
		}, true
	}
	if matchedFile {
		return project.FunctionEffects{}, true
	}
	return project.FunctionEffects{}, false
}

func sameSharedPath(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	return left == right || runtime.GOOS == "windows" && strings.EqualFold(left, right)
}
