package main

import (
	"os"

	"dangernoodle.io/terra-tools/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
