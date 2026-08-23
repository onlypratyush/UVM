package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.1"

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:     "uvm",
		Short:   "Universal Version Manager",
		Long: `uvm (Universal Version Manager) is a fast, lightweight CLI tool to install, 
manage, and switch between multiple programming language runtimes and versions seamlessly.`,
		Version: version,
	}

	// 1. uvm install <runtime> <version>
	installCmd := &cobra.Command{
		Use:     "install [runtime] [version]",
		Short:   "Install a specific runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm install node 20.11.0\n  uvm install go 1.22.0\n  uvm install python 3.12.2",
		Run: func(cmd *cobra.Command, args []string) {
			runtime, ver := args[0], args[1]
			fmt.Printf("Installing %s version %s...\n", runtime, ver)
		},
	}

	// 2. uvm use <runtime> <version>
	useCmd := &cobra.Command{
		Use:     "use [runtime] [version]",
		Short:   "Switch to a specific installed runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm use node 20.11.0\n  uvm use go 1.22.0",
		Run: func(cmd *cobra.Command, args []string) {
			runtime, ver := args[0], args[1]
			fmt.Printf("Using %s version %s\n", runtime, ver)
		},
	}

	// 3. uvm list [runtime]
	listCmd := &cobra.Command{
		Use:     "list [runtime]",
		Aliases: []string{"ls"},
		Short:   "List installed versions for runtimes",
		Args:    cobra.MaximumNArgs(1),
		Example: "  uvm list\n  uvm list node",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Listing all managed runtimes and versions...")
			} else {
				runtime := args[0]
				fmt.Printf("Listing installed versions for %s...\n", runtime)
			}
		},
	}

	// 4. uvm remove <runtime> <version>
	removeCmd := &cobra.Command{
		Use:     "remove [runtime] [version]",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove a specific runtime version",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm remove node 20.11.0\n  uvm rm go 1.22.0",
		Run: func(cmd *cobra.Command, args []string) {
			runtime, ver := args[0], args[1]
			fmt.Printf("Removing %s version %s...\n", runtime, ver)
		},
	}

	rootCmd.AddCommand(installCmd, useCmd, listCmd, removeCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

