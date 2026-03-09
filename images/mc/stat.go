package main

import (
	"context"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func cmdStat(args []string) {
	if len(args) == 0 {
		fatalf("Usage: stat <alias>/<bucket>[/<object>]\n")
	}
	target := strings.TrimRight(args[0], "/")
	aliasName, rest := parseAlias(target)
	if rest == "" {
		fatalf("invalid target %q: expected alias/bucket\n", target)
	}

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
	bucket, object, _ := strings.Cut(rest, "/")

	if object == "" {
		// Stat bucket
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			fatalf("error: %v\n", err)
		}
		if !exists {
			fatalf("bucket %q does not exist\n", bucket)
		}
		logf("Name      : %s\n", bucket)
		return
	}

	// Stat object
	info, err := client.StatObject(ctx, bucket, object, minio.StatObjectOptions{})
	if err != nil {
		fatalf("error: %v\n", err)
	}
	logf("Name          : %s\n", object)
	logf("Size          : %d\n", info.Size)
	logf("LastModified  : %s\n", info.LastModified)
	logf("ContentType   : %s\n", info.ContentType)
}
