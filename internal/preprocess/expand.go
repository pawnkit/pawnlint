package preprocess

import (
	"sort"
	"strconv"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

const maximumExpansionDepth = 64
const maximumExpandedTokens = 1_000_000

type Result struct {
	Source   []byte
	Parsed   *parser.File
	Complete bool
	Changed  bool
}

type CompactResult struct {
	Source   []byte
	Parsed   *parser.CompactFile
	Complete bool
	Changed  bool
}

type renderedExpansion struct {
	source   []byte
	tokens   []token.Token
	complete bool
	changed  bool
}

type State struct {
	definitions map[string]definition
	undefined   map[string]struct{}
}

type definition struct {
	name       string
	parameters []string
	body       []piece
	function   bool
}

type directive struct {
	node *parser.Node
	last int
}

type piece struct {
	kind    token.Kind
	text    string
	origin  *token.Origin
	span    token.Span
	newline bool
}

type expander struct {
	definitions map[string]definition
	undefined   map[string]struct{}
	complete    bool
	changed     bool
	count       int
}

func Expand(parsed *parser.File, tree *walk.Model, fileID uint32) Result {
	result, _ := ExpandWithState(parsed, tree, fileID, nil, nil)
	return result
}

func ExpandWithState(parsed *parser.File, tree *walk.Model, fileID uint32, initial *State, imports map[int]*State) (Result, *State) {
	expanded, state := expandTokensWithState(parsed, tree, fileID, initial, imports)
	if !expanded.changed {
		return Result{Source: parsed.Source, Parsed: parsed, Complete: expanded.complete}, state
	}
	return Result{
		Source: expanded.source, Parsed: parser.ParseTokens(expanded.source, expanded.tokens),
		Complete: expanded.complete, Changed: true,
	}, state
}

func ExpandCompactWithState(parsed *parser.File, tree *walk.Model, fileID uint32, initial *State, imports map[int]*State) (CompactResult, *State) {
	expanded, state := expandTokensWithState(parsed, tree, fileID, initial, imports)
	if !expanded.changed {
		return CompactResult{Source: parsed.Source, Complete: expanded.complete}, state
	}
	return CompactResult{
		Source:   expanded.source,
		Parsed:   parser.ParseTokensCompact(expanded.source, expanded.tokens, parser.ParseOptions{DiscardTrivia: true}),
		Complete: expanded.complete, Changed: true,
	}, state
}

func expandTokensWithState(parsed *parser.File, tree *walk.Model, fileID uint32, initial *State, imports map[int]*State) (renderedExpansion, *State) {
	if parsed == nil || tree == nil {
		return renderedExpansion{}, nil
	}
	directives := expansionDirectives(parsed, tree)
	current := &expander{definitions: make(map[string]definition), undefined: make(map[string]struct{}), complete: true}
	current.merge(initial)
	var output []piece
	for index := 0; index < len(parsed.Tokens); {
		currentToken := parsed.Tokens[index]
		if currentToken.Kind == token.EOF {
			break
		}
		if item, ok := directives[currentToken.Start.Offset]; ok {
			for ; index <= item.last; index++ {
				if current.changed {
					output = append(output, sourcePiece(parsed.Source, parsed.Tokens[index], fileID, true))
				}
			}
			current.applyDirective(parsed, tree, item.node, fileID)
			current.merge(imports[item.node.Start])
			continue
		}
		if currentToken.Kind != token.Identifier || len(current.definitions) == 0 {
			if current.changed {
				output = append(output, sourcePiece(parsed.Source, currentToken, fileID, true))
			}
			index++
			continue
		}
		input := sourcePiece(parsed.Source, currentToken, fileID, true)
		changed := current.changed
		expanded, consumed := current.expandInvocation(parsed, index, input, fileID, 0, nil)
		if !changed && current.changed {
			output = sourcePiecesOriginal(parsed, 0, index, fileID)
		}
		if current.changed {
			output = append(output, expanded...)
		}
		index += consumed
	}
	if !current.changed {
		return renderedExpansion{source: parsed.Source, complete: current.complete}, current.state()
	}
	source, tokens := render(output)
	return renderedExpansion{source: source, tokens: tokens, complete: current.complete, changed: true}, current.state()
}

func (e *expander) merge(state *State) {
	if state == nil {
		return
	}
	for name := range state.undefined {
		delete(e.definitions, name)
		e.undefined[name] = struct{}{}
	}
	for name, current := range state.definitions {
		e.definitions[name] = current
		delete(e.undefined, name)
	}
}

func (e *expander) state() *State {
	state := &State{definitions: make(map[string]definition, len(e.definitions)), undefined: make(map[string]struct{}, len(e.undefined))}
	for name, current := range e.definitions {
		state.definitions[name] = current
	}
	for name := range e.undefined {
		state.undefined[name] = struct{}{}
	}
	return state
}

func expansionDirectives(parsed *parser.File, tree *walk.Model) map[int]directive {
	result := make(map[int]directive)
	for kind := parser.KindDirectiveInclude; kind <= parser.KindDirectiveRaw; kind++ {
		for _, node := range tree.OfKind(kind) {
			first := sort.Search(len(parsed.Tokens), func(index int) bool {
				return parsed.Tokens[index].Start.Offset >= node.Start
			})
			last := -1
			for index := first; index < len(parsed.Tokens); index++ {
				current := parsed.Tokens[index]
				if current.Kind == token.EOF || current.Start.Offset > node.End {
					break
				}
				if current.Start.Offset >= node.Start && current.End.Offset <= node.End {
					last = index
				}
			}
			if last >= 0 {
				result[parsed.Tokens[first].Start.Offset] = directive{node: node, last: last}
			}
		}
	}
	return result
}

func (e *expander) applyDirective(parsed *parser.File, tree *walk.Model, node *parser.Node, fileID uint32) {
	if tree.Inactive(node) {
		return
	}
	if tree.Uncertain(node) {
		e.complete = false
		return
	}
	nameNode := node.Field("name")
	name := tree.Text(nameNode)
	if node.Kind == parser.KindDirectiveUndef {
		name = directivePayloadName(parsed, node)
	}
	switch node.Kind {
	case parser.KindDirectiveUndef:
		delete(e.definitions, name)
		e.undefined[name] = struct{}{}
	case parser.KindDirectiveDefine:
		if name == "" || node.HasError {
			e.complete = false
			return
		}
		value := node.Field("value")
		body := piecesInRange(parsed, value, fileID)
		parameters := node.Field("parameters")
		definition := definition{name: name, body: body, function: parameters != nil}
		if parameters != nil {
			for _, parameter := range parameters.Children {
				definition.parameters = append(definition.parameters, tree.Text(parameter))
			}
		}
		e.definitions[name] = definition
		delete(e.undefined, name)
	}
}

func directivePayloadName(parsed *parser.File, node *parser.Node) string {
	seen := 0
	for _, current := range parsed.Tokens {
		if current.Start.Offset < node.Start || current.End.Offset > node.End || current.Kind == token.EOF {
			continue
		}
		seen++
		if seen == 3 && current.Kind == token.Identifier {
			return current.Text(parsed.Source)
		}
	}
	return ""
}

func piecesInRange(parsed *parser.File, node *parser.Node, fileID uint32) []piece {
	if node == nil {
		return nil
	}
	var result []piece
	for _, current := range parsed.Tokens {
		if current.Kind == token.EOF || current.Start.Offset > node.End {
			break
		}
		if current.Start.Offset >= node.Start && current.End.Offset <= node.End {
			result = append(result, sourcePiece(parsed.Source, current, fileID, false))
		}
	}
	return result
}

func sourcePiece(source []byte, current token.Token, fileID uint32, newline bool) piece {
	return piece{
		kind:    current.Kind,
		text:    current.Text(source),
		origin:  current.Origin,
		span:    token.Span{File: fileID, Start: current.Start, End: current.End},
		newline: newline && tokenEndsLine(current),
	}
}

func tokenEndsLine(current token.Token) bool {
	for _, trivia := range current.TrailingTrivia {
		if trivia.Kind == token.Newline {
			return true
		}
	}
	return false
}

func (e *expander) expandInvocation(parsed *parser.File, index int, input piece, fileID uint32, depth int, disabled map[string]bool) ([]piece, int) {
	definition, known := e.definitions[input.text]
	if input.kind != token.Identifier || !known {
		return []piece{input}, 1
	}
	if depth >= maximumExpansionDepth || disabled[definition.name] {
		e.complete = false
		return []piece{input}, 1
	}
	consumed := 1
	var arguments [][]piece
	if definition.function {
		if index+1 >= len(parsed.Tokens) || parsed.Tokens[index+1].Kind != token.LParen {
			return []piece{input}, 1
		}
		var ok bool
		arguments, consumed, ok = invocationArguments(parsed, index, fileID)
		if !ok || len(arguments) != len(definition.parameters) {
			e.complete = false
			return sourcePieces(parsed, index, consumed, fileID), consumed
		}
	}
	invocation := &token.Origin{Span: input.span, Macro: definition.name, Parent: input.origin}
	replaced, ok := substitute(definition, arguments, invocation)
	if !ok {
		e.complete = false
		return sourcePieces(parsed, index, consumed, fileID), consumed
	}
	if e.count > maximumExpandedTokens {
		e.complete = false
		return sourcePieces(parsed, index, consumed, fileID), consumed
	}
	nextDisabled := make(map[string]bool, len(disabled)+1)
	for name := range disabled {
		nextDisabled[name] = true
	}
	nextDisabled[definition.name] = true
	result := e.expandPieces(replaced, depth+1, nextDisabled)
	e.changed = true
	return result, consumed
}

func invocationArguments(parsed *parser.File, index int, fileID uint32) ([][]piece, int, bool) {
	depth := 0
	start := index + 2
	var arguments [][]piece
	for current := index + 1; current < len(parsed.Tokens); current++ {
		switch parsed.Tokens[current].Kind {
		case token.LParen, token.LBracket, token.LBrace:
			depth++
		case token.RParen:
			depth--
			if depth == 0 {
				if current > start || len(arguments) != 0 {
					arguments = append(arguments, sourcePieces(parsed, start, current-start, fileID))
				}
				return arguments, current - index + 1, true
			}
		case token.RBracket, token.RBrace:
			depth--
		case token.Comma:
			if depth == 1 {
				arguments = append(arguments, sourcePieces(parsed, start, current-start, fileID))
				start = current + 1
			}
		case token.EOF:
			return nil, current - index, false
		}
		if depth < 0 {
			return nil, current - index + 1, false
		}
	}
	return nil, len(parsed.Tokens) - index, false
}

func sourcePieces(parsed *parser.File, start, count int, fileID uint32) []piece {
	result := make([]piece, 0, count)
	for index := start; index < start+count && index < len(parsed.Tokens); index++ {
		if parsed.Tokens[index].Kind != token.EOF {
			result = append(result, sourcePiece(parsed.Source, parsed.Tokens[index], fileID, false))
		}
	}
	return result
}

func sourcePiecesOriginal(parsed *parser.File, start, count int, fileID uint32) []piece {
	result := make([]piece, 0, max(count, len(parsed.Tokens)-1))
	for index := start; index < start+count && index < len(parsed.Tokens); index++ {
		if parsed.Tokens[index].Kind != token.EOF {
			result = append(result, sourcePiece(parsed.Source, parsed.Tokens[index], fileID, true))
		}
	}
	return result
}

func substitute(definition definition, arguments [][]piece, invocation *token.Origin) ([]piece, bool) {
	indexes := make(map[string]int, len(definition.parameters))
	for index, parameter := range definition.parameters {
		indexes[parameter] = index
		indexes["%"+strconv.Itoa(index)] = index
	}
	var result []piece
	for _, current := range definition.body {
		if current.kind == token.Hash || current.text == "%%" {
			return nil, false
		}
		if index, exists := indexes[current.text]; exists {
			for _, argument := range arguments[index] {
				argument.origin = &token.Origin{Span: argument.span, Macro: definition.name, Parent: invocation}
				result = append(result, argument)
			}
			continue
		}
		current.origin = &token.Origin{Span: current.span, Macro: definition.name, Parent: invocation}
		result = append(result, current)
	}
	return result, true
}

func (e *expander) expandPieces(input []piece, depth int, disabled map[string]bool) []piece {
	var result []piece
	for index := 0; index < len(input); {
		if e.count > maximumExpandedTokens {
			e.complete = false
			return append(result, input[index:]...)
		}
		current := input[index]
		definition, known := e.definitions[current.text]
		if current.kind != token.Identifier || !known {
			result = append(result, current)
			e.count++
			index++
			continue
		}
		if disabled[definition.name] {
			e.complete = false
			result = append(result, current)
			e.count++
			index++
			continue
		}
		if depth >= maximumExpansionDepth {
			e.complete = false
			return append(result, input[index:]...)
		}
		consumed := 1
		var arguments [][]piece
		if definition.function {
			if index+1 >= len(input) || input[index+1].kind != token.LParen {
				result = append(result, current)
				e.count++
				index++
				continue
			}
			var ok bool
			arguments, consumed, ok = pieceArguments(input, index)
			if !ok || len(arguments) != len(definition.parameters) {
				e.complete = false
				result = append(result, input[index:index+consumed]...)
				e.count += consumed
				index += consumed
				continue
			}
		}
		invocation := &token.Origin{Span: current.span, Macro: definition.name, Parent: current.origin}
		replaced, ok := substitute(definition, arguments, invocation)
		if !ok {
			e.complete = false
			result = append(result, input[index:index+consumed]...)
			e.count += consumed
			index += consumed
			continue
		}
		nextDisabled := make(map[string]bool, len(disabled)+1)
		for name := range disabled {
			nextDisabled[name] = true
		}
		nextDisabled[definition.name] = true
		result = append(result, e.expandPieces(replaced, depth+1, nextDisabled)...)
		e.changed = true
		index += consumed
	}
	return result
}

func pieceArguments(input []piece, index int) ([][]piece, int, bool) {
	depth := 0
	start := index + 2
	var arguments [][]piece
	for current := index + 1; current < len(input); current++ {
		switch input[current].kind {
		case token.LParen, token.LBracket, token.LBrace:
			depth++
		case token.RParen:
			depth--
			if depth == 0 {
				if current > start || len(arguments) != 0 {
					arguments = append(arguments, append([]piece(nil), input[start:current]...))
				}
				return arguments, current - index + 1, true
			}
		case token.RBracket, token.RBrace:
			depth--
		case token.Comma:
			if depth == 1 {
				arguments = append(arguments, append([]piece(nil), input[start:current]...))
				start = current + 1
			}
		}
	}
	return nil, len(input) - index, false
}

func render(pieces []piece) ([]byte, []token.Token) {
	sourceLength := len(pieces)
	for _, current := range pieces {
		sourceLength += len(current.text)
	}
	source := make([]byte, 0, sourceLength)
	tokens := make([]token.Token, 0, len(pieces)+1)
	trivia := make([]token.Trivia, len(pieces))
	line, column := 1, 1
	for index, current := range pieces {
		start := token.Position{Offset: len(source), Line: line, Col: column}
		source = append(source, current.text...)
		column += len(current.text)
		end := token.Position{Offset: len(source), Line: line, Col: column}
		output := token.Token{Kind: current.kind, Start: start, End: end, Origin: current.origin}
		separatorStart := end
		if current.newline {
			source = append(source, '\n')
			line++
			column = 1
			separatorEnd := token.Position{Offset: len(source), Line: line, Col: column}
			trivia[index] = token.Trivia{Kind: token.Newline, Start: separatorStart, End: separatorEnd}
		} else {
			source = append(source, ' ')
			column++
			separatorEnd := token.Position{Offset: len(source), Line: line, Col: column}
			trivia[index] = token.Trivia{Kind: token.Whitespace, Start: separatorStart, End: separatorEnd}
		}
		output.TrailingTrivia = trivia[index : index+1 : index+1]
		tokens = append(tokens, output)
	}
	end := token.Position{Offset: len(source), Line: line, Col: column}
	tokens = append(tokens, token.Token{Kind: token.EOF, Start: end, End: end})
	return source, tokens
}
