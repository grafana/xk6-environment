package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"xk6-environment/pkg/fs"

	"github.com/grafana/xk6-kubernetes/pkg/resources"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// This package tries to re-use xk6-kubernetes
// (for now, ignoring all the injections for tests)

func loadConfig(configPath string) (clientcmd.ClientConfig, error) {
	kubeconfig := configPath
	if kubeconfig == "" {
		home := homedir.HomeDir()
		if home == "" {
			return nil, errors.New("home directory not found")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		}), nil
}

func CurrentContext(configPath string) (string, error) {
	// is there a better way to get current context?
	cfg, err := loadConfig(configPath)
	if err != nil {
		return "", err
	}

	rawConfig, err := cfg.RawConfig()
	if err != nil {
		return "", err
	}

	return rawConfig.CurrentContext, nil
}

// updates Kubeconfig
func SetContext(configPath, ctxName string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	rawCfg, err := cfg.RawConfig()
	if err != nil {
		return err
	}

	if rawCfg.Contexts[ctxName] == nil {
		return fmt.Errorf("context %s doesn't exist", ctxName)
	}
	rawCfg.CurrentContext = ctxName
	return clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), rawCfg, true)
}

func getClientConfig(configPath string) (*rest.Config, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return cfg.ClientConfig()
}

type Client struct {
	*resources.Client
	configPath string
	clientset  *k8s.Clientset
}

func NewClient(ctx context.Context, configPath, ctxName string) (*Client, error) {
	if ctxName != "" {
		if err := SetContext(configPath, ctxName); err != nil {
			return nil, err
		}
	}

	restConfig, err := getClientConfig(configPath)
	if err != nil {
		return nil, err
	}

	c, err := resources.NewFromConfig(ctx, restConfig)
	if err != nil {
		return nil, err
	}

	// apparently, mapper is a requirement for k8s ops to work...
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	if err != nil {
		return nil, err
	}

	c.WithMapper(mapper)

	clientset, err := k8s.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &Client{c, configPath, clientset}, nil
}

func (c *Client) Deploy(ke fs.KubernetesEnv) error {
	if ke.IsKustomize() {
		yamls, err := sortResources(ke.KustomizeDir)
		if err != nil {
			return err
		}

		for i := range yamls {
			if err = c.Apply(yamls[i]); err != nil {
				return err
			}
		}

		return nil
	}

	// without kustomize, just apply as is
	for ke.ManifestsLeft() {
		content, err := ke.ReadManifest()
		if err != nil {
			return err
		}

		if err = c.Apply(content); err != nil {
			return err
		}
	}

	return nil
}
