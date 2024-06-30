package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	ini "gopkg.in/ini.v1"
)

// Small abstractions to deal with filesystem

type Test struct {
	Kubernetes KubernetesEnv

	Def     TestDef
	Options string

	folder string
}

func FindTest(folder string) (*Test, error) {
	var f = &Test{}
	if len(folder) == 0 {
		return f, fmt.Errorf("no init folder specified")
	}

	fsys := os.DirFS(folder)
	manifests := make([]string, 0)

	// TODO figure out how to remove duplicating
	f.folder = folder
	f.Def.folder = folder
	f.Kubernetes.folder = folder

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ext := filepath.Ext(path)

		// in the root of the folder
		if d.Name() == path {
			if ext == ".js" || ext == ".tar" {
				if len(f.Def.Location) > 0 {
					return errors.New(fmt.Sprintf("Invalid setup: there are two entrypoints; %s and %s - which one to use?", f.Def.Location, path))
				}
				f.Def.Location = path
				f.Def.Type = k6Standalone
			}

			if ext == ".yaml" {
				if len(f.Def.Location) > 0 {
					return errors.New(fmt.Sprintf("Invalid setup: there are two entrypoints; %s and %s - which one to use?", f.Def.Location, path))
				}
				f.Def.Location = path
				f.Def.Type = k6Operator
			}

			if ext == ".ini" {
				f.Options = path
			}
		} else {
			// manifests are in subtree of the folder
			if ext == ".yaml" {
				if filepath.Base(path) == "kustomization.yaml" {
					f.Kubernetes.kustomizationPresent = true
					relativePath, _ := filepath.Split(path)
					f.Kubernetes.KustomizeDir = folder + relativePath
				} else {
					manifests = append(manifests, path)
				}
			}
		}

		return nil
	})

	f.Kubernetes.Manifests = manifests

	return f, err
}

// name of the folder where test is located without all the previous ones
func (f Test) FolderName() string {
	return filepath.Base(f.folder)
}

func (f *Test) ReadOptions() error {
	var opts = defaultK6Opts

	if len(f.Options) > 0 {
		cfg, err := ini.Load(f.folder + "/" + f.Options)
		if err != nil {
			return err
		}

		if v := cfg.Section("k6").Key("version").String(); len(v) > 0 {
			opts.Version = v
		}

		opts.Command = cfg.Section("k6").Key("command").
			In("run", []string{"run", "cloud"})
		opts.Arguments = cfg.Section("k6").Key("arguments").String()
		// opts.Envvars = cfg.Section("k6").Key("envvars").String()
	}

	f.Def.Opts = opts
	return nil
}
