package utils

import (
	"fmt"
	"math/rand"
)

var (
	openings = []string{
		"and so it begins",
		"let's get it started",
		"let's do this",
		"do it, do it, do it",
		"work, work",
		"I'll do all the work then",
	}
	closings = []string{
		"bye",
		"the end",
		"go away now",
		"finally",
		"let us know celebrate",
		"time for a beer",
		"well that's one thing you've done today",
	}
)

func a(items []string) string {
	return items[rand.Intn(len(items))]
}

func Opening() string {
	return fmt.Sprintf("%s...", a(openings))
}

func Closing() string {
	return fmt.Sprintf("...%s.\n", a(closings))
}
