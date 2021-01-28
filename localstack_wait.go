package main

import (
	"fmt"
	u "ork/utils"
	"os"
	"os/exec"
)

func removeLocalStackWait() {
	u.PrintInfo("removing localstack-wait")
	if err := os.Remove("localstack-wait"); err != nil {
		u.PrintError(fmt.Errorf("unable to remove localstack-wait: %w", err))
	}
}

func getLocalStackWait() error {
	cmd := exec.Command("go", "get", "-u", "github.com/bbc/trb/localstack-wait")
	cmd.Env = append(os.Environ(), "GONOSUMDB=github.com/bbc/trb,github.com/bbc/cec")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to get localstack-wait: %v, \noutput:\n%s", err, out)
	}

	cmd = exec.Command("go", "build", "github.com/bbc/trb/localstack-wait")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to build localstack-wait: %v, \noutput:\n%s", err, out)
	}

	return nil
}
