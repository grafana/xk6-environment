package kubernetes

import (
	"context"
	"errors"
	"path/filepath"

	"xk6-environment/pkg/fs"

	"github.com/grafana/xk6-kubernetes/pkg/resources"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
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

func getClientConfig(configPath string) (*rest.Config, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return cfg.ClientConfig()
}

type Client struct {
	*resources.Client
	discoveryClient *discovery.DiscoveryClient
	configPath      string
	restConfig      *rest.Config
	clientset       *k8s.Clientset
	dynamicClient   *dynamic.DynamicClient
	restMapper      *restmapper.DeferredDiscoveryRESTMapper
}

func NewClient(ctx context.Context, configPath string) (client *Client, err error) {
	client = &Client{
		configPath: configPath,
	}
	client.restConfig, err = getClientConfig(configPath)
	if err != nil {
		return nil, err
	}

	c, err := resources.NewFromConfig(ctx, client.restConfig)
	if err != nil {
		return nil, err
	}
	client.Client = c

	client.discoveryClient, err = discovery.NewDiscoveryClientForConfig(client.restConfig)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(client.discoveryClient))
	if err != nil {
		return nil, err
	}

	client.Client.WithMapper(mapper)
	client.restMapper = mapper

	client.clientset, err = k8s.NewForConfig(client.restConfig)
	if err != nil {
		return nil, err
	}

	client.dynamicClient, err = dynamic.NewForConfig(client.restConfig)
	return
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
