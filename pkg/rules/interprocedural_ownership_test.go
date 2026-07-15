package rules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestInterproceduralOwnershipInference(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "resource.inc")
	mainPath := filepath.Join(dir, "main.pwn")
	include := `stock File:OpenInner()
{
	return OpenPrimitive();
}

stock File:OpenWrapped()
{
	return OpenInner();
}

stock CloseInner(File:resource)
{
	ClosePrimitive(resource);
}

stock CloseWrapped(File:resource)
{
	CloseInner(resource);
}

stock File:OpenConditional(bool:open)
{
	if (open)
	{
		return OpenPrimitive();
	}
	return File:0;
}

stock CloseConditional(File:resource, bool:close)
{
	if (close)
	{
		ClosePrimitive(resource);
	}
}
`
	main := `#include "resource.inc"

main()
{
	OpenWrapped();
	new File:leaked = OpenWrapped();
	new File:closed = OpenWrapped();
	new File:alias = closed;
	CloseWrapped(alias);
	InspectPrimitive(closed);
	new File:uncertain = OpenWrapped();
	CloseConditional(uncertain, true);
	InspectPrimitive(uncertain);
	new File:not_tracked = OpenConditional(true);
	InspectPrimitive(not_tracked);
}
`
	if err := os.WriteFile(includePath, []byte(include), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	model, err := project.Build([]project.Source{{Path: mainPath, Content: []byte(main)}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := api.Merge("openmp", &api.Metadata{Functions: map[string]api.Function{
		"OpenPrimitive":    {ReturnTag: "File", Release: "ClosePrimitive"},
		"ClosePrimitive":   {Parameters: []api.Parameter{{Name: "resource", Tag: "File", Ownership: "transferred"}}},
		"InspectPrimitive": {Parameters: []api.Parameter{{Name: "resource", Tag: "File", Ownership: "borrowed"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	unreleased := lintProjectRule(t, model, metadata, mainPath, "unreleased-resource-handle")
	if len(unreleased) != 1 || !strings.Contains(unreleased[0].Message, `resource handle "leaked"`) {
		t.Fatalf("unreleased diagnostics = %#v", unreleased)
	}
	reads := lintProjectRule(t, model, metadata, mainPath, "read-after-release")
	if len(reads) != 1 || !strings.Contains(reads[0].Message, `resource handle "closed" is used after release`) {
		t.Fatalf("read diagnostics = %#v", reads)
	}
	discarded := lintProjectRule(t, model, metadata, mainPath, "discarded-resource-handle")
	if len(discarded) != 1 || !strings.Contains(discarded[0].Message, `returned by "OpenWrapped" is discarded`) {
		t.Fatalf("discarded diagnostics = %#v", discarded)
	}
}
