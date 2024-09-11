package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Small abstractions to deal with filesystem

func FindEnv(folder string) (*EnvDescription, error) {
	var f = &EnvDescription{}
	if len(folder) == 0 {
		return f, nil
	}

	fsys := os.DirFS(folder)
	manifests := make([]string, 0)

	f.folder = folder

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
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
