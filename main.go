package main

import (
	"flag"
	"fmt"
	"math/rand"
	u "ork/utils"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
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

func theStuff() int {
	rand.Seed(time.Now().UnixNano())
	// ensure from now on ctrl-c kills process
	captureInterrupt()

	u.PrintStage(u.Opening())
	defer u.PrintStage(u.Closing())

	if isLocal() {
		if err := u.LoginToECR(); err != nil {
			u.PrintError(err)
			return 1
		}

		if err := buildLambda(); err != nil {
			u.PrintError(err)
			return 1
		}
		defer removeLambda()
	}

	// create random network to hold it all together
	networkName := u.RandomHex(4)

	// create docker network
	if err := createNetwork(networkName); err != nil {
		u.PrintError(err)
		return 2
	}
	defer removeNetwork(networkName)

	if err := getLocalStackWait(); err != nil {
		u.PrintError(err)
		return 2
	}
	defer removeLocalStackWait()

	u.PrintStage("starting localstack...")
	out, err := NewLocalStackDockerCMD(networkName).CombinedOutput()
	if err != nil {
		u.PrintError(fmt.Errorf("unable to get localstack-wait: %v, \noutput:\n%s", err, out))
		return 3
	}
	var localStackContainerID = strings.TrimSpace(string(out))
	defer killLocalStackContainer(localStackContainerID)

	u.PrintStage("waiting for localstack to be ready...")
	out, err = NewWaitForLocalStackDockerCMD(localStackContainerID).CombinedOutput()
	if err != nil {
		u.PrintError(fmt.Errorf("failed to launch localstack: %v, \noutput:\n%s", err, out))
		return 5
	}

	u.PrintStage("running setup...")
	out, err = NewLocalStackSetupDockerCMD(networkName).CombinedOutput()
	if err != nil {
		u.PrintError(fmt.Errorf("failed to setup localstack: %v, \noutput:\n%s", err, out))
		return 7
	}

	u.PrintInfo("localstack setup successfully")
	fmt.Println(string(out))

	u.PrintStage("running test...")
	out, err = NewRunTestDockerCMD(networkName).CombinedOutput()
	if err != nil {
		u.PrintError(fmt.Errorf("test failed or errored: %v, \noutput:\n%s", err, out))
	}

	u.PrintStage("localstack logs...")
	logs, _ := exec.Command("docker", "logs", localStackContainerID).CombinedOutput()
	fmt.Printf("\n%s\n", logs)

	u.PrintStage("test results...")
	fmt.Printf("\n%s\n", out)

	u.PrintStage("tidy up...")
	return 0
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
		u.PrintInfo("Ctrl+C pressed in Terminal")
		runtime.Goexit()
	}()
}
