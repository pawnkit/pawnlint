package project

import (
	"path/filepath"
	"testing"
)

func FuzzFunctionEffects(f *testing.F) {
	f.Add("Pure(value) { return value + 1; }\n")
	f.Add("new shared; Mutate(&value) { value = shared; } Wrap(&value) { Mutate(value); }\n")
	f.Add("Recur(value) { if (value) return Recur(value - 1); return value; }\n")
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 32*1024 {
			t.Skip()
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "main.pwn")
		model, err := Build([]Source{{Path: path, Content: []byte(input)}}, Options{WorkingDir: dir, DefinesComplete: true})
		if err != nil {
			return
		}
		if model.CallGraph == nil {
			return
		}
		for _, function := range model.CallGraph.Functions {
			_, _ = model.FunctionEffects(function)
		}
	})
}
