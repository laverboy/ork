package main

import (
	"fmt"
	"os"
	"os/exec"
)

func NewWaitForLocalStackDockerCMD(localStackName string) *exec.Cmd {
	return exec.Command("docker", "exec", "-t",
		localStackName,
		"/usr/local/bin/localstack-wait",
	)
}

func NewLocalStackDockerCMD(networkName string) *exec.Cmd {
	return exec.Command("docker", "run",
		"--detach",
		"--network", networkName,
		"--network-alias", "localstack",
		"-e", fmt.Sprintf("LAMBDA_DOCKER_NETWORK=%s", networkName),
		"--env-file", ".env",
		"-v", fmt.Sprintf("%s/localstack-wait:/usr/local/bin/localstack-wait", PWD),
		"-v", fmt.Sprintf("%s/.localstack:/tmp/localstack", PWD),
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"localstack/localstack:0.11.6",
	)
}

func killLocalStackContainer(id string) {
	infoln("killing localstack")
	cmd := exec.Command("docker", "container", "kill", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("failed to kill container %s: error: %w\noutput: %s\n", id, err, out))
	}

	cmd = exec.Command("docker", "container", "rm", id)
	out, err = cmd.CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("failed to remove container %s: error: %w\noutput: %s\n", id, err, out))
	}
}

func NewLocalStackSetupDockerCMD(networkName string) *exec.Cmd {
	return exec.Command("docker", "run", "--rm",
		"--network", networkName,
		"--env-file", ".env",
		"-v", fmt.Sprintf("%s/setup.sh:/usr/local/bin/setup.sh", PWD),
		"-v", fmt.Sprintf("%s/:/usr/src/service", PWD),
		"-w", "/usr/src/service",
		`--entrypoint`, "/usr/local/bin/setup.sh",
		"mesosphere/aws-cli",
	)
}

func NewRunTestDockerCMD(networkName string) *exec.Cmd {
	sshKeyLocation, ok := os.LookupEnv("SSH_KEY_LOCATION")
	var sshDir string
	if ok {
		sshDir = fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
	} else {
		sshDir = sshKeyLocation
	}

	goDir := fmt.Sprintf("%s/go", HOME)
	goCacheDir := fmt.Sprintf("%s/go-cache", HOME)

	return exec.Command("docker", "run", "--rm",
		"--network", networkName,
		"--env-file", ".env",
		"--network-alias", "test-runner.local",
		"-v", fmt.Sprintf("%s:/var/lib/jenkins/.ssh/id_rsa", sshDir),
		"-v", fmt.Sprintf("%s:/var/lib/jenkins/.cache/go-build", goCacheDir),
		"-v", fmt.Sprintf("%s:/go", goDir),
		"-v", fmt.Sprintf("%s:/usr/src/service", PWD),
		"-w", "/usr/src/service",
		"-u", "jenkins",
		"010894407141.dkr.ecr.eu-west-1.amazonaws.com/build-container/docker-go:latest",
		"go", "test",
	)
}
