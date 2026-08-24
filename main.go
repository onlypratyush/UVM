package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.1"

// NewRootCmd creates and configures the root command and all subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "uvm",
		Short: "Universal Version Manager",
		Long: `uvm (Universal Version Manager) is a fast, lightweight CLI tool to install, 
manage, and switch between multiple programming language runtimes and versions seamlessly.`,
		Version:      version,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		newInstallCmd(),
		newUseCmd(),
		newListCmd(),
		newRemoveCmd(),
	)

	return rootCmd
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "install [runtime] [version]",
		Short:   "Install a specific runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm install node 20.11.0\n  uvm install go 1.22.0\n  uvm install python 3.12.2",
		Run: func(cmd *cobra.Command, args []string) {
			runtime, ver := args[0], args[1]
			fmt.Fprintf(cmd.OutOrStdout(), "Installing %s version %s...\n", runtime, ver)
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use [runtime] [version]",
		Short:   "Switch to a specific installed runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm use node 20.11.0\n  uvm use go 1.22.0",
		Run: func(cmd *cobra.Command, args []string) {
			runtime, ver := args[0], args[1]
			fmt.Fprintf(cmd.OutOrStdout(), "Using %s version %s\n", runtime, ver)
		},
	}
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [runtime]",
		Aliases: []string{"ls"},
		Short:   "List installed versions for runtimes",
		Args:    cobra.MaximumNArgs(1),
		Example: "  uvm list\n  uvm list node",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Listing all managed runtimes and versions...")
			} else {
				runtime := args[0]
				fmt.Fprintf(cmd.OutOrStdout(), "Listing installed versions for %s...\n", runtime)
			}
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove [runtime] [version]",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove a specific runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm remove node 20.11.0\n  uvm rm go 1.22.0",
		Run: func(cmd *cobra.Command, args []string) {
			runtime, ver := args[0], args[1]
			fmt.Fprintf(cmd.OutOrStdout(), "Removing %s version %s...\n", runtime, ver)
		},
	}
}

// Execute runs the CLI application with given arguments and streams output to out/err writers.
func Execute(args []string, outStream, errStream io.Writer) error {
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	if outStream != nil {
		cmd.SetOut(outStream)
	}
	if errStream != nil {
		cmd.SetErr(errStream)
	}
	return cmd.Execute()
}

func run() error {
	return Execute(os.Args[1:], os.Stdout, os.Stderr)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
