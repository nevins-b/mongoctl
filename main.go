package main

import (
	"os"

	"github.com/aocsolutions/mongoctl/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
