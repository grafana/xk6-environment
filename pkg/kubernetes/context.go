package kubernetes

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

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

// updates Kubeconfig
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
