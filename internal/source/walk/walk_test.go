package walk_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func mustParse(t *testing.T, src string) *parser.File {
	t.Helper()
	f := parser.Parse([]byte(src))
	if f == nil || f.Root == nil {
		t.Fatalf("nil root for %q", src)
	}
	return f
}

func TestModelIndexAndParent(t *testing.T) {
	src := "main()\n{\n if (a)\n {\n }\n}\n"
	f := mustParse(t, src)
	m := walk.New("x.pwn", f)
	if m.Root() == nil {
		t.Fatal("nil root")
	}
	ifs := m.OfKind(parser.KindIfStatement)
	if len(ifs) != 1 {
		t.Fatalf("ifs: %d", len(ifs))
	}
	parent := m.Parent(ifs[0])
	if parent == nil || parent.Kind != parser.KindBlock {
		t.Errorf("parent kind %v", kindOf(parent))
	}
	anc := m.Ancestors(ifs[0])
	if len(anc) == 0 {
		t.Fatal("no ancestors")
	}
}

func kindOf(n *parser.Node) parser.Kind {
	if n == nil {
		return parser.KindInvalid
	}
	return n.Kind
}

func TestRangeAndText(t *testing.T) {
	src := "ab cd\nmain(){}\n"
	f := mustParse(t, src)
	m := walk.New("x.pwn", f)
	r := m.Range(m.Root())
	if r.Start.Offset != 0 {
		t.Errorf("start %d", r.Start.Offset)
	}
	if m.Text(m.Root()) == "" {
		t.Error("empty text")
	}
}

func TestEnclosingFunctionAndConditional(t *testing.T) {
	src := "stock f()\n{\n #if defined X\n return 1;\n #endif\n}\n"
	f := mustParse(t, src)
	m := walk.New("x.pwn", f)
	rets := m.OfKind(parser.KindReturnStatement)
	if len(rets) < 1 {
		t.Fatal("no return")
	}
	if !m.IsInsideConditionalBranch(rets[0]) {
		t.Error("return should be inside conditional")
	}
	if m.EnclosingFunction(rets[0]) == nil {
		t.Error("enclosing function nil")
	}
}

func TestConditionalLiteralState(t *testing.T) {
	src := "main() {\n#if 08\nnew active;\n#else\nnew inactive;\n#endif\n#if FEATURE\nnew uncertain;\n#endif\n}\n"
	f := mustParse(t, src)
	m := walk.New("x.pwn", f)
	declarations := m.OfKind(parser.KindVariableDeclarator)
	if len(declarations) != 3 {
		t.Fatalf("declarations: %d", len(declarations))
	}
	if m.Uncertain(declarations[0]) || m.Inactive(declarations[0]) {
		t.Fatal("known active branch was uncertain")
	}
	if !m.Uncertain(declarations[1]) || !m.Inactive(declarations[1]) {
		t.Fatal("known inactive branch was not inactive")
	}
	if !m.Uncertain(declarations[2]) || m.Inactive(declarations[2]) {
		t.Fatal("unknown branch state was incorrect")
	}
}

func TestConditionalDefinedState(t *testing.T) {
	src := "#define LOCAL\n#if defined LOCAL\nnew local_active;\n#endif\n#if defined CONFIGURED\nnew configured_active;\n#endif\n"
	f := mustParse(t, src)
	m := walk.NewWithDefines("x.pwn", f, []string{"CONFIGURED"})
	declarations := m.OfKind(parser.KindVariableDeclarator)
	if len(declarations) != 2 {
		t.Fatalf("declarations: %d", len(declarations))
	}
	for _, declaration := range declarations {
		if m.Uncertain(declaration) || m.Inactive(declaration) {
			t.Fatalf("known defined branch %q was uncertain", m.Text(declaration))
		}
	}
}

func TestCompleteDefineContextTreatsAbsentNamesAsUndefined(t *testing.T) {
	src := "#if defined MISSING\nnew inactive;\n#else\nnew active;\n#endif\n"
	f := mustParse(t, src)
	m := walk.NewWithDefineContext("x.pwn", f, nil, nil, true)
	declarations := m.OfKind(parser.KindVariableDeclarator)
	if len(declarations) != 2 {
		t.Fatalf("declarations = %d", len(declarations))
	}
	if !m.Inactive(declarations[0]) || m.Uncertain(declarations[1]) || m.Inactive(declarations[1]) {
		t.Fatalf("complete context states are incorrect")
	}
}

func TestKnownDefinesTracksActiveConditionalDefinesAndSpecificUndef(t *testing.T) {
	src := "#define KEEP\n#define REMOVE\n#if defined ENABLED\n#define CONDITIONAL\n#endif\n#undef REMOVE\n"
	f := mustParse(t, src)
	m := walk.NewWithDefineContext("x.pwn", f, []string{"ENABLED"}, nil, true)
	defines := m.KnownDefinesAt(len(src) + 1)
	if !hasString(defines, "KEEP") || !hasString(defines, "CONDITIONAL") || hasString(defines, "REMOVE") {
		t.Fatalf("defines = %v", defines)
	}
}

func TestDefineSnapshotAffectsLaterConditional(t *testing.T) {
	src := "#include \"shared.inc\"\n#if defined EXPORTED\nnew active;\n#endif\n"
	f := mustParse(t, src)
	include := f.Root.Children[0]
	m := walk.NewWithDefineContext("x.pwn", f, nil, []walk.DefineSnapshot{{Offset: include.End, Defines: []string{"EXPORTED"}}}, true)
	declarations := m.OfKind(parser.KindVariableDeclarator)
	if len(declarations) != 1 || m.Uncertain(declarations[0]) || m.Inactive(declarations[0]) {
		t.Fatalf("snapshot did not activate later declaration")
	}
}

func hasString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func TestCompilerPredefinedState(t *testing.T) {
	src := "#if defined __PawnBuild\nnew active;\n#endif\n"
	f := mustParse(t, src)
	m := walk.New("x.pwn", f)
	declarations := m.OfKind(parser.KindVariableDeclarator)
	if len(declarations) != 1 || m.Uncertain(declarations[0]) {
		t.Fatalf("compiler predefined branch was uncertain: %+v", declarations)
	}
}

func TestRootErrorDoesNotTaintChildren(t *testing.T) {
	child := &parser.Node{Kind: parser.KindIdentifier}
	root := &parser.Node{Kind: parser.KindSourceFile, HasError: true, Children: []*parser.Node{child}}
	m := walk.New("x.pwn", &parser.File{Root: root})
	if m.Uncertain(child) {
		t.Fatal("root aggregate error tainted child")
	}
}

func TestIsStatement(t *testing.T) {
	if !walk.IsStatement(&parser.Node{Kind: parser.KindIfStatement}) {
		t.Error("if should be statement")
	}
	if walk.IsStatement(&parser.Node{Kind: parser.KindIdentifier}) {
		t.Error("identifier not statement")
	}
}

func TestTokenText(t *testing.T) {
	src := "x;\n"
	f := mustParse(t, src)
	m := walk.New("x.pwn", f)
	if len(f.Tokens) == 0 {
		t.Fatal("no tokens")
	}
	if m.TokenText(f.Tokens[0]) == "" {
		t.Error("empty token text")
	}
	_ = source.Range{}
	_ = token.Identifier
}

func FuzzWalk(f *testing.F) {
	f.Add([]byte("main(){}"))
	f.Add([]byte("#if defined A\nx;\n#else\ny;\n#endif\n"))
	f.Add([]byte("new a[10]; main(){ a[0]=1; }\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		pf := parser.Parse(src)
		if pf == nil || pf.Root == nil {
			return
		}
		m := walk.New("fuzz.pwn", pf)
		m.Iter(func(n *parser.Node) {
			_ = m.Range(n)
			_ = m.Text(n)
			_ = m.EnclosingFunction(n)
			_ = m.IsInsideConditionalBranch(n)
			_ = m.Inactive(n)
			_ = m.Parent(n)
		})
	})
}
