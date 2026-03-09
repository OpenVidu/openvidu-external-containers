package main

import (
	"context"
	"fmt"
	"os"
)

func cmdAlias(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: alias <set|ls|remove> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		cmdAliasSet(args[1:])
	case "ls", "list":
		cmdAliasList()
	case "remove", "rm":
		cmdAliasRemove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown alias subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdAliasSet(args []string) {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: alias set <name> <url> <access-key> <secret-key>")
		os.Exit(1)
	}
	name, rawURL, accessKey, secretKey := args[0], args[1], args[2], args[3]

	// Probe the server — matches upstream mc behaviour where alias set fails
	// when the server is unreachable or credentials are invalid. Scripts like
	//   until mc alias set myminio ... ; do sleep 5; done
	// rely on this to wait for MinIO to be ready.
	client := newMinioClientFromParts(rawURL, accessKey, secretKey)
	if _, err := client.ListBuckets(context.Background()); err != nil {
		fatalf("error: %v\n", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		fatalf("error loading config: %v\n", err)
	}
	cfg.Aliases[name] = Alias{URL: rawURL, AccessKey: accessKey, SecretKey: secretKey}
	if err := saveConfig(cfg); err != nil {
		fatalf("error saving config: %v\n", err)
	}
	logf("Added `%s` successfully.\n", name)
}

func cmdAliasList() {
	cfg, err := loadConfig()
	if err != nil {
		fatalf("error loading config: %v\n", err)
	}
	for name, a := range cfg.Aliases {
		fmt.Printf("%s\n", name)
		fmt.Printf("  URL       : %s\n", a.URL)
		fmt.Printf("  AccessKey : %s\n", a.AccessKey)
		fmt.Printf("  SecretKey : %s\n", a.SecretKey)
		fmt.Println()
	}
}

func cmdAliasRemove(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: alias remove <name>")
		os.Exit(1)
	}
	name := args[0]
	cfg, err := loadConfig()
	if err != nil {
		fatalf("error loading config: %v\n", err)
	}
	delete(cfg.Aliases, name)
	if err := saveConfig(cfg); err != nil {
		fatalf("error saving config: %v\n", err)
	}
	logf("Removed `%s` successfully.\n", name)
}
