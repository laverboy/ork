package main

import (
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

var PWD string

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

	info("tidy up...")
	return 0
}

func main() {
	PWD = os.Getenv("PWD")
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
