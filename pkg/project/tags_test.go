package project

import (
	"path/filepath"
	"testing"
)

func TestTagAliasesExpandFunctionMacros(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`#define __TAG(%0) T_%0
#define VEHICLE_TYRE_STATUS __TAG(VEHICLE_TYRE_STATUS)
#define VEHICLE_TIRE_STATUS __TAG(VEHICLE_TYRE_STATUS)
new VEHICLE_TIRE_STATUS:tires;
`)
	model, err := Build([]Source{{Path: path, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	file := model.File(path)
	tags := file.normalizeTags([]string{"VEHICLE_TYRE_STATUS", "VEHICLE_TIRE_STATUS"})
	if len(tags) != 2 || tags[0] != "T_VEHICLE_TYRE_STATUS" || tags[1] != "T_VEHICLE_TYRE_STATUS" {
		t.Fatalf("normalized tags = %v, aliases = %v", tags, file.tagAliases)
	}
}
