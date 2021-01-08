package main

import (
	"errors"
	"fmt"
	"log"
	"ork/utils"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

func currentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return wd
}

func info(msg string) {
	log.Println(fmt.Sprintf("\033[1;33m%s\033[0m", msg))
}

func theStuff() int {
	info("and so it begins...")

	if err := buildLambda(); err != nil {
		fmt.Println(err, "buildLambda")
		return 1
	}
	defer removeLambda()

	// ensure from now on ctrl-c kills process
	captureInterrupt()

	// create random network to hold it all together
	networkName := utils.RandomHex(4)

	// create docker network
	if err := createNetwork(networkName); err != nil {
		fmt.Println(err)
		return 2
	}
	defer removeNetwork(networkName)

	if err := getLocalStackWait(); err != nil {
		fmt.Println(err)
		return 2
	}
	defer removeLocalStackWait()

	info("starting localstack...")
	out, err := NewLocalStackDockerCMD(networkName).CombinedOutput()
	if err != nil {
		fmt.Printf("unable to get localstack-wait: %v, \noutput:\n%s", err, out)
		return 3
	}
	var localStackContainerID = strings.TrimSpace(string(out))
	defer killLocalStackContainer(localStackContainerID)

	info("waiting for localstack to be ready...")
	out, err = NewWaitForLocalStackDockerCMD(localStackContainerID).CombinedOutput()
	if err != nil {
		fmt.Printf("failed to launch localstack: %v, \noutput:\n%s", err, out)
		return 5
	}

	info("running setup...")
	out, err = NewLocalStackSetupDockerCMD(networkName).CombinedOutput()
	if err != nil {
		fmt.Printf("failed to setup localstack: %v, \noutput:\n%s", err, out)
		return 7
	}
	fmt.Printf("localstack setup successfully: \n%s", out)

	info("running test...")
	out, err = NewRunTestDockerCMD(networkName).CombinedOutput()
	if err != nil {
		fmt.Printf("test failed or errored: %v, \noutput:\n%s", err, out)
		return 8
	}

	info("localstack logs...")
	logs, _ := exec.Command("docker", "logs", localStackContainerID).CombinedOutput()
	fmt.Printf("localstack logs: \n%s\n", logs)

	info("test results...")
	fmt.Printf("test ran successfully: \n%s", out)

	return 0
}

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
		"-v", fmt.Sprintf("%s/localstack-wait:/usr/local/bin/localstack-wait", os.Getenv("PWD")),
		"-v", fmt.Sprintf("%s/.localstack:/tmp/localstack", os.Getenv("PWD")),
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"localstack/localstack:0.11.6",
	)
}

func NewLocalStackSetupDockerCMD(networkName string) *exec.Cmd {
	return exec.Command("docker", "run", "--rm",
		"--network", networkName,
		"--env-file", ".env",
		"-v", fmt.Sprintf("%s/setup.sh:/usr/local/bin/setup.sh", os.Getenv("PWD")),
		"-v", fmt.Sprintf("%s/:/usr/src/service", os.Getenv("PWD")),
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

	goDir := fmt.Sprintf("%s/go", os.Getenv("HOME"))
	goCacheDir := fmt.Sprintf("%s/go-cache", os.Getenv("HOME"))

	return exec.Command("docker", "run", "--rm",
		"--network", networkName,
		"--env-file", ".env",
		"--network-alias", "test-runner.local",
		"-v", fmt.Sprintf("%s:/var/lib/jenkins/.ssh/id_rsa", sshDir),
		"-v", fmt.Sprintf("%s:/var/lib/jenkins/.cache/go-build", goCacheDir),
		"-v", fmt.Sprintf("%s:/go", goDir),
		"-v", fmt.Sprintf("%s:/usr/src/service", os.Getenv("PWD")),
		"-w", "/usr/src/service",
		"-u", "jenkins",
		"010894407141.dkr.ecr.eu-west-1.amazonaws.com/build-container/docker-go:latest",
		"go", "test",
	)
}
func killLocalStackContainer(id string) {
	fmt.Println("killing localstack")
	cmd := exec.Command("docker", "container", "kill", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("failed to kill container %s: error: %s\noutput: %s\n", id, err, out)
	}

	cmd = exec.Command("docker", "container", "rm", id)
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("failed to remove container %s: error: %s\noutput: %s\n", id, err, out)
	}
}

func removeLambda() {
	fmt.Println("removing lambda.zip")
	if err := os.Remove("lambda.zip"); err != nil {
		log.Println("unable to remove lambda.zip")
	}
}

func removeLocalStackWait() {
	fmt.Println("removing localstack-wait")
	if err := os.Remove("localstack-wait"); err != nil {
		log.Println("unable to remove localstack-wait")
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

func createNetwork(name string) error {
	cmd := exec.Command("docker", "network", "create", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to create network: %v, \noutput:\n%s", err, out)
	}
	return nil
}

func removeNetwork(name string) {
	fmt.Println("removing network")
	cmd := exec.Command("docker", "network", "rm", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("failed to remove network %s: error: %s\noutput: %s\n", name, err, out)
	}
}

func main() {
	os.Exit(theStuff())
}

func buildLambda() error {
	// if app dir does not exist escape
	if _, err := os.Stat("../app"); os.IsNotExist(err) {
		return errors.New("app dir does not exist")
	}

	fmt.Println("building lambda handler")
	cmd := exec.Command("go", "build", "-o", "handler")
	cmd.Dir = "../app"
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running go build: %w", err)
	}

	// add handler and possibly app/resources and integration-tests/cosmos to zip file
	fmt.Println("zipping")
	files := []string{"../app/handler"}

	if _, err := os.Stat("../app/resources"); !os.IsNotExist(err) {
		files = append(files, "../app/resources")
	}

	if _, err := os.Stat("cosmos"); !os.IsNotExist(err) {
		files = append(files, "cosmos")
	}

	if err := utils.ZipFiles("lambda.zip", files); err != nil {
		return fmt.Errorf("error zipping files: %w", err)
	}
	fmt.Println("zipped")

	return nil
}

func captureInterrupt() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		runtime.Goexit()
	}()
}
