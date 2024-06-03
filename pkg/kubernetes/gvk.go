package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func kindToGVK(kind string, discoveryClient *discovery.DiscoveryClient) (schema.GroupVersionKind, error) {
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionKind{}, err
	}

	// each item contains a list of resources
	for _, list := range apiResources {
		for _, resource := range list.APIResources {
			if resource.Kind == kind {
				group, version := splitGV(list.GroupVersion)
				return schema.GroupVersionKind{Group: group, Version: version, Kind: kind}, nil
			}
		}
	}
	return schema.GroupVersionKind{}, fmt.Errorf("kind not found")
}

func splitGV(groupVersion string) (string, string) {
	if strings.Contains(groupVersion, "/") {
		s := strings.Split(groupVersion, "/")
		return s[0], s[1]
	}
	return "", groupVersion
}
