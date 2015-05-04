package main

import (
	"os"

	"github.com/aocsolutions/mongo-automation/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
