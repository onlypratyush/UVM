package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"uvm/pkg/node"
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
		newCurrentCmd(),
	)

	return rootCmd
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "install [runtime] [version]",
		Short:   "Install a specific runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm install node 20.11.0\n  uvm install node lts\n  uvm install node latest\n  uvm install go 1.22.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, ver := args[0], args[1]
			if runtime == "node" || runtime == "nodejs" {
				mgr := node.NewManager("")
				return mgr.Install(ver, cmd.OutOrStdout())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installing %s version %s...\n", runtime, ver)
			return nil
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use [runtime] [version]",
		Short:   "Switch to a specific installed runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm use node 20.11.0\n  uvm use node lts\n  uvm use go 1.22.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, ver := args[0], args[1]
			if runtime == "node" || runtime == "nodejs" {
				mgr := node.NewManager("")
				return mgr.Use(ver, cmd.OutOrStdout())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Using %s version %s\n", runtime, ver)
			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "node" || args[0] == "nodejs" {
				mgr := node.NewManager("")
				list, err := mgr.ListInstalled()
				if err != nil {
					return err
				}

				if len(list) == 0 {
					if len(args) > 0 {
						fmt.Fprintf(cmd.OutOrStdout(), "No installed Node.js versions found. Run 'uvm install node <version>' to install one.\n")
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "Listing all managed runtimes and versions...")
						fmt.Fprintln(cmd.OutOrStdout(), "  (No Node.js versions installed yet)")
					}
					return nil
				}

				fmt.Fprintln(cmd.OutOrStdout(), "Installed Node.js versions:")
				for _, v := range list {
					if v.IsActive {
						fmt.Fprintf(cmd.OutOrStdout(), "  * %s (active)\n", v.Version)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", v.Version)
					}
				}
				return nil
			}

			runtime := args[0]
			fmt.Fprintf(cmd.OutOrStdout(), "Listing installed versions for %s...\n", runtime)
			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, ver := args[0], args[1]
			if runtime == "node" || runtime == "nodejs" {
				mgr := node.NewManager("")
				return mgr.Remove(ver, cmd.OutOrStdout())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removing %s version %s...\n", runtime, ver)
			return nil
		},
	}
}

func newCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "current [runtime]",
		Short:   "Display currently active runtime version",
		Args:    cobra.MaximumNArgs(1),
		Example: "  uvm current\n  uvm current node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "node" || args[0] == "nodejs" {
				mgr := node.NewManager("")
				ver, err := mgr.Current()
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "No active Node.js version set.")
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Current Node.js version: %s\n", ver)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Current version for %s: none\n", args[0])
			return nil
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
