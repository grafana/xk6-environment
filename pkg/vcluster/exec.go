// Package vcluster provides functionality to manipulate vclusters.
package vcluster

import (
	"fmt"
	"os/exec"
)

// Temporary! Replace with Helm chart deployment.

// Create creates a vcluster with the given name.
func Create(name string) error {
	// This command connects by default; without connection, vcluster doesn't create kubectl context
	// Flags checked and removed: "--update-current=true", "--connect=false")
	cmd := exec.Command("vcluster", "create", name, fmt.Sprintf("--kube-config-context-name=%s", name)) // #nosec G204

	_, err := cmd.Output()
	return err
}

// Delete removes the vcluster with the given name.
func Delete(name string) error {
	// vcluster disconnect won't work here;
	// probably because we connected "manually"
	cmd := exec.Command("vcluster", "delete", name)

	_, err := cmd.Output()
	return err
}
