package main

import (
	"os"

	"github.com/spf13/pflag"

	"github.com/nathforge/kubectl-split-yaml/internal/cmd"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-split-yaml", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdSplitYAML(cmd.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
