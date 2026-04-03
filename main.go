package main

import (
	"fmt"
	"os"

	"github.com/snana7mi/conchtalk-dlc/cmd"
)

// Set via -ldflags at build time
var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
