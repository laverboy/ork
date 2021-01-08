package main

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"go/build"
)

type ContainerHostConfig struct {
	conf     *container.Config
	hostConf *container.HostConfig
}

func localstackContainerConfig(envConfig []string) ContainerHostConfig {
	return ContainerHostConfig{
		conf: &container.Config{
			Image: "docker.io/localstack/localstack", // add docker.io/ to "make it canonical"!
			Env:   envConfig,
			Tty:   true,
			ExposedPorts: nat.PortSet{
				"4566/tcp": struct{}{},
			},
		},
		hostConf: &container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: "/var/run/docker.sock",
					Target: "/var/run/docker.sock",
				},
			},
			PortBindings: nat.PortMap{
				"4566/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "4566",
					},
				},
			},
		},
	}
}

func setupContainerConfig(envConfig []string) ContainerHostConfig {
	return ContainerHostConfig{
		conf: &container.Config{
			Image:      "docker.io/mesosphere/aws-cli", // add docker.io/ to "make it canonical"!
			Env:        envConfig,
			Cmd:        []string{"setup.sh"},
			WorkingDir: "/usr/src/service",
			Entrypoint: []string{""}, // need this
		},
		hostConf: &container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: fmt.Sprintf("%s/setup.sh", currentDir()),
					Target: "/usr/local/bin/setup.sh",
				},
				{
					Type:   mount.TypeBind,
					Source: currentDir(),
					Target: "/usr/src/service",
				},
			},
		},
	}
}

func testContainerConfig(envConfig []string) ContainerHostConfig {
	return ContainerHostConfig{
		conf: &container.Config{
			Image:      "010894407141.dkr.ecr.eu-west-1.amazonaws.com/build-container/docker-go",
			Env:        envConfig,
			Cmd:        []string{"go", "test"},
			User:       "jenkins",
			WorkingDir: "/usr/src/service",
		},
		hostConf: &container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: fmt.Sprintf("%s/setup.sh", currentDir()),
					Target: "/usr/local/bin/setup.sh",
				},
				{
					Type:   mount.TypeBind,
					Source: currentDir(),
					Target: "/usr/src/service",
				},
				{
					Type:   mount.TypeBind,
					Source: build.Default.GOPATH,
					Target: "/go",
				},
			},
		},
	}
}
