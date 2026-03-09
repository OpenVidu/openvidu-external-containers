package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func cmdLs(args []string) {
	if len(args) == 0 {
		fatalf("Usage: ls <alias>[/<bucket>[/<prefix>]]\n")
	}
	target := args[0]
	aliasName, rest := parseAlias(target)

	a, err := getAlias(aliasName)
	if err != nil {
		fatalf("error: %v\n", err)
	}
	endpoint, useSSL, err := aliasEndpoint(a)
	if err != nil {
		fatalf("error: %v\n", err)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(a.AccessKey, a.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		fatalf("error creating client: %v\n", err)
	}

	ctx := context.Background()

	if rest == "" {
		// List buckets
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			fatalf("error listing buckets: %v\n", err)
		}
		for _, b := range buckets {
			logf("[%s] %8s %s\n", b.CreationDate.Format("2006-01-02 15:04:05 MST"), "0B", b.Name)
		}
		return
	}

	// Split rest into bucket[/prefix]
	bucket, prefix, _ := strings.Cut(rest, "/")

	for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix}) {
		if obj.Err != nil {
			fatalf("error: %v\n", obj.Err)
		}
		logf("[%s] %8s %s\n", obj.LastModified.Format("2006-01-02 15:04:05 MST"), formatSize(obj.Size), obj.Key)
	}
}

func formatSize(n int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.1fGiB", float64(n)/GB)
	case n >= MB:
		return fmt.Sprintf("%.1fMiB", float64(n)/MB)
	case n >= KB:
		return fmt.Sprintf("%.1fKiB", float64(n)/KB)
	default:
		return fmt.Sprintf("%dB", n)
	}
}
