package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func cmdAnonymous(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: anonymous <set|get> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		cmdAnonymousSet(args[1:])
	case "get":
		cmdAnonymousGet(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown anonymous subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdAnonymousSet(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: anonymous set <policy> <alias>/<bucket>[/]")
		fmt.Fprintln(os.Stderr, "Policies: private, public, download, upload")
		os.Exit(1)
	}
	policy := args[0]
	target := strings.TrimRight(args[1], "/")

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

	// "private" is the canonical upstream name for "no public access";
	// "none" is also accepted for compatibility.
	if policy == "private" {
		policy = "none"
	}

	policyJSON, err := bucketPolicy(bucket, policy)
	if err != nil {
		fatalf("%v\n", err)
	}

	// minio-go removes the policy when the string is empty
	if err := client.SetBucketPolicy(context.Background(), bucket, policyJSON); err != nil {
		fatalf("error setting bucket policy: %v\n", err)
	}
	logf("Access permission for `%s/%s` is set to `%s`\n", aliasName, bucket, policy)
}

func cmdAnonymousGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: anonymous get <alias>/<bucket>[/]")
		os.Exit(1)
	}
	target := strings.TrimRight(args[0], "/")

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

	policyJSON, err := client.GetBucketPolicy(context.Background(), bucket)
	if err != nil {
		fatalf("error getting bucket policy: %v\n", err)
	}

	fmt.Printf("Access permission for `%s/%s` is `%s`\n", aliasName, bucket, policyJSONToName(policyJSON))
}

// policyJSONToName maps a raw bucket policy JSON back to its human-readable
// name (none, download, upload, public) by inspecting the actions present in
// the policy statements.
func policyJSONToName(policyJSON string) string {
	if policyJSON == "" {
		return "none"
	}

	var doc struct {
		Statement []struct {
			Action []string `json:"Action"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		return "custom"
	}

	var hasGet, hasPut bool
	for _, stmt := range doc.Statement {
		for _, action := range stmt.Action {
			switch action {
			case "s3:GetObject":
				hasGet = true
			case "s3:PutObject":
				hasPut = true
			}
		}
	}

	switch {
	case hasGet && hasPut:
		return "public"
	case hasGet:
		return "download"
	case hasPut:
		return "upload"
	default:
		return "custom"
	}
}

func bucketPolicy(bucket, policy string) (string, error) {
	arn := "arn:aws:s3:::" + bucket
	switch policy {
	case "none":
		return "", nil
	case "download":
		return fmt.Sprintf(
			`{"Version":"2012-10-17","Statement":[`+
				`{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetBucketLocation","s3:ListBucket"],"Resource":[%q]},`+
				`{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":[%q]}`+
				`]}`,
			arn, arn+"/*",
		), nil
	case "upload":
		return fmt.Sprintf(
			`{"Version":"2012-10-17","Statement":[`+
				`{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Resource":[%q]},`+
				`{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Resource":[%q]}`+
				`]}`,
			arn, arn+"/*",
		), nil
	case "public":
		return fmt.Sprintf(
			`{"Version":"2012-10-17","Statement":[`+
				`{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetBucketLocation","s3:ListBucket"],"Resource":[%q]},`+
				`{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject","s3:PutObject"],"Resource":[%q]}`+
				`]}`,
			arn, arn+"/*",
		), nil
	}
	return "", fmt.Errorf("unknown policy %q: valid values are private, public, download, upload", policy)
}
