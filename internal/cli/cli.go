package cli

import (
	"flag"
	"fmt"
	"io"
)

func Run(args []string, version string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("reprobundle", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	repo := fs.String("repo", ".", "repository root to slice from")
	entry := fs.String("entry", "", "entry point: pytest test ID, agent class, or path to a JSONL trace")
	out := fs.String("out", "", "output directory for the bundle")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return nil
	}
	if *entry == "" || *out == "" {
		fs.Usage()
		return fmt.Errorf("--entry and --out are required")
	}
	return fmt.Errorf("not implemented: would slice %q from %q into %q", *entry, *repo, *out)
}
