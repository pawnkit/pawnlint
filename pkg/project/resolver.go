package project

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pawnkit/pawn-project/fsx"
	"github.com/pawnkit/pawn-project/include"
	"github.com/pawnkit/pawn-project/pathutil"
)

type includeResolution struct {
	fsys      fsx.FS
	roots     []string
	resolvers map[string]*include.Resolver
}

func newIncludeResolver(sources []Source, options Options) *includeResolution {
	files := make(map[string][]byte, len(sources))
	for _, source := range sources {
		path, err := canonicalPath(source.Path, options.WorkingDir)
		if err == nil {
			slashPath := pathutil.ToSlash(path)
			files[slashPath] = source.Content
		}
	}
	roots := make([]string, 0, len(options.IncludePaths)+1)
	for _, root := range options.IncludePaths {
		roots = append(roots, pathutil.ToSlash(root))
	}
	roots = append(roots, pathutil.ToSlash(options.WorkingDir))

	return &includeResolution{
		fsys:      sourceFS{files: files},
		roots:     roots,
		resolvers: make(map[string]*include.Resolver),
	}
}

func (r *includeResolution) ResolveAll(fromFile, spec string, quoted bool, includeRoot string) []string {
	root := pathutil.ToSlash(includeRoot)
	resolver := r.resolvers[root]
	if resolver == nil {
		resolver = include.NewWithQuotedRoots(r.fsys, r.roots, []string{root})
		r.resolvers[root] = resolver
	}
	return resolver.ResolveAll(pathutil.ToSlash(fromFile), spec, quoted)
}

type sourceFS struct {
	files map[string][]byte
}

func (s sourceFS) Stat(path string) (fs.FileInfo, error) {
	path = pathutil.Clean(path)
	if content, ok := s.files[path]; ok {
		return sourceFileInfo{name: filepath.Base(path), size: int64(len(content))}, nil
	}
	return os.Stat(path)
}

func (s sourceFS) ReadFile(path string) ([]byte, error) {
	path = pathutil.Clean(path)
	if content, ok := s.files[path]; ok {
		return append([]byte(nil), content...), nil
	}
	return os.ReadFile(path) //nolint:gosec // Include roots constrain resolver paths.
}

func (s sourceFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

type sourceFileInfo struct {
	name string
	size int64
}

func (i sourceFileInfo) Name() string     { return i.name }
func (i sourceFileInfo) Size() int64      { return i.size }
func (sourceFileInfo) Mode() fs.FileMode  { return 0o644 }
func (sourceFileInfo) ModTime() time.Time { return time.Time{} }
func (sourceFileInfo) IsDir() bool        { return false }
func (sourceFileInfo) Sys() any           { return nil }

var _ fsx.FS = sourceFS{}
