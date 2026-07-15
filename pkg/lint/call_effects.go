package lint

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func (e *Engine) controlFlowOptions(file *project.File, tree *walk.Model) controlflow.Options {
	return controlflow.Options{ResolveCallEffects: func(call *parser.Node) (controlflow.CallEffects, bool) {
		return e.resolveCallEffects(file, tree, call)
	}}
}

func (e *Engine) resolveCallEffects(file *project.File, tree *walk.Model, call *parser.Node) (controlflow.CallEffects, bool) {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return controlflow.CallEffects{}, false
	}
	name := tree.Text(callee)
	if file != nil && file.Walk != nil {
		name = file.Walk.Text(callee)
	}
	if e.Project != nil && file != nil {
		variants := e.Project.FunctionVariants(file, callee)
		if len(variants) != 0 {
			mutated := make(map[int]bool)
			projectFunction := false
			complete := true
			for _, variant := range variants {
				if variant.Kind != semantic.SymbolFunction || variant.Node == nil || walk.HasChildToken(variant.Node, token.KwNative) {
					continue
				}
				projectFunction = true
				effects, known := e.Project.FunctionEffects(variant)
				if !known || !effects.Complete {
					complete = false
					continue
				}
				for _, index := range effects.MutatedParameters {
					mutated[index] = true
				}
			}
			if projectFunction {
				return controlflow.CallEffects{Complete: complete, MutatedArguments: sortedCallEffectIndexes(mutated)}, true
			}
		}
		for _, declaration := range e.Project.Declarations[name] {
			if declaration.Kind == semantic.SymbolFunction && len(variants) == 0 {
				return controlflow.CallEffects{}, true
			}
		}
	}
	if native, ok := e.natives()[name]; ok {
		return apiCallEffects(native.Parameters, native.Buffers), true
	}
	if function, ok := e.functions()[name]; ok {
		return apiCallEffects(function.Parameters, nil), true
	}
	return controlflow.CallEffects{}, false
}

func (e *Engine) natives() map[string]api.Native {
	if e.API != nil {
		return e.API.Natives
	}
	return api.Natives(e.Target)
}

func (e *Engine) functions() map[string]api.Function {
	if e.API != nil {
		return e.API.Functions
	}
	return nil
}

func apiCallEffects(parameters []api.Parameter, buffers []api.Buffer) controlflow.CallEffects {
	mutated := make(map[int]bool)
	for index, parameter := range parameters {
		if parameter.Reference || parameter.Output || parameter.ArrayRank > 0 && !parameter.Const {
			mutated[index] = true
		}
	}
	for _, buffer := range buffers {
		if buffer.Parameter > 0 {
			mutated[buffer.Parameter-1] = true
		}
	}
	return controlflow.CallEffects{Complete: true, MutatedArguments: sortedCallEffectIndexes(mutated)}
}

func sortedCallEffectIndexes(indexes map[int]bool) []int {
	result := make([]int, 0, len(indexes))
	for index := range indexes {
		result = append(result, index)
	}
	sort.Ints(result)
	return result
}
