// Package vcluster provides functionality to manipulate vclusters.
package vcluster

import (
	"context"
	"fmt"
	"os/exec"
)

// Temporary! Replace with Helm chart deployment.

// Create creates a vcluster with the given name.
func Create(ctx context.Context, name string) error {
	// This command connects by default; without connection, vcluster doesn't create kubectl context
	// Flags checked and removed: "--update-current=true", "--connect=false")
	contextFlag := fmt.Sprintf("--kube-config-context-name=%s", name)
	cmd := exec.CommandContext(ctx, "vcluster", "create", name, contextFlag) // #nosec G204

	_, err := cmd.Output()
	return err
}

// Delete removes the vcluster with the given name.
func Delete(ctx context.Context, name string) error {
	// vcluster disconnect won't work here;
	// probably because we connected "manually"
	cmd := exec.CommandContext(ctx, "vcluster", "delete", name) // #nosec G204

	_, err := cmd.Output()
	return err
}
