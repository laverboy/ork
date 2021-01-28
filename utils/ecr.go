package utils

import (
	"fmt"
	"os/exec"
)

func LoginToECR() error {
	PrintInfo("logging in to ecr")
	cmd := exec.Command("aws", "ecr", "get-login", "--no-include-email", "--region", "eu-west-1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to get ecr login: error: %s\noutput: %s\nuse: aws-creds trb-prod\n", err, out)
	}

	cmd = exec.Command("bash", "-c", string(out))
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to login to ecr: error: %s\noutput: %s\n", err, out)
	}

	return nil
}
