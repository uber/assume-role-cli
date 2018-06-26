package main

import (
	"os"

	"github.com/uber/assume-role/cli"
)

func main() {
	os.Exit(cli.Main(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}
