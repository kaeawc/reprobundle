package main

import (
	"fmt"
	"os"

	"github.com/kaeawc/reprobundle/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Run(os.Args[1:], version, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "reprobundle:", err)
		os.Exit(1)
	}
}
