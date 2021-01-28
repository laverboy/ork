package main

import (
	"errors"
	"fmt"
	u "ork/utils"
	"os"
	"os/exec"
)

func removeLambda() {
	u.PrintInfo("removing lambda.zip")
	if err := os.Remove("lambda.zip"); err != nil {
		u.PrintError(fmt.Errorf("unable to remove lambda.zip: %w", err))
	}
}

func buildLambda() error {
	u.PrintInfo("building lambda handler")

	// if app dir does not exist escape
	if _, err := os.Stat("../app"); os.IsNotExist(err) {
		return errors.New("app dir does not exist")
	}

	cmd := exec.Command("go", "build", "-o", "handler")
	cmd.Dir = "../app"
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running go build: %w", err)
	}

	// add handler and possibly app/resources and integration-tests/cosmos to zip file
	u.PrintInfo("zipping")
	files := []string{"../app/handler"}

	if _, err := os.Stat("../app/resources"); !os.IsNotExist(err) {
		files = append(files, "../app/resources")
	}

	if _, err := os.Stat("cosmos"); !os.IsNotExist(err) {
		files = append(files, "cosmos")
	}

	if err := u.ZipFiles("lambda.zip", files); err != nil {
		return fmt.Errorf("error zipping files: %w", err)
	}
	u.PrintInfo("zipped")

	return nil
}
