package main

import (
	"errors"
	"fmt"
	"log"
	"ork/utils"
	"os"
	"os/exec"
)

func removeLambda() {
	fmt.Println("removing lambda.zip")
	if err := os.Remove("lambda.zip"); err != nil {
		log.Println("unable to remove lambda.zip")
	}
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
