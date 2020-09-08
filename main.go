package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"ork/utils"
	"os"
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

func main() {
	fmt.Println("and so it begins")

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.ExitWithErr(err, "NewEnvClient")

	// create random network to hold it all together
	nw, err := cli.NetworkCreate(ctx, utils.RandomHex(4), types.NetworkCreate{})
	utils.ExitWithErr(err, "NetworkCreate")

	// read env file, to be passed to containers
	envFile, err := utils.ReadEnvFile()
	utils.ExitWithErr(err, "ReadEnvFile")

	localstackContainer, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "docker.io/localstack/localstack", // add docker.io/ to "make it canonical"!
		Env:   envFile,
		Tty:   true,
	}, nil, nil, "")
	utils.ExitWithErr(err, "create localstack container", envFile)

	captureInterrupt(func() { tidyUp(ctx, cli, localstackContainer.ID, nw.ID) })

	if err := cli.NetworkConnect(ctx, nw.ID, localstackContainer.ID, &network.EndpointSettings{Aliases: []string{"localstack"}}); err != nil {
		tidyUp(ctx, cli, localstackContainer.ID, nw.ID)
		log.Fatalln("NetworkConnect error", err)
	}

	if err := cli.ContainerStart(ctx, localstackContainer.ID, types.ContainerStartOptions{}); err != nil {
		tidyUp(ctx, cli, localstackContainer.ID, nw.ID)
		log.Fatalln("NetworkConnect error", err)
	}

	go streamLogs(ctx, cli, localstackContainer.ID)

	// should do wait for it
	fmt.Println("waiting 10 seconds for localstack to start")
	time.Sleep(10 * time.Second)

	// setup
	setupLogs, err := runContainer(ctx, cli, nw, setupContainerConfig(envFile))
	if err != nil {
		tidyUp(ctx, cli, localstackContainer.ID, nw.ID)
		log.Fatalln("Running setup error", err)
	}

	// tests
	testLogs, err := runContainer(ctx, cli, nw, testContainerConfig(envFile))
	if err != nil {
		tidyUp(ctx, cli, localstackContainer.ID, nw.ID)
		log.Fatalln("Running tests error", err)
	}

	tidyUp(ctx, cli, localstackContainer.ID, nw.ID)

	fmt.Println("")
	fmt.Println("")
	fmt.Println("===================================================")
	fmt.Println("====== Setup ======================================")
	fmt.Println("===================================================")
	stdcopy.StdCopy(os.Stdout, os.Stderr, setupLogs)
	fmt.Println("")
	fmt.Println("")

	fmt.Println("===================================================")
	fmt.Println("====== Test =======================================")
	fmt.Println("===================================================")
	stdcopy.StdCopy(os.Stdout, os.Stderr, testLogs)
	fmt.Println("===================================================")
}

func streamLogs(ctx context.Context, cli *client.Client, localstackContainerID string) {
	func() {
		reader, err := cli.ContainerLogs(ctx, localstackContainerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			log.Println("unable to get logs")
			return
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()
}

func captureInterrupt(f func()) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		f()
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
}

func runContainer(ctx context.Context, cli *client.Client, nw types.NetworkCreateResponse, c ContainerHostConfig) (io.ReadCloser, error) {
	cont, err := cli.ContainerCreate(ctx, c.conf, c.hostConf, nil, "")
	if err != nil {
		return nil, fmt.Errorf("error creating container: %w", err)
	}

	if err := cli.NetworkConnect(ctx, nw.ID, cont.ID, nil); err != nil {
		return nil, fmt.Errorf("error connecting to network: %w", err)
	}

	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("error starting container: %w", err)
	}

	if _, err := cli.ContainerWait(ctx, cont.ID); err != nil {
		timeout := 5 * time.Second // give the container 5s to stop by itself
		if err := cli.ContainerStop(ctx, cont.ID, &timeout); err != nil {
			return nil, fmt.Errorf("unable to stop container: %w", err)
		}

		return nil, fmt.Errorf("error waiting for container: %w", err)
	}

	logs, err := cli.ContainerLogs(ctx, cont.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, fmt.Errorf("error getting container logs: %w", err)
	}

	return logs, nil
}
