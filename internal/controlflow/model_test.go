package controlflow_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func build(t *testing.T, src string) (*controlflow.Function, *walk.Model) {
	t.Helper()
	file := parser.Parse([]byte(src))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	if len(model.Functions) != 1 {
		t.Fatalf("functions = %d", len(model.Functions))
	}
	return model.Functions[0], tree
}

func TestReturnMakesFollowingStatementUnreachable(t *testing.T) {
	function, tree := build(t, "main() { return; new value; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	if function.Reachable(declaration) {
		t.Fatal("declaration is reachable")
	}
	if function.CanFallThrough() {
		t.Fatal("function falls through")
	}
}

func TestGotoReachesLabel(t *testing.T) {
	function, tree := build(t, "main() { goto done; new value; done: value = 1; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	label := tree.OfKind(parser.KindLabelStatement)[0]
	assignment := tree.OfKind(parser.KindExpressionStatement)[0]
	if function.Reachable(declaration) {
		t.Fatal("declaration is reachable")
	}
	if !function.Reachable(label) || !function.Reachable(assignment) {
		t.Fatal("goto target is unreachable")
	}
}

func TestBothIfBranchesTerminate(t *testing.T) {
	function, tree := build(t, "main() { if (value) return; else return; new result; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	if function.Reachable(declaration) {
		t.Fatal("declaration is reachable")
	}
}

func TestOneIfBranchFallsThrough(t *testing.T) {
	function, tree := build(t, "main() { if (value) return; new result; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	if !function.Reachable(declaration) {
		t.Fatal("declaration is unreachable")
	}
}

func TestInfiniteLoopDoesNotFallThrough(t *testing.T) {
	function, tree := build(t, "main() { while (true) {} new result; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	if function.Reachable(declaration) || function.CanFallThrough() {
		t.Fatal("infinite loop falls through")
	}
}

func TestBreakLeavesInfiniteLoop(t *testing.T) {
	function, tree := build(t, "main() { while (true) { break; } new result; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	if !function.Reachable(declaration) || !function.CanFallThrough() {
		t.Fatal("break does not leave loop")
	}
}

func TestDoWhileReturnDoesNotFallThrough(t *testing.T) {
	function, tree := build(t, "main() { do { return; } while (false); new result; }")
	declaration := tree.OfKind(parser.KindVariableDeclaration)[0]
	if function.Reachable(declaration) || function.CanFallThrough() {
		t.Fatal("terminating do loop falls through")
	}
}

func TestDoWhileConditionUsesConditionBlock(t *testing.T) {
	function, tree := build(t, "main() { new value; do { value = 1; } while (value); }")
	loop := tree.OfKind(parser.KindDoWhileStatement)[0]
	condition := loop.Field("condition")
	bodyAssignment := tree.OfKind(parser.KindAssignmentExpression)[0]
	if function.Block(condition) == nil || function.Block(condition) == function.Block(bodyAssignment) {
		t.Fatal("condition and body share a block")
	}
}

func TestValuePropagationThroughAssignment(t *testing.T) {
	file := parser.Parse([]byte("main() { new value; value = 2; if (value == 2) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	value, ok := model.Eval(condition)
	if !ok || value != 1 {
		t.Fatalf("condition = %d, %v", value, ok)
	}
}

func TestValuePropagationThroughInitializer(t *testing.T) {
	file := parser.Parse([]byte("new values[4]; main() { new index = 4; values[index] = 1; }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	index := tree.OfKind(parser.KindSubscriptExpression)[0].Field("index")
	value, ok := model.Eval(index)
	if !ok || value != 4 {
		t.Fatalf("index = %d, %v", value, ok)
	}
}

func TestValuePropagationJoinsMatchingBranches(t *testing.T) {
	file := parser.Parse([]byte("main(bool:check) { new value; if (check) value = 2; else value = 2; if (value == 2) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[1].Field("condition")
	value, ok := model.Eval(condition)
	if !ok || value != 1 {
		t.Fatalf("condition = %d, %v", value, ok)
	}
}

func TestValuePropagationRejectsDivergentBranches(t *testing.T) {
	file := parser.Parse([]byte("main(bool:check) { new value; if (check) value = 1; else value = 2; if (value == 2) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[1].Field("condition")
	if _, ok := model.Eval(condition); ok {
		t.Fatal("divergent value is known")
	}
}

func TestValuePropagationInvalidatesCallArguments(t *testing.T) {
	file := parser.Parse([]byte("main() { new value = 1; Change(value); if (value == 1) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	if _, ok := model.Eval(condition); ok {
		t.Fatal("call argument value is known")
	}
}

func TestValuePropagationKeepsByValueCallArguments(t *testing.T) {
	file := parser.Parse([]byte("Read(value) {} main() { new value = 1; Read(value); if (value == 1) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	value, ok := model.Eval(condition)
	if !ok || value != 1 {
		t.Fatalf("condition = %d, %v", value, ok)
	}
}

func TestValuePropagationInvalidatesReferenceArguments(t *testing.T) {
	file := parser.Parse([]byte("Change(&value) { value = 2; } main() { new value = 1; Change(value); if (value == 1) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	if _, ok := model.Eval(condition); ok {
		t.Fatal("reference argument value is known")
	}
}

func TestValuePropagationKeepsSubscriptIndexes(t *testing.T) {
	file := parser.Parse([]byte("Change(values[]) {} main() { new values[2]; new index = 1; Change(values[index]); if (index == 1) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	value, ok := model.Eval(condition)
	if !ok || value != 1 {
		t.Fatalf("condition = %d, %v", value, ok)
	}
}

func TestValuePropagationUsesResolvedCallEffects(t *testing.T) {
	file := parser.Parse([]byte("Change(value) {} main() { new value = 1; Change(value); if (value == 1) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	calls := 0
	model := controlflow.BuildWithOptions(tree, semantics, controlflow.Options{ResolveCallEffects: func(call *parser.Node) (controlflow.CallEffects, bool) {
		calls++
		return controlflow.CallEffects{Complete: true, MutatedArguments: []int{0}}, true
	}})
	if calls != 0 {
		t.Fatal("value propagation was built eagerly")
	}
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	if _, ok := model.Eval(condition); ok {
		t.Fatal("resolved mutation was ignored")
	}
	if calls == 0 {
		t.Fatal("call effects were not resolved")
	}
}

func TestValuePropagationKeepsLoopInvariant(t *testing.T) {
	file := parser.Parse([]byte("forward Check(); main() { new value = 2; while (Check()) {} if (value == 2) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	value, ok := model.Eval(condition)
	if !ok || value != 1 {
		t.Fatalf("condition = %d, %v", value, ok)
	}
}

func TestValuePropagationRejectsLoopMutation(t *testing.T) {
	file := parser.Parse([]byte("forward Check(); main() { new value = 2; while (Check()) { value = 3; } if (value == 2) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	if _, ok := model.Eval(condition); ok {
		t.Fatal("loop-mutated value is known")
	}
}

func TestValuePropagationRejectsConditionalAssignment(t *testing.T) {
	file := parser.Parse([]byte("main(bool:check) { new value = 0; check ? (value = 1) : 0; if (value == 1) {} }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	if _, ok := model.Eval(condition); ok {
		t.Fatal("conditionally assigned value is known")
	}
}
