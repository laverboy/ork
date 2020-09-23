package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

func waitForIt(seconds int) error {
	timeout := time.After(time.Duration(seconds) * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return errors.New("timed out")
		case <-tick:
			c := http.Client{Timeout: 400 * time.Millisecond}
			resp, err := c.Get("http://localhost:4566/health?reload")
			if err != nil {
				continue
			}
			if resp.StatusCode != http.StatusOK {
				continue
			}
			if isAllRunning(parseServiceStatusResponse(resp.Body)) {
				return nil
			}
		}
	}
}

func isAllRunning(statuses map[string]string) bool {
	for _, status := range statuses {
		if status != "running" {
			return false
		}
	}
	return true
}

func parseServiceStatusResponse(body io.ReadCloser) map[string]string {
	i := struct {
		Services map[string]string
	}{}
	if err := json.NewDecoder(body).Decode(&i); err != nil {
		log.Fatal("unknown body in status response: ", err)
	}

	return i.Services
}
