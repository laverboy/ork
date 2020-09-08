package utils

import (
	"bufio"
	"os"
)

func ReadEnvFile() ([]string, error) {
	var lines []string
	file, err := os.Open(".env")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() != "" {
			lines = append(lines, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
