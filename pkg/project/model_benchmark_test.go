package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkBuildContextualIncludes(b *testing.B) {
	dir := b.TempDir()
	var root strings.Builder
	for i := 0; i < 25; i++ {
		name := fmt.Sprintf("include_%02d", i)
		path := filepath.Join(dir, name+".inc")
		source := fmt.Sprintf("#define CONTEXT_%02d\nstock Function%02d() {}\n", i, i)
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			b.Fatal(err)
		}
		fmt.Fprintf(&root, "#include \"%s.inc\"\n", name)
	}
	root.WriteString("main() {}\n")
	entry := filepath.Join(dir, "main.pwn")
	source := []byte(root.String())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Build([]Source{{Path: entry, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true}); err != nil {
			b.Fatal(err)
		}
	}
}
