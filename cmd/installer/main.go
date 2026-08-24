package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"uvm/pkg/installer"
)

// RunApp runs the installer application logic with given arguments and streams.
func RunApp(args []string, in io.Reader, out, errOut io.Writer) error {
	fs := flag.NewFlagSet("uvm-installer", flag.ContinueOnError)
	fs.SetOutput(errOut)

	webMode := fs.Bool("web", false, "Launch visual Web GUI installer in browser")
	port := fs.Int("port", 8484, "Port for web UI server")
	silent := fs.Bool("silent", false, "Run unattended installation with default options")
	installDir := fs.String("dir", "", "Custom installation directory")
	noPath := fs.Bool("no-path", false, "Do not modify shell profile PATH variable")
	uninstall := fs.Bool("uninstall", false, "Uninstall uvm and clean up binary")
	help := fs.Bool("help", false, "Show help message")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *help {
		fmt.Fprintln(out, "uvm Visual Installer")
		fmt.Fprintln(out, "Usage: uvm-installer [options]")
		fmt.Fprintln(out, "Options:")
		fs.SetOutput(out)
		fs.PrintDefaults()
		return nil
	}

	opts := installer.Options{
		InstallDir: *installDir,
		ModifyPath: !*noPath,
		Uninstall:  *uninstall,
		Silent:     *silent,
		WebMode:    *webMode,
		Port:       *port,
	}

	if *webMode {
		return installer.StartWebUI(opts, "", runtime.GOOS)
	}

	return installer.RunVisualCLI(opts, in, out, "", runtime.GOOS)
}

func run() error {
	return RunApp(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
