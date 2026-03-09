package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"

	"github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func cmdAdmin(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: admin <info|service> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "info":
		cmdAdminInfo(args[1:])
	case "service":
		cmdAdminService(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown admin subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdAdminInfo(args []string) {
	// --json may appear anywhere in args (before or after alias name)
	asJSON := false
	var rest []string
	for _, arg := range args {
		if arg == "--json" || arg == "-json" {
			asJSON = true
		} else {
			rest = append(rest, arg)
		}
	}

	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: admin info <alias> [--json]")
		os.Exit(1)
	}
	aliasName := rest[0]

	client, err := newAdminClient(aliasName)
	if err != nil {
		fatalf("error creating admin client: %v\n", err)
	}

	// Upstream mc does NOT exit 1 on ServerInfo failure — it prints a
	// structured error response and exits 0. Only client init failures exit 1.
	info, infoErr := client.ServerInfo(context.Background())

	if asJSON {
		var out map[string]any
		if infoErr != nil {
			out = map[string]any{"status": "error", "error": infoErr.Error()}
		} else {
			out = map[string]any{"status": "success", "info": info}
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
	} else {
		if infoErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", infoErr)
		} else {
			printAdminInfo(info)
		}
	}
}

func cmdAdminService(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: admin service <stop|restart> <alias>")
		os.Exit(1)
	}
	switch args[0] {
	case "stop":
		cmdAdminServiceStop(args[1:])
	case "restart":
		cmdAdminServiceRestart(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown service subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdAdminServiceStop(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: admin service stop <alias>")
		os.Exit(1)
	}
	client, err := newAdminClient(args[0])
	if err != nil {
		fatalf("error creating admin client: %v\n", err)
	}
	if err := client.ServiceStop(context.Background()); err != nil {
		fatalf("error stopping service: %v\n", err)
	}
	logf("MinIO service stopped successfully\n")
}

func cmdAdminServiceRestart(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: admin service restart <alias>")
		os.Exit(1)
	}
	client, err := newAdminClient(args[0])
	if err != nil {
		fatalf("error creating admin client: %v\n", err)
	}
	if err := client.ServiceRestart(context.Background()); err != nil {
		fatalf("error restarting service: %v\n", err)
	}
	logf("MinIO service restarted successfully\n")
}

func printAdminInfo(info madmin.InfoMessage) {
	for _, srv := range info.Servers {
		netOnline := 0
		for _, v := range srv.Network {
			if v == "online" {
				netOnline++
			}
		}
		driveOK := 0
		for _, d := range srv.Disks {
			if d.State == "ok" {
				driveOK++
			}
		}
		fmt.Printf("%s\n", srv.Endpoint)
		fmt.Printf("   Uptime: %s\n", formatUptime(srv.Uptime))
		fmt.Printf("   Version: %s\n", srv.Version)
		fmt.Printf("   Network: %d/%d OK\n", netOnline, len(srv.Network))
		fmt.Printf("   Drives: %d/%d OK\n", driveOK, len(srv.Disks))
		fmt.Printf("   Pool: %d\n", srv.PoolNumber)
	}
	fmt.Println()

	// Collect disk space per pool across all servers.
	type poolStats struct {
		total uint64
		used  uint64
	}
	pools := make(map[int]*poolStats)
	for _, srv := range info.Servers {
		for _, d := range srv.Disks {
			pi := d.PoolIndex
			if pools[pi] == nil {
				pools[pi] = &poolStats{}
			}
			pools[pi].total += d.TotalSpace
			pools[pi].used += d.UsedSpace
		}
	}

	indices := make([]int, 0, len(pools))
	for pi := range pools {
		indices = append(indices, pi)
	}
	sort.Ints(indices)

	fmt.Printf("┌──────┬───────────────────────┬─────────────────────┬──────────────┐\n")
	fmt.Printf("│ Pool │ Drives Usage          │ Erasure stripe size │ Erasure sets │\n")
	for _, pi := range indices {
		ps := pools[pi]
		var pct float64
		if ps.total > 0 {
			pct = float64(ps.used) / float64(ps.total) * 100
		}
		usageStr := fmt.Sprintf("%.1f%% (total: %s)", pct, formatSize(int64(ps.total)))
		drivesPerSet := 0
		if pi < len(info.Backend.DrivesPerSet) {
			drivesPerSet = info.Backend.DrivesPerSet[pi]
		}
		totalSets := 0
		if pi < len(info.Backend.TotalSets) {
			totalSets = info.Backend.TotalSets[pi]
		}
		fmt.Printf("│ %-4s │ %-21s │ %-19d │ %-12d │\n", ordinal(pi+1), usageStr, drivesPerSet, totalSets)
	}
	fmt.Printf("└──────┴───────────────────────┴─────────────────────┴──────────────┘\n")
	fmt.Println()

	fmt.Printf("%s Used, %s, %s\n",
		formatSize(int64(info.Usage.Size)),
		pluralize(int(info.Buckets.Count), "Bucket", "Buckets"),
		pluralize(int(info.Objects.Count), "Object", "Objects"),
	)
	fmt.Printf("%s, %s, EC:%d\n",
		pluralize(info.Backend.OnlineDisks, "drive online", "drives online"),
		pluralize(info.Backend.OfflineDisks, "drive offline", "drives offline"),
		info.Backend.StandardSCParity,
	)
}

func formatUptime(seconds int64) string {
	switch {
	case seconds >= 86400:
		return pluralize(int(seconds/86400), "day", "days")
	case seconds >= 3600:
		return pluralize(int(seconds/3600), "hour", "hours")
	case seconds >= 60:
		return pluralize(int(seconds/60), "minute", "minutes")
	default:
		return pluralize(int(seconds), "second", "seconds")
	}
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}

func pluralize(n int, s, p string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, s)
	}
	return fmt.Sprintf("%d %s", n, p)
}

func newAdminClient(aliasName string) (*madmin.AdminClient, error) {
	a, err := getAlias(aliasName)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(a.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	return madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds:  credentials.NewStaticV4(a.AccessKey, a.SecretKey, ""),
		Secure: u.Scheme == "https",
	})
}
