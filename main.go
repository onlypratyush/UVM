package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"uvm/pkg/golang"
	"uvm/pkg/node"
	"uvm/pkg/python"
)

var version = "0.0.4"

var supportedRuntimes = []string{"node", "go", "python"}

// NewRootCmd creates and configures the root command and all subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "uvm",
		Short: "Universal Version Manager",
		Long: `uvm (Universal Version Manager) is a fast, lightweight CLI tool to install, 
manage, and switch between multiple programming language runtimes (Node.js, Go, Python) and versions seamlessly.`,
		Version:      version,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		newInstallCmd(),
		newUseCmd(),
		newListCmd(),
		newListRemoteCmd(),
		newRemoveCmd(),
		newCurrentCmd(),
	)

	return rootCmd
}

func runtimeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return supportedRuntimes, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func versionCompletion(forInstall bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return supportedRuntimes, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			rt := args[0]
			if forInstall {
				var completions []string
				switch rt {
				case "node", "nodejs":
					completions = []string{"latest", "lts", "24", "22", "20", "18"}
				case "go", "golang":
					completions = []string{"latest", "stable", "1.24", "1.23", "1.22", "1.21"}
				case "python", "py", "python3":
					completions = []string{"latest", "lts", "3.13", "3.12", "3.11", "3.10"}
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			// For use and remove: return locally installed versions
			var installedVers []string
			switch rt {
			case "node", "nodejs":
				mgr := node.NewManager("")
				if list, err := mgr.ListInstalled(); err == nil {
					for _, v := range list {
						installedVers = append(installedVers, v.Version)
					}
				}
			case "go", "golang":
				mgr := golang.NewManager("")
				if list, err := mgr.ListInstalled(); err == nil {
					for _, v := range list {
						installedVers = append(installedVers, v.Version)
					}
				}
			case "python", "py", "python3":
				mgr := python.NewManager("")
				if list, err := mgr.ListInstalled(); err == nil {
					for _, v := range list {
						installedVers = append(installedVers, v.Version)
					}
				}
			}
			return installedVers, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install [runtime] [version]",
		Short:   "Install a specific runtime version (supports exact or prefix like '24')",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm install node 24\n  uvm install node 20.11.0\n  uvm install node lts\n  uvm install go 1.22\n  uvm install python 3.12",
		ValidArgsFunction: versionCompletion(true),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, ver := args[0], args[1]
			switch runtime {
			case "node", "nodejs":
				mgr := node.NewManager("")
				return mgr.Install(ver, cmd.OutOrStdout())
			case "go", "golang":
				mgr := golang.NewManager("")
				return mgr.Install(ver, cmd.OutOrStdout())
			case "python", "py", "python3":
				mgr := python.NewManager("")
				return mgr.Install(ver, cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", runtime)
			}
		},
	}
	return cmd
}

func newUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "use [runtime] [version]",
		Short:   "Switch to a specific installed runtime version (supports partial prefix e.g. '24')",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm use node 24\n  uvm use node 20.11.0\n  uvm use go 1.22\n  uvm use python 3.12",
		ValidArgsFunction: versionCompletion(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, ver := args[0], args[1]
			switch runtime {
			case "node", "nodejs":
				mgr := node.NewManager("")
				return mgr.Use(ver, cmd.OutOrStdout())
			case "go", "golang":
				mgr := golang.NewManager("")
				return mgr.Use(ver, cmd.OutOrStdout())
			case "python", "py", "python3":
				mgr := python.NewManager("")
				return mgr.Use(ver, cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", runtime)
			}
		},
	}
	return cmd
}

func newListCmd() *cobra.Command {
	var remoteFlag bool

	cmd := &cobra.Command{
		Use:     "list [runtime]",
		Aliases: []string{"ls"},
		Short:   "List installed versions for runtimes (use --remote for available remote versions)",
		Args:    cobra.MaximumNArgs(1),
		Example: "  uvm list\n  uvm list node\n  uvm list --remote node\n  uvm list --remote",
		ValidArgsFunction: runtimeCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := ""
			if len(args) > 0 {
				rt = args[0]
			}

			if remoteFlag {
				return printRemoteList(cmd, rt)
			}

			if rt == "" {
				nodeMgr := node.NewManager("")
				nodeList, _ := nodeMgr.ListInstalled()

				goMgr := golang.NewManager("")
				goList, _ := goMgr.ListInstalled()

				pyMgr := python.NewManager("")
				pyList, _ := pyMgr.ListInstalled()

				if len(nodeList) == 0 && len(goList) == 0 && len(pyList) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Listing all managed runtimes and versions...")
					fmt.Fprintln(cmd.OutOrStdout(), "  (No runtime versions installed yet)")
					return nil
				}

				if len(nodeList) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Installed Node.js versions:")
					for _, v := range nodeList {
						if v.IsActive {
							fmt.Fprintf(cmd.OutOrStdout(), "  * %s (active)\n", v.Version)
						} else {
							fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", v.Version)
						}
					}
				}

				if len(goList) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Installed Go versions:")
					for _, v := range goList {
						if v.IsActive {
							fmt.Fprintf(cmd.OutOrStdout(), "  * %s (active)\n", v.Version)
						} else {
							fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", v.Version)
						}
					}
				}

				if len(pyList) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Installed Python versions:")
					for _, v := range pyList {
						if v.IsActive {
							fmt.Fprintf(cmd.OutOrStdout(), "  * %s (active)\n", v.Version)
						} else {
							fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", v.Version)
						}
					}
				}

				return nil
			}

			switch rt {
			case "node", "nodejs":
				mgr := node.NewManager("")
				list, err := mgr.ListInstalled()
				if err != nil {
					return err
				}
				if len(list) == 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "No installed Node.js versions found. Run 'uvm install node <version>' to install one.\n")
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

			case "go", "golang":
				mgr := golang.NewManager("")
				list, err := mgr.ListInstalled()
				if err != nil {
					return err
				}
				if len(list) == 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "No installed Go versions found. Run 'uvm install go <version>' to install one.\n")
					return nil
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Installed Go versions:")
				for _, v := range list {
					if v.IsActive {
						fmt.Fprintf(cmd.OutOrStdout(), "  * %s (active)\n", v.Version)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", v.Version)
					}
				}
				return nil

			case "python", "py", "python3":
				mgr := python.NewManager("")
				list, err := mgr.ListInstalled()
				if err != nil {
					return err
				}
				if len(list) == 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "No installed Python versions found. Run 'uvm install python <version>' to install one.\n")
					return nil
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Installed Python versions:")
				for _, v := range list {
					if v.IsActive {
						fmt.Fprintf(cmd.OutOrStdout(), "  * %s (active)\n", v.Version)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", v.Version)
					}
				}
				return nil

			default:
				return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", rt)
			}
		},
	}

	cmd.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "List available remote versions")
	return cmd
}

func newListRemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-remote [runtime]",
		Aliases: []string{"ls-remote", "list-all", "ls-r"},
		Short:   "List available remote versions for installation",
		Args:    cobra.MaximumNArgs(1),
		Example: "  uvm list-remote\n  uvm list-remote node\n  uvm list-remote go\n  uvm list-remote python",
		ValidArgsFunction: runtimeCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := ""
			if len(args) > 0 {
				rt = args[0]
			}
			return printRemoteList(cmd, rt)
		},
	}
	return cmd
}

func printRemoteList(cmd *cobra.Command, rt string) error {
	if rt == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Available remote versions:")

		nodeMgr := node.NewManager("")
		if releases, err := nodeMgr.ListRemote(10); err == nil && len(releases) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "\nNode.js (latest releases):")
			for _, r := range releases {
				if r.LTS != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-12s (%s)\n", r.Version, r.LTS)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Version)
				}
			}
		}

		goMgr := golang.NewManager("")
		if releases, err := goMgr.ListRemote(10); err == nil && len(releases) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "\nGo (latest stable releases):")
			for _, r := range releases {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Version)
			}
		}

		pyMgr := python.NewManager("")
		if releases, err := pyMgr.ListRemote(10); err == nil && len(releases) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "\nPython (available releases):")
			for _, r := range releases {
				if r.LTS != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-12s (%s)\n", r.Version, r.LTS)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Version)
				}
			}
		}

		return nil
	}

	switch rt {
	case "node", "nodejs":
		mgr := node.NewManager("")
		releases, err := mgr.ListRemote(25)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Available Node.js versions:")
		for _, r := range releases {
			if r.LTS != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s (%s)\n", r.Version, r.LTS)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Version)
			}
		}
		return nil

	case "go", "golang":
		mgr := golang.NewManager("")
		releases, err := mgr.ListRemote(25)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Available Go versions:")
		for _, r := range releases {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Version)
		}
		return nil

	case "python", "py", "python3":
		mgr := python.NewManager("")
		releases, err := mgr.ListRemote(25)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Available Python versions:")
		for _, r := range releases {
			if r.LTS != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s (%s)\n", r.Version, r.LTS)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Version)
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", rt)
	}
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove [runtime] [version]",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove a specific runtime version (supports partial prefix e.g. '24')",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm remove node 24\n  uvm remove node 20.11.0\n  uvm rm go 1.22\n  uvm rm python 3.12",
		ValidArgsFunction: versionCompletion(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, ver := args[0], args[1]
			switch runtime {
			case "node", "nodejs":
				mgr := node.NewManager("")
				return mgr.Remove(ver, cmd.OutOrStdout())
			case "go", "golang":
				mgr := golang.NewManager("")
				return mgr.Remove(ver, cmd.OutOrStdout())
			case "python", "py", "python3":
				mgr := python.NewManager("")
				return mgr.Remove(ver, cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", runtime)
			}
		},
	}
	return cmd
}

func newCurrentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "current [runtime]",
		Short:   "Display currently active runtime version",
		Args:    cobra.MaximumNArgs(1),
		Example: "  uvm current\n  uvm current node\n  uvm current go\n  uvm current python",
		ValidArgsFunction: runtimeCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				nodeVer, nodeErr := node.NewManager("").Current()
				goVer, goErr := golang.NewManager("").Current()
				pyVer, pyErr := python.NewManager("").Current()

				if nodeErr != nil && goErr != nil && pyErr != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "No active runtime versions set.")
					return nil
				}

				if nodeErr == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Node.js: %s\n", nodeVer)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Node.js: (none)")
				}

				if goErr == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Go:      %s\n", goVer)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Go:      (none)")
				}

				if pyErr == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Python:  %s\n", pyVer)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Python:  (none)")
				}

				return nil
			}

			runtime := args[0]
			switch runtime {
			case "node", "nodejs":
				mgr := node.NewManager("")
				ver, err := mgr.Current()
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "No active Node.js version set.")
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Current Node.js version: %s\n", ver)
				return nil

			case "go", "golang":
				mgr := golang.NewManager("")
				ver, err := mgr.Current()
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "No active Go version set.")
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Current Go version: %s\n", ver)
				return nil

			case "python", "py", "python3":
				mgr := python.NewManager("")
				ver, err := mgr.Current()
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "No active Python version set.")
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Current Python version: %s\n", ver)
				return nil

			default:
				return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", runtime)
			}
		},
	}
	return cmd
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
