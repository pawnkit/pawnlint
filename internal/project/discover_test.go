package project_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pawnkit/pawnlint/internal/project"
)

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pat, path string
		want      bool
	}{
		{"gamemodes/**/*.pwn", "gamemodes/main.pwn", true},
		{"gamemodes/**/*.pwn", "gamemodes/sub/main.pwn", true},
		{"gamemodes/**/*.pwn", "includes/x.inc", false},
		{"vendor/**", "vendor/a/b.inc", true},
		{"vendor/**", "vendor/a/b.pwn", true},
		{"*.pwn", "main.pwn", true},
		{"*.pwn", "gamemodes/main.pwn", false},
		{"includes/**/*.inc", "includes/a/b/c.inc", true},
		{"**", "anything/here.inc", true},
		{"a/?/c", "a/b/c", true},
		{"a/?/c", "a/bb/c", false},
		{"*", "a", true},
		{"*", "a/b", false},
		{"**/*.pwn", "x/y/z.pwn", true},
		{"**/*.pwn", "z.pwn", true},
		{"generated/**", "generated/foo/bar.inc", true},
		{"dependencies/**", "dependencies/x.pwn", true},
	}
	for _, c := range cases {
		got := project.MatchGlob(c.pat, c.path)
		if got != c.want {
			t.Errorf("MatchGlob(%q,%q): got %v want %v", c.pat, c.path, got, c.want)
		}
	}
}

func TestDiscoverDir(t *testing.T) {
	root := t.TempDir()
	mk := func(rel string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("main(){}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("gamemodes/main.pwn")
	mk("gamemodes/sub/extra.pwn")
	mk("includes/util.inc")
	mk("vendor/v.pwn")
	mk("notes.txt")

	files, err := project.Discover(project.Options{
		Roots:      []string{root},
		WorkingDir: root,
		Include:    []string{"gamemodes/**/*.pwn", "includes/**/*.inc"},
		Exclude:    []string{"vendor/**"},
	})
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		rel, _ := filepath.Rel(root, f.Path)
		names = append(names, rel)
	}
	sort.Strings(names)
	want := []string{"gamemodes/main.pwn", "gamemodes/sub/extra.pwn", "includes/util.inc"}
	if len(names) != len(want) {
		t.Fatalf("got %v want %v", names, want)
	}
	for i := range names {
		if names[i] != want[i] {
			t.Errorf("got %v want %v", names, want)
			break
		}
	}
}

func TestDiscoverExplicitFile(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "x.pwn")
	if err := os.WriteFile(p, []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := project.Discover(project.Options{Roots: []string{p}, WorkingDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || string(files[0].Content) != "y\n" {
		t.Fatalf("got %+v", files)
	}
}

func TestDiscoverExplicitFileOverridesExclude(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "vendor", "x.pwn")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := project.Discover(project.Options{
		Roots:      []string{p},
		WorkingDir: root,
		Exclude:    []string{"vendor/**"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("explicit file was excluded: %+v", files)
	}
}
