package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	configDir string
	quiet     bool
)

func main() {
	// MC_CONFIG_DIR env var mirrors the upstream mc behaviour.
	defaultConfigDir := os.Getenv("MC_CONFIG_DIR")
	if defaultConfigDir == "" {
		homeDir, _ := os.UserHomeDir()
		defaultConfigDir = filepath.Join(homeDir, ".mc")
	}

	flag.StringVar(&configDir, "config-dir", defaultConfigDir, "path to configuration folder")
	flag.BoolVar(&quiet, "quiet", false, "suppress output")
	flag.BoolVar(&quiet, "q", false, "suppress output")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: mc [--config-dir DIR] [--quiet] <command> [arguments]")
		fmt.Fprintln(os.Stderr, "Commands: alias, ls, stat, mb, rb, mirror, anonymous, admin")
		os.Exit(1)
	}

	switch args[0] {
	case "alias":
		cmdAlias(args[1:])
	case "ls":
		cmdLs(args[1:])
	case "stat":
		cmdStat(args[1:])
	case "mb":
		cmdMb(args[1:])
	case "rb":
		cmdRb(args[1:])
	case "mirror":
		cmdMirror(args[1:])
	case "anonymous":
		cmdAnonymous(args[1:])
	case "admin":
		cmdAdmin(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func logf(format string, args ...any) {
	if !quiet {
		fmt.Printf(format, args...)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
