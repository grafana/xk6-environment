package fs

import (
	"io/fs"
	"os"
)

type KubernetesEnv struct {
	Manifests []string
	i         int

	kustomizationPresent bool
	KustomizeDir         string

	folder string
}

func (k *KubernetesEnv) ReadManifest() (string, error) {
	if k.i >= len(k.Manifests) {
		return "", nil
	}
	fsys := os.DirFS(k.folder)
	data, err := fs.ReadFile(fsys, k.Manifests[k.i])
	k.i++

	return string(data), err
}

func (k KubernetesEnv) ManifestsLeft() bool {
	return k.i < len(k.Manifests)
}

func (k KubernetesEnv) IsKustomize() bool {
	return k.kustomizationPresent
}

// TODO: add smth like:
// type EnvOnFS interface{
// 	Apply(Deployer)
// }

// type Deployer interface{
// 	Deploy([]byte)
// }
