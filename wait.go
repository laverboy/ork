package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func waitForIt() error {
	timeout := time.After(15 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return errors.New("timed out")
		case <-tick:
			c := http.Client{Timeout: 400 * time.Millisecond}
			resp, err := c.Get("http://localhost:4566/status")
			if err != nil {
				// this is normal, because the web host takes time to start
				fmt.Println("unable to talk to localstack yet")
				continue
			}
			bytes, _ := ioutil.ReadAll(resp.Body)
			if string(bytes) == `{"status": "running"}` {
				fmt.Println("localstack ready")
				return nil
			}
		}
	}
}
