package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"uvm/pkg/config"
	"uvm/pkg/golang"
	"uvm/pkg/node"
	"uvm/pkg/python"
	"uvm/pkg/scaffold"
)

var version = "0.0.4"

var supportedRuntimes = []string{"node", "go", "python"}

// NewRootCmd creates and configures the root command and all subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "uvm",
		Short: "Universal Version Manager",
		Long: `uvm (Universal Version Manager) is a fast, lightweight CLI tool to install, 
manage, and switch between multiple programming language runtimes (Node.js, Go, Python), 
auto-switch versions via .uvmrc, and scaffold production-ready clean architecture projects.`,
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
		newInitCmd(),
		newPinCmd(),
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
		Short:   "Switch to a specific installed runtime version or auto-switch from .uvmrc",
		Args:    cobra.MaximumNArgs(2),
		Example: "  uvm use                  # Auto-switches using .uvmrc in current directory\n  uvm use node 24          # Switches Node.js to v24.x\n  uvm use go 1.22          # Switches Go to 1.22.x\n  uvm use python 3.12      # Switches Python to 3.12.x",
		ValidArgsFunction: versionCompletion(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Auto-detection mode if no arguments provided
			if len(args) == 0 {
				runtimes, matchedFile, err := config.DetectProjectRuntimes("")
				if err != nil {
					return fmt.Errorf("no runtime specified and no .uvmrc or project configuration found in current directory\nUsage: 'uvm use <runtime> <version>' or run 'uvm pin <runtime> <version>'")
				}

				fmt.Fprintf(cmd.OutOrStdout(), "📁 Found %s -> Auto-switching runtime versions:\n", matchedFile)

				for rt, ver := range runtimes {
					fmt.Fprintf(cmd.OutOrStdout(), "  -> Switching %s to %s...\n", rt, ver)
					switch rt {
					case "node", "nodejs":
						mgr := node.NewManager("")
						if err := mgr.Use(ver, cmd.OutOrStdout()); err != nil {
							return err
						}
					case "go", "golang":
						mgr := golang.NewManager("")
						if err := mgr.Use(ver, cmd.OutOrStdout()); err != nil {
							return err
						}
					case "python", "py", "python3":
						mgr := python.NewManager("")
						if err := mgr.Use(ver, cmd.OutOrStdout()); err != nil {
							return err
						}
					}
				}
				return nil
			}

			if len(args) == 1 {
				return fmt.Errorf("missing version argument for runtime %q (run 'uvm use %s <version>')", args[0], args[0])
			}

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

func newPinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pin [runtime] [version]",
		Short:   "Pin a runtime version in current directory's .uvmrc",
		Args:    cobra.ExactArgs(2),
		Example: "  uvm pin node 20\n  uvm pin go 1.22\n  uvm pin python 3.12",
		ValidArgsFunction: versionCompletion(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, ver := strings.ToLower(args[0]), args[1]
			switch rt {
			case "node", "nodejs":
				rt = "node"
			case "go", "golang":
				rt = "go"
			case "python", "py", "python3":
				rt = "python"
			default:
				return fmt.Errorf("unsupported runtime %q (supported: node, go, python)", rt)
			}

			// Read existing .uvmrc if exists in current dir
			existing, _, _ := config.DetectProjectRuntimes(".")
			if existing == nil {
				existing = make(map[string]string)
			}
			existing[rt] = ver

			if err := config.WriteUvmrc(".", existing); err != nil {
				return fmt.Errorf("failed to write .uvmrc: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Pinned %s %s to .uvmrc in current directory\n", rt, ver)
			return nil
		},
	}
	return cmd
}

func newInitCmd() *cobra.Command {
	var (
		lang        string
		framework   string
		isTS        bool
		includeCRUD bool
	)

	cmd := &cobra.Command{
		Use:     "init [project-name]",
		Aliases: []string{"create", "scaffold", "new"},
		Short:   "Scaffold a production-grade clean architecture project with .uvmrc",
		Args:    cobra.MaximumNArgs(1),
		Example: `  uvm init my-api
  uvm init my-node-api --lang node --framework express --ts --crud
  uvm init my-fastify-api --lang node --framework fastify --ts
  uvm init my-go-api --lang go --framework gin --crud
  uvm init my-chi-api --lang go --framework chi`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := ""
			if len(args) > 0 {
				projectName = args[0]
			}

			// If no arguments and no flags provided -> interactive wizard mode
			if projectName == "" && cmd.Flags().NFlag() == 0 {
				reader := bufio.NewReader(cmd.InOrStdin())

				fmt.Fprint(cmd.OutOrStdout(), "Project Name: ")
				input, _ := reader.ReadString('\n')
				projectName = strings.TrimSpace(input)
				if projectName == "" {
					return fmt.Errorf("project name cannot be empty")
				}

				fmt.Fprintln(cmd.OutOrStdout(), "\nSelect programming language:")
				fmt.Fprintln(cmd.OutOrStdout(), "  1) Node.js (Express, Fastify)")
				fmt.Fprintln(cmd.OutOrStdout(), "  2) Go (Gin, Chi, Fiber)")
				fmt.Fprint(cmd.OutOrStdout(), "Choice [1-2] (default 1): ")
				input, _ = reader.ReadString('\n')
				choice := strings.TrimSpace(input)
				if choice == "2" || strings.HasPrefix(strings.ToLower(choice), "g") {
					lang = "go"
				} else {
					lang = "node"
				}

				if lang == "node" {
					fmt.Fprint(cmd.OutOrStdout(), "Use TypeScript? [Y/n] (default Y): ")
					input, _ = reader.ReadString('\n')
					ans := strings.TrimSpace(strings.ToLower(input))
					if ans == "" || ans == "y" || ans == "yes" {
						isTS = true
					}

					fmt.Fprintln(cmd.OutOrStdout(), "\nSelect Web Framework:")
					fmt.Fprintln(cmd.OutOrStdout(), "  1) Express (Clean Layered Architecture)")
					fmt.Fprintln(cmd.OutOrStdout(), "  2) Fastify (High Performance & Schemas)")
					fmt.Fprint(cmd.OutOrStdout(), "Choice [1-2] (default 1): ")
					input, _ = reader.ReadString('\n')
					choice = strings.TrimSpace(input)
					if choice == "2" || strings.HasPrefix(strings.ToLower(choice), "f") {
						framework = "fastify"
					} else {
						framework = "express"
					}
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "\nSelect Go Web Framework:")
					fmt.Fprintln(cmd.OutOrStdout(), "  1) Gin (Popular, fast & robust)")
					fmt.Fprintln(cmd.OutOrStdout(), "  2) Chi (Lightweight & idiomatic)")
					fmt.Fprintln(cmd.OutOrStdout(), "  3) Fiber (Express-inspired high throughput)")
					fmt.Fprint(cmd.OutOrStdout(), "Choice [1-3] (default 1): ")
					input, _ = reader.ReadString('\n')
					choice = strings.TrimSpace(input)
					if choice == "2" || strings.HasPrefix(strings.ToLower(choice), "c") {
						framework = "chi"
					} else if choice == "3" || strings.HasPrefix(strings.ToLower(choice), "f") {
						framework = "fiber"
					} else {
						framework = "gin"
					}
				}
			}

			if projectName == "" {
				return fmt.Errorf("project name cannot be empty (usage: 'uvm init <project-name>')")
			}

			if lang == "" {
				lang = "node"
			}

			dialect := "js"
			if isTS {
				dialect = "ts"
			}

			if framework == "" {
				if lang == "go" {
					framework = "gin"
				} else {
					framework = "express"
				}
			}

			// Get current active version to pin in .uvmrc
			pinnedVersion := ""
			if lang == "node" {
				if v, err := node.NewManager("").Current(); err == nil {
					pinnedVersion = v
				} else {
					pinnedVersion = "20"
				}
			} else if lang == "go" {
				if v, err := golang.NewManager("").Current(); err == nil {
					pinnedVersion = v
				} else {
					pinnedVersion = "1.22"
				}
			}

			mgr := scaffold.NewManager()
			return mgr.Scaffold(scaffold.ProjectOptions{
				ProjectName: projectName,
				TargetDir:   projectName,
				Language:    lang,
				Dialect:     dialect,
				Framework:   framework,
				IncludeCRUD: includeCRUD,
				Version:     pinnedVersion,
			}, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&lang, "lang", "l", "", "Language runtime: node, go")
	cmd.Flags().StringVarP(&framework, "framework", "f", "", "Web Framework: express, fastify, gin, chi, fiber")
	cmd.Flags().BoolVar(&isTS, "ts", false, "Use TypeScript for Node.js projects")
	cmd.Flags().BoolVar(&includeCRUD, "crud", true, "Include complete working CRUD API endpoints")

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
