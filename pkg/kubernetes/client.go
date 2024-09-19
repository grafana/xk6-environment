// Package kubernetes provides the functionality to access and
// manipulate Kubernetes objects.
package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/grafana/xk6-environment/pkg/fs"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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

// Client encapsulates the key structures of Kubernetes libraries
// that are used to access Kubernetes.
type Client struct {
	discoveryClient *discovery.DiscoveryClient
	configPath      string
	restConfig      *rest.Config
	clientset       *k8s.Clientset
	dynamicClient   *dynamic.DynamicClient
	restMapper      *restmapper.DeferredDiscoveryRESTMapper
	crClient        crclient.Client
}

// NewClient constructs the Client.
func NewClient(_ context.Context, configPath string) (*Client, error) {
	var (
		client = &Client{
			configPath: configPath,
		}
		err error
	)
	client.restConfig, err = getClientConfig(configPath)
	if err != nil {
		return nil, err
	}

	client.discoveryClient, err = discovery.NewDiscoveryClientForConfig(client.restConfig)
	if err != nil {
		return nil, err
	}
	client.restMapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(client.discoveryClient))

	// TODO: this should be suppressing this warning:
	// `[controller-runtime] log.SetLogger(...) was never called; logs will not be displayed.`
	// src: https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/client#New
	// Possibly related: breaking changes in controller-runtime logger happened here PR 2317:
	// https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.15.0
	client.restConfig.WarningHandler = rest.NoWarnings{}
	client.crClient, err = crclient.New(client.restConfig, crclient.Options{
		Cache: nil,
	})
	if err != nil {
		return nil, err
	}

	client.clientset, err = k8s.NewForConfig(client.restConfig)
	if err != nil {
		return nil, err
	}

	client.dynamicClient, err = dynamic.NewForConfig(client.restConfig)
	return client, err
}

// Deploy deploys the initial environment as described in envDesc, taking
// care of kustomize specifics if needed.
func (c *Client) Deploy(ctx context.Context, envDesc *fs.EnvDescription) error {
	if envDesc.IsKustomize() {
		yamls, err := sortResources(envDesc.KustomizeDir)
		if err != nil {
			return err
		}

		for i := range yamls {
			if err = c.Apply(ctx, bytes.NewBufferString(yamls[i])); err != nil {
				return err
			}
		}

		return nil
	}

	// without kustomize, just apply as is
	for envDesc.ManifestsLeft() {
		content, err := envDesc.ReadManifest()
		if err != nil {
			return err
		}

		if err = c.Apply(ctx, bytes.NewBufferString(content)); err != nil {
			return err
		}
	}

	return nil
}

// Apply deploys the manifest in data.
func (c *Client) Apply(ctx context.Context, data *bytes.Buffer) error {
	d := yaml.NewYAMLOrJSONDecoder(data, 4096)

	var (
		gvk *schema.GroupVersionKind
		obj runtime.Object
		ext runtime.RawExtension
		err error
	)
	if err = d.Decode(&ext); err != nil {
		return err
	}

	if _, gvk, err = unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, obj); err != nil {
		return err
	}

	var unstructObj unstructured.Unstructured
	unstructObj.Object = make(map[string]interface{})
	var blob interface{}
	if err = json.Unmarshal(ext.Raw, &blob); err != nil {
		return err
	}
	m, ok := blob.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to extract map[string]interface{} from the object during apply")
	}
	unstructObj.Object = m

	mapper, err := c.crClient.RESTMapper().RESTMapping(gvk.GroupKind())
	if err != nil {
		return err
	}

	// namespaced object should not have empty namespace
	if mapper.Scope.Name() == meta.RESTScopeNameNamespace && len(unstructObj.GetNamespace()) == 0 {
		unstructObj.SetNamespace("default")
	}

	// server side apply
	// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#example-Client-Apply
	return c.crClient.Patch(
		ctx,
		&unstructObj,
		crclient.Apply,
		crclient.ForceOwnership,
		crclient.FieldOwner("xk6-environment"))
}

// GetN is a hopefully temporary substitute for Get
func (c *Client) GetN(ctx context.Context, namespace string, opts *metav1.ListOptions) (int, error) {
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, *opts)
	return len(podList.Items), err
}
