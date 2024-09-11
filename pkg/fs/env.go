package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

type EnvDescription struct {
	Manifests []string
	i         int

	kustomizationPresent bool
	KustomizeDir         string

	folder string
}

func (k *EnvDescription) ReadManifest() (string, error) {
	if k.i >= len(k.Manifests) {
		return "", nil
	}
	fsys := os.DirFS(k.folder)
	data, err := fs.ReadFile(fsys, k.Manifests[k.i])
	k.i++

	return string(data), err
}

func (k EnvDescription) ManifestsLeft() bool {
	return k.i < len(k.Manifests)
}

func (k EnvDescription) IsKustomize() bool {
	return k.kustomizationPresent
}

// name of the init folder
func (f EnvDescription) InitFolder() string {
	return filepath.Base(f.folder)
}
