package main

import (
	"fmt"
	"os"
)

func main() {
	err := makeSubcommand().Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
