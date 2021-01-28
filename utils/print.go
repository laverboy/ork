package utils

import (
	"fmt"
	"strings"
)

const (
	red    = "31"
	yellow = "33"
	blue   = "34"
)

func PrintStage(msg string) {
	fmt.Println(fmt.Sprintf("\n\033[48;5;220m \033[0m \033[1;%sm%s\033[0m", yellow, strings.ToUpper(msg)))
}

func PrintInfo(msg string) {
	fmt.Println(fmt.Sprintf("\033[1;%sm- %s\033[0m", blue, msg))
}

func PrintError(err error) {
	fmt.Println(fmt.Sprintf("\n\033[1;%sm%s\033[0m", red, err))
}
