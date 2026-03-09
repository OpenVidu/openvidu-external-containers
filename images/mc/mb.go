package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func cmdMb(args []string) {
	fs := flag.NewFlagSet("mb", flag.ExitOnError)
	region := fs.String("region", "", "region name")
	ignoreExisting := fs.Bool("ignore-existing", false, "ignore if bucket already exists")
	fs.BoolVar(ignoreExisting, "p", false, "ignore if bucket already exists (shorthand for --ignore-existing)")
	fs.Parse(args)
	rest := fs.Args()

	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: mb [--region REGION] [--ignore-existing] <alias>/<bucket>")
		os.Exit(1)
	}
	target := rest[0]
	aliasName, bucket := parseAlias(target)
	if bucket == "" {
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

	if err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: *region}); err != nil {
		if *ignoreExisting {
			switch minio.ToErrorResponse(err).Code {
			case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
				logf("Bucket `%s/%s` already exists, ignoring.\n", strings.TrimRight(target, "/"+bucket), bucket)
				return
			}
		}
		fatalf("error creating bucket: %v\n", err)
	}
	logf("Bucket created successfully `%s/%s`.\n", strings.TrimRight(target, "/"+bucket), bucket)
}
