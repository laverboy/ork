package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"ork/utils"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
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
		utils.ExitWithErr(err, "buildLambda")
	}

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.ExitWithErr(err, "NewEnvClient")

	// create random network to hold it all together
	networkName := utils.RandomHex(4)
	nw, err := cli.NetworkCreate(ctx, networkName, types.NetworkCreate{})
	utils.ExitWithErr(err, "NetworkCreate")

	// read env file, to be passed to containers
	envFile, err := utils.ReadEnvFile()
	utils.ExitWithErr(err, "ReadEnvFile")

	envFile = append(envFile, fmt.Sprintf("LAMBDA_DOCKER_NETWORK=%s", networkName))

	// create main localstack container
	c := localstackContainerConfig(envFile)
	localstackContainer, err := cli.ContainerCreate(ctx, c.conf, c.hostConf, nil, "")
	utils.ExitWithErr(err, "create localstack container", envFile)

	defer tidyUp(ctx, cli, localstackContainer.ID, nw.ID)

	// ensure from now on ctrl-c kills localstack
	captureInterrupt()

	// connect localstack to container
	if err := cli.NetworkConnect(ctx, nw.ID, localstackContainer.ID, &network.EndpointSettings{Aliases: []string{"localstack"}}); err != nil {
		log.Fatalln("NetworkConnect error", err)
	}

	// start localstack
	if err := cli.ContainerStart(ctx, localstackContainer.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalln("NetworkConnect error", err)
	}
	go streamLogs(ctx, cli, localstackContainer.ID)

	// wait for localstack to be ready
	info("waiting upto 30s for localstack to start")
	if err := waitForIt(30); err != nil {
		log.Fatalln("Localstack did not start in time", err)
	}

	info("running setup...")
	setupLogs, err, _ := runContainer(ctx, cli, nw, setupContainerConfig(envFile))
	if err != nil {
		log.Fatalln("Running setup error", err)
	}

	info("running tests...")
	testLogs, err, statusCode := runContainer(ctx, cli, nw, testContainerConfig(envFile))
	if err != nil {
		log.Fatalln("Running tests error", err)
	}

	if _, err := cli.ContainerWait(ctx, localstackContainer.ID); err != nil {
		timeout := 5 * time.Second // give the container 5s to stop by itself
		if err := cli.ContainerStop(ctx, localstackContainer.ID, &timeout); err != nil {
			log.Fatalf("unable to stop localstack container: %v\n", err)
		}

		log.Fatalf("error waiting for localstack container: %v\n", err)
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println("==========================================================================================")
	fmt.Println("====== Setup =============================================================================")
	fmt.Println("==========================================================================================")
	stdcopy.StdCopy(os.Stdout, os.Stderr, setupLogs)
	fmt.Println("")
	fmt.Println("")

	fmt.Println("==========================================================================================")
	fmt.Println("====== Test ==============================================================================")
	fmt.Println("==========================================================================================")
	stdcopy.StdCopy(os.Stdout, os.Stderr, testLogs)
	fmt.Println("==========================================================================================")

	return statusCode
}

func main() {
	os.Exit(theStuff())
}

func buildLambda() error {
	// if app dir does not exist escape
	if _, err := os.Stat("../app"); os.IsNotExist(err) {
		return errors.New("app dir does not exist")
	}

	home, _ := os.UserHomeDir()

	fmt.Println("building lambda handler")
	cmd := exec.Command("go", "build", "-o", "handler")
	cmd.Dir = "../app"
	cmd.Env = []string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "HOME=" + home}
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
		os.Exit(1)
	}()
}

func tidyUp(ctx context.Context, cli *client.Client, localstackContainerID, networkID string) {
	timeout := 5 * time.Second // give the container 5s to stop by itself
	if err := cli.ContainerStop(ctx, localstackContainerID, &timeout); err != nil {
		log.Println("unable to stop localstack container", err)
	}
	if err := cli.NetworkRemove(ctx, networkID); err != nil {
		log.Println("unable to stop network", err)
	}
	if err := os.Remove("lambda.zip"); err != nil {
		log.Println("unable to remove lambda zip")
	}
	if err := os.Remove("../app/handler"); err != nil {
		log.Println("unable to remove handler")
	}
}

func runContainer(ctx context.Context, cli *client.Client, nw types.NetworkCreateResponse, c ContainerHostConfig) (io.ReadCloser, error, int) {
	cont, err := cli.ContainerCreate(ctx, c.conf, c.hostConf, nil, "")
	if err != nil {
		return nil, fmt.Errorf("error creating container: %w", err), 0
	}

	if err := cli.NetworkConnect(ctx, nw.ID, cont.ID, nil); err != nil {
		return nil, fmt.Errorf("error connecting to network: %w", err), 0
	}

	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("error starting container: %w", err), 0
	}

	status, err := cli.ContainerWait(ctx, cont.ID)
	if err != nil {
		timeout := 5 * time.Second // give the container 5s to stop by itself
		if err := cli.ContainerStop(ctx, cont.ID, &timeout); err != nil {
			return nil, fmt.Errorf("unable to stop container: %w", err), 0
		}

		return nil, fmt.Errorf("error waiting for container: %w", err), 0
	}

	logs, err := cli.ContainerLogs(ctx, cont.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, fmt.Errorf("error getting container logs: %w", err), 0
	}

	return logs, nil, int(status)
}
