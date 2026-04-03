package main

import (
	"fmt"
	"os"

	"github.com/cheung/conchtalk-dlc/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
