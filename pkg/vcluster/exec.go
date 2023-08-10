package vcluster

import (
	"fmt"
	"os/exec"
)

// Temporary! Replace with Helm chart deployment.

func Create(name string) error {
	// This command connects by default; without connection, vcluster doesn't create kubectl context
	// Flags checked and removed: "--update-current=true", "--connect=false")
	cmd := exec.Command("vcluster", "create", name, fmt.Sprintf("--kube-config-context-name=%s", name))

	_, err := cmd.Output()
	// fmt.Println(cmd.String())
	// fmt.Println(string(out))
	return err
}

func Delete(name string) error {
	// vcluster disconnect won't work here;
	// probably because we connected "manually"
	cmd := exec.Command("vcluster", "delete", name)

	out, err := cmd.Output()
	fmt.Println(cmd.String())
	fmt.Println(string(out))
	return err
}
