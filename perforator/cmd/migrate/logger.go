package main

import "log"

type logger struct {
}

func (*logger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Verbose implements migrate.Logger.
func (*logger) Verbose() bool {
	return true
}
