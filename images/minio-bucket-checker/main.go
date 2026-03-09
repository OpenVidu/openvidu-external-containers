package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	host := os.Getenv("MINIO_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	endpoint := fmt.Sprintf("%s:%s", host, os.Getenv("MINIO_API_PORT_NUMBER"))
	bucket := os.Getenv("BUCKET_NAME")

	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			os.Getenv("MINIO_ROOT_USER"),
			os.Getenv("MINIO_ROOT_PASSWORD"),
			"",
		),
		Secure: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create MinIO client: %v\n", err)
		os.Exit(1)
	}

	for {
		_, err := client.ListBuckets(context.Background())
		if err == nil {
			break
		}
		fmt.Println("Waiting for MinIO to start...")
		time.Sleep(5 * time.Second)
	}

	for {
		exists, err := client.BucketExists(context.Background(), bucket)
		if err == nil && exists {
			break
		}
		fmt.Printf("Bucket %q not found, waiting...\n", bucket)
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("MinIO and bucket %q are ready!\n", bucket)
}
