package main

import (
	"os"

	"dangernoodle.io/terranoodle/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
