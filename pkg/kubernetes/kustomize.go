package kubernetes

import (
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func sortResources(path string) ([]string, error) {
	// kustomization file must already be in the path!
	// TODO figure out how to do it smarter

	opts := krusty.Options{
		Reorder:      krusty.ReorderOptionLegacy,
		PluginConfig: &types.PluginConfig{},
	}
	k := krusty.MakeKustomizer(&opts)
	fsys := filesys.MakeFsOnDisk()

	resmap, err := k.Run(fsys, path) // <- ordering happens here
	if err != nil {
		return nil, err
	}

	resources := resmap.Resources()
	yamls := make([]string, len(resources))
	for i, resource := range resources {
		d, err := resource.AsYAML()
		if err != nil {
			return nil, err
		}
		yamls[i] = string(d)
	}

	return yamls, nil
}
