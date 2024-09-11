package kubernetes

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

// CurrentContext returns the name of currently active
// Kubernetes context.
// configPath is the path to Kubeconfig.
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

// SetContext changes active context to the required one.
// configPath is the path to Kubeconfig. It is modified by this function.
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

// DeleteContext removes the provided context.
// configPath is the path to Kubeconfig. It is modified by this function.
func DeleteContext(configPath, ctxName string) error {
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
	delete(rawCfg.Contexts, ctxName)
	return clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), rawCfg, true)
}
