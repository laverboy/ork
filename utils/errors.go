package utils

import "log"

func ExitWithErr(err error, message ...interface{}) {
	if err != nil {
		log.Fatalln(err, message)
	}
}
