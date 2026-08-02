package lint

import (
	"path/filepath"
	"runtime"
	"strings"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-analysis/sema"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type sharedFunctionKey struct {
	path       string
	name       string
	start, end int
}

type sharedFunctionNameKey struct {
	path string
	name string
}

type sharedFunctionIndex struct {
	effects map[sharedFunctionKey]project.FunctionEffects
	names   map[sharedFunctionNameKey]struct{}
}

func newSharedFunctionIndex(shared *analysis.Result) *sharedFunctionIndex {
	if !hasSharedFunctionFacts(shared) || shared.Registry == nil {
		return nil
	}
	table := shared.ExpandedSymbols
	if table == nil {
		table = shared.Symbols
	}
	if table == nil {
		return nil
	}
	index := &sharedFunctionIndex{
		effects: make(map[sharedFunctionKey]project.FunctionEffects),
		names:   make(map[sharedFunctionNameKey]struct{}),
	}
	for _, item := range table.Symbols {
		if !item.Kind.IsCallable() {
			continue
		}
		uri, ok := shared.Registry.URI(item.Span.File)
		if !ok {
			continue
		}
		path, err := uri.Filename()
		if err != nil {
			continue
		}
		path = sharedPathKey(path)
		index.names[sharedFunctionNameKey{path: path, name: item.Name}] = struct{}{}
		facts, ok := shared.FunctionFacts[item.ID]
		if !ok {
			continue
		}
		index.effects[sharedFunctionKey{
			path: path, name: item.Name, start: int(item.Span.Start), end: int(item.Span.End),
		}] = functionEffects(facts)
	}
	return index
}

func functionEffects(facts sema.FunctionFacts) project.FunctionEffects {
	return project.FunctionEffects{
		Complete: facts.Complete,
		Pure: facts.Complete && !facts.IntrinsicImpure &&
			len(facts.ReadsGlobals) == 0 && len(facts.WritesGlobals) == 0 &&
			len(facts.MutatedParameters) == 0,
		MutatedParameters: append([]int(nil), facts.MutatedParameters...),
	}
}

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
		return functionEffects(facts), true
	}
	if matchedFile {
		return project.FunctionEffects{}, true
	}
	return project.FunctionEffects{}, false
}

func sharedFunctionEffectsIndexed(
	shared *analysis.Result,
	declaration project.Declaration,
	index *sharedFunctionIndex,
) (project.FunctionEffects, bool) {
	if index == nil {
		return sharedFunctionEffects(shared, declaration)
	}
	if shared == nil || declaration.File == nil || declaration.Node == nil ||
		len(shared.FunctionFacts) == 0 {
		return project.FunctionEffects{}, false
	}
	name := declaration.Node.Field("name")
	if name == nil {
		return project.FunctionEffects{}, false
	}
	path := sharedPathKey(declaration.File.Path)
	key := sharedFunctionKey{path: path, name: declaration.Name, start: name.Start, end: name.End}
	if effects, ok := index.effects[key]; ok {
		return effects, true
	}
	if _, ok := index.names[sharedFunctionNameKey{path: path, name: declaration.Name}]; ok {
		return project.FunctionEffects{}, true
	}
	return project.FunctionEffects{}, false
}

func sharedPathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func sameSharedPath(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	return left == right || runtime.GOOS == "windows" && strings.EqualFold(left, right)
}
