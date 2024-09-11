package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// EnvDescription holds key data that describes the
// Kubernetes environment.
type EnvDescription struct {
	Manifests    []string
	KustomizeDir string

	kustomizationPresent bool
	folder               string
	i                    int
}

// ReadManifest reads one manifest file and returns it.
// The pointer to the "next" manifest in the list is being
// stored internally.
func (ed *EnvDescription) ReadManifest() (string, error) {
	if ed.i >= len(ed.Manifests) {
		return "", nil
	}
	//nolint:forbidigo
	fsys := os.DirFS(ed.folder)
	data, err := fs.ReadFile(fsys, ed.Manifests[ed.i])
	ed.i++

	return string(data), err
}

// ManifestsLeft indicates whether all manifests were read,
// judging by internal pointer.
func (ed EnvDescription) ManifestsLeft() bool {
	return ed.i < len(ed.Manifests)
}

// IsKustomize indicates whether this environment has
// kustomization.yaml file.
func (ed EnvDescription) IsKustomize() bool {
	return ed.kustomizationPresent
}

// InitFolder returns the name of the init folder, given
// during configuration of this environment.
func (ed EnvDescription) InitFolder() string {
	return filepath.Base(ed.folder)
}
