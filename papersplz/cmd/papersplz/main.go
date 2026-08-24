package main

import (
	"os"

	"github.com/mhlotto/vibrazioni/papersplz/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, os.LookupEnv))
}
