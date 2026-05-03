package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kaeawc/reprobundle/internal/bundler"
	"github.com/kaeawc/reprobundle/internal/intake"
	"github.com/kaeawc/reprobundle/internal/scanner"
	"github.com/kaeawc/reprobundle/internal/slicer"
)

// Run executes the reprobundle CLI. stdout/stderr are split so callers (and
// tests) can capture each independently. The caller owns argument parsing
// errors via the returned error; flag.ContinueOnError keeps the process from
// calling os.Exit on bad flags.
func Run(args []string, version string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("reprobundle", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	repo := fs.String("repo", ".", "repository root to slice from")
	entry := fs.String("entry", "", "entry point: pytest test ID (path::test, path::Class::test)")
	out := fs.String("out", "", "output directory for the bundle")
	dryRun := fs.Bool("dry-run", false, "print the slice without writing the bundle")

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

	return runSlice(context.Background(), *repo, *entry, *out, *dryRun, stdout)
}

func runSlice(ctx context.Context, repo, entry, out string, dryRun bool, stdout io.Writer) error {
	seed, err := intake.ParsePytestID(entry)
	if err != nil {
		return fmt.Errorf("parse entry: %w", err)
	}
	resolved, absRoot, err := intake.ResolveOnDisk(repo, seed)
	if err != nil {
		return fmt.Errorf("resolve seed: %w", err)
	}

	srcFS := os.DirFS(absRoot)
	resolver := scanner.NewPyResolver(srcFS)
	set, err := slicer.WalkFiles(ctx, resolver, resolved.File)
	if err != nil {
		return fmt.Errorf("walk imports: %w", err)
	}

	fmt.Fprintf(stdout, "repo: %s\n", absRoot)
	fmt.Fprintf(stdout, "entry: %s", resolved.File)
	if resolved.Class != "" {
		fmt.Fprintf(stdout, " (class %s)", resolved.Class)
	}
	fmt.Fprintf(stdout, " :: %s", resolved.Symbol)
	if resolved.Param != "" {
		fmt.Fprintf(stdout, " [%s]", resolved.Param)
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout)

	fmt.Fprintf(stdout, "files (%d):\n", len(set.Files))
	for _, f := range set.Files {
		fmt.Fprintf(stdout, "  %s\n", f)
	}
	if len(set.External) > 0 {
		fmt.Fprintf(stdout, "\nexternal (%d):\n", len(set.External))
		for _, e := range set.External {
			fmt.Fprintf(stdout, "  %s\n", e)
		}
	}

	if dryRun {
		fmt.Fprintf(stdout, "\ndry-run: skipping bundle write to %s\n", out)
		return nil
	}

	bundle := bundler.FromWalk(srcFS, absRoot, resolved, set)
	if err := bundler.Write(out, bundle); err != nil {
		return fmt.Errorf("bundle: %w", err)
	}
	fmt.Fprintf(stdout, "\nwrote bundle to %s\n", out)
	return nil
}
