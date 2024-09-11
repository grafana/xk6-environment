// Package fs provides small abstractions to deal with filesystem.
package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FindEnv walks the folder and locates Kubernetes manifests
// as an environment description.
func FindEnv(folder string) (*EnvDescription, error) {
	f := &EnvDescription{}
	if len(folder) == 0 {
		return f, nil
	}

	//nolint:forbidigo
	fsys := os.DirFS(folder)
	manifests := make([]string, 0)

	f.folder = folder

	err := fs.WalkDir(fsys, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
			if filepath.Base(path) == "kustomization.yaml" {
				f.kustomizationPresent = true
				relativePath, _ := filepath.Split(path)
				f.KustomizeDir = folder + relativePath
			} else {
				manifests = append(manifests, path)
			}
		}

		return nil
	})

	f.Manifests = manifests

	return f, err
}
