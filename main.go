package main

import (
	"flag"
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

var (
	remote bool
	PWD    string
	HOME   string // for location of go and go cache
)

func isRemote() bool {
	return remote
}

func isLocal() bool {
	return !remote
}

func infoln(msg string) {
	log.Println(fmt.Sprintf("\033[1;33m%s\033[0m", msg))
}

func errorln(err error) {
	log.Println(fmt.Sprintf("\033[1;31m%s\033[0m", err))
}

func theStuff() int {
	infoln("and so it begins...")

	if isLocal() {
		if err := loginToECR(); err != nil {
			errorln(err)
			return 1
		}

		if err := buildLambda(); err != nil {
			errorln(err)
			return 1
		}
		defer removeLambda()
	}

	// ensure from now on ctrl-c kills process
	captureInterrupt()

	// create random network to hold it all together
	networkName := utils.RandomHex(4)

	// create docker network
	if err := createNetwork(networkName); err != nil {
		errorln(err)
		return 2
	}
	defer removeNetwork(networkName)

	if err := getLocalStackWait(); err != nil {
		errorln(err)
		return 2
	}
	defer removeLocalStackWait()

	infoln("starting localstack...")
	out, err := NewLocalStackDockerCMD(networkName).CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("unable to get localstack-wait: %v, \noutput:\n%s", err, out))
		return 3
	}
	var localStackContainerID = strings.TrimSpace(string(out))
	defer killLocalStackContainer(localStackContainerID)

	infoln("waiting for localstack to be ready...")
	out, err = NewWaitForLocalStackDockerCMD(localStackContainerID).CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("failed to launch localstack: %v, \noutput:\n%s", err, out))
		return 5
	}

	infoln("running setup...")
	out, err = NewLocalStackSetupDockerCMD(networkName).CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("failed to setup localstack: %v, \noutput:\n%s", err, out))
		return 7
	}
	fmt.Printf("localstack setup successfully: \n%s", out)

	infoln("running test...")
	out, err = NewRunTestDockerCMD(networkName).CombinedOutput()
	if err != nil {
		errorln(fmt.Errorf("test failed or errored: %v, \noutput:\n%s", err, out))
	}

	infoln("localstack logs...")
	logs, _ := exec.Command("docker", "logs", localStackContainerID).CombinedOutput()
	fmt.Printf("\n%s\n", logs)

	infoln("test results...")
	fmt.Printf("\n%s\n", out)

	infoln("tidy up...")
	return 0
}

func loginToECR() error {
	fmt.Println("logging in to ecr")
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

func main() {
	flag.BoolVar(&remote, "remote", false, "set to remote if running as CI")
	flag.Parse()

	PWD = os.Getenv("PWD")
	HOME = os.Getenv("HOME")
	if isRemote() {
		HOME = "/usr/local"
	}

	os.Exit(theStuff())
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
