package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRejectsUnknownTarget(t *testing.T) {
	if _, err := render("other", metadata{}); err == nil {
		t.Fatal("unknown target accepted")
	}
}

func TestLoadExtractsNativesAndDeprecations(t *testing.T) {
	dir := t.TempDir()
	source := "#define TEST_CONSTANT (1)\n#define TEST_MACRO(%0) (%0)\n#define EMPTY\nconst TEST_DECLARATION = 2;\nenum { TEST_ENUM }\nstock Local() { const LOCAL_CONSTANT = 3; }\n#pragma deprecated This stock is broken.\nstock BrokenStock() { return 0; }\nforward OnReady();\n#pragma deprecated Use Replacement.\nforward RemovedFunction();\nnative TestNative(output[], size = sizeof (output), optional = 1, Tag:...);\n#pragma deprecated Use NewNative.\nnative OldNative();\n"
	if err := os.WriteFile(filepath.Join(dir, "omp_core.inc"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := load("openmp", dir)
	if err != nil {
		t.Fatal(err)
	}
	entry := data.Natives["TestNative"]
	if len(entry.Parameters) != 4 || !entry.Parameters[1].Default || !entry.Parameters[2].Default || !entry.Parameters[3].Variadic {
		t.Fatalf("native metadata = %#v", entry)
	}
	if len(entry.Buffers) != 1 || entry.Buffers[0] != (buffer{Parameter: 1, SizeParameter: 2}) {
		t.Fatalf("buffer metadata = %#v", entry.Buffers)
	}
	if !entry.OpenMPOnly {
		t.Fatal("open.mp source classification was not generated")
	}
	if resourceRelease("fopen") != "fclose" || resourceRelease("printf") != "" {
		t.Fatal("resource release classification is incorrect")
	}
	if data.Unsupported["RemovedFunction"].Suggested != "Use Replacement." {
		t.Fatalf("unsupported function metadata = %#v", data.Unsupported["RemovedFunction"])
	}
	if data.DeprecatedFunctions["BrokenStock"].Suggested != "This stock is broken." {
		t.Fatalf("deprecated function metadata = %#v", data.DeprecatedFunctions["BrokenStock"])
	}
	for _, name := range []string{"TEST_CONSTANT", "TEST_DECLARATION", "TEST_ENUM"} {
		if !data.Constants[name].OpenMPOnly {
			t.Fatalf("constant %q was not classified", name)
		}
	}
	if _, ok := data.Constants["TEST_MACRO"]; ok {
		t.Fatal("function-like macro was classified as a constant")
	}
	for _, name := range []string{"EMPTY", "LOCAL_CONSTANT"} {
		if _, ok := data.Constants[name]; ok {
			t.Fatalf("non-API name %q was classified as a constant", name)
		}
	}
	if data.Natives["OldNative"].Deprecated != "Use NewNative." {
		t.Fatalf("deprecation = %q", data.Natives["OldNative"].Deprecated)
	}
}

func TestRenderSortsCallbacks(t *testing.T) {
	generated, err := render("samp", metadata{Callbacks: map[string]callback{
		"OnZed": {Name: "OnZed"}, "OnAlpha": {Name: "OnAlpha"},
	}, Natives: map[string]native{"Zed": {Name: "Zed"}, "Alpha": {Name: "Alpha"}}, Constants: map[string]constant{"ZED": {Name: "ZED"}, "ALPHA": {Name: "ALPHA"}}, Unsupported: map[string]unsupported{"ZedRemoved": {Name: "ZedRemoved"}, "AlphaRemoved": {Name: "AlphaRemoved"}}, DeprecatedFunctions: map[string]deprecatedFunction{"ZedBroken": {Name: "ZedBroken"}, "AlphaBroken": {Name: "AlphaBroken"}}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	if strings.Index(text, "OnAlpha") > strings.Index(text, "OnZed") {
		t.Fatal("callbacks are not sorted")
	}
	if strings.Index(text, `"Alpha"`) > strings.Index(text, `"Zed"`) {
		t.Fatal("natives are not sorted")
	}
	if strings.Index(text, `"ALPHA"`) > strings.Index(text, `"ZED"`) {
		t.Fatal("constants are not sorted")
	}
	if strings.Index(text, `"AlphaRemoved"`) > strings.Index(text, `"ZedRemoved"`) {
		t.Fatal("unsupported functions are not sorted")
	}
	if strings.Index(text, `"AlphaBroken"`) > strings.Index(text, `"ZedBroken"`) {
		t.Fatal("deprecated functions are not sorted")
	}
}

func TestSameCallback(t *testing.T) {
	left := callback{Name: "OnEvent", Parameters: []parameter{{Name: "value", Tag: "bool"}}}
	right := callback{Name: "OnEvent", Parameters: []parameter{{Name: "value", Tag: "bool"}}}
	if !sameCallback(left, right) {
		t.Fatal("equal callbacks differ")
	}
	right.Parameters[0].Reference = true
	if sameCallback(left, right) {
		t.Fatal("different callbacks match")
	}
}
