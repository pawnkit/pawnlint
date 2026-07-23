package project

import (
	"errors"
	"path/filepath"

	projectfs "github.com/pawnkit/pawn-project/fsx"
	projectmodel "github.com/pawnkit/pawn-project/project"
	"github.com/pawnkit/pawn-project/workspace"
	"github.com/pawnkit/pawnkit-core/source"
)

// Canonical loads Pawn project paths when a manifest is available.
func Canonical(start string, managedRoots []string) (*projectmodel.Project, error) {
	absolute, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	project, err := projectmodel.Load(source.NewRegistry(), projectfs.OS{}, absolute, projectmodel.Options{
		ManagedIncludeRoots: managedRoots,
	})
	if errors.Is(err, workspace.ErrNotFound) {
		return nil, nil
	}
	return project, err
}

// IncludeRoots returns native paths from a loaded project.
func IncludeRoots(project *projectmodel.Project) []string {
	if project == nil {
		return nil
	}
	roots := project.Paths().IncludeRoots
	result := make([]string, len(roots))
	for index, root := range roots {
		result[index] = filepath.FromSlash(root)
	}
	return result
}
