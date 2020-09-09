package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"log"
)

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
