package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
)

func cmdRb(args []string) {
	fs := flag.NewFlagSet("rb", flag.ExitOnError)
	force := fs.Bool("force", false, "force a recursive remove operation on all object versions")
	dangerous := fs.Bool("dangerous", false, "allow site-wide removal of objects")
	fs.Parse(args)
	rest := fs.Args()

	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: rb [--force] [--dangerous] <alias>/<bucket>")
		os.Exit(1)
	}

	_ = dangerous // accepted for CLI compatibility; no extra guard needed at this scope

	for _, target := range rest {
		aliasName, bucket := parseAlias(target)
		if bucket == "" {
			fatalf("invalid target %q: expected alias/bucket\n", target)
		}
		removeBucket(aliasName, bucket, target, *force)
	}
}

func removeBucket(aliasName, bucket, target string, force bool) {
	client := newMinioClient(aliasName)
	ctx := context.Background()

	if force {
		// Delete all objects (and their versions) before removing the bucket.
		objectsCh := make(chan minio.ObjectInfo)
		go func() {
			defer close(objectsCh)
			for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true, WithVersions: true}) {
				if obj.Err != nil {
					fatalf("rb: listing objects in %q: %v\n", bucket, obj.Err)
				}
				objectsCh <- obj
			}
		}()

		for rErr := range client.RemoveObjects(ctx, bucket, objectsCh, minio.RemoveObjectsOptions{}) {
			if rErr.Err != nil {
				fatalf("rb: removing object %q: %v\n", rErr.ObjectName, rErr.Err)
			}
		}
	}

	if err := client.RemoveBucket(ctx, bucket); err != nil {
		fatalf("rb: removing bucket %q: %v\n", target, err)
	}
	logf("Removed `%s` successfully.\n", target)
}
