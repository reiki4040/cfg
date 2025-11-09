package main

import (
	"fmt"
	"os"

	"github.com/reiki4040/cfg/cli"
)

func main() {
	// Set the version function for CLI to use
	cli.GetVersionString = GetVersionString

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
