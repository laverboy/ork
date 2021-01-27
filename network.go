package main

import (
	"fmt"
	"os/exec"
)

func createNetwork(name string) error {
	cmd := exec.Command("docker", "network", "create", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to create network: %v, \noutput:\n%s", err, out)
	}
	return nil
}

func removeNetwork(name string) {
	infoln("removing network")
	cmd := exec.Command("docker", "network", "rm", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("failed to remove network %s: error: %w\noutput: %s\n", name, err, out))
	}
}
