package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func cmdMirror(args []string) {
	fs2 := flag.NewFlagSet("mirror", flag.ExitOnError)
	overwrite := fs2.Bool("overwrite", false, "overwrite existing destination files/objects")
	workers := fs2.Int("max-workers", 0, "maximum number of concurrent copies (default: autodetect)")
	fs2.Parse(args)
	rest := fs2.Args()

	if len(rest) < 2 {
		fatalf("Usage: mirror [--overwrite] <source> <target>\n")
	}
	src, dst := rest[0], rest[1]

	w := *workers
	if w == 0 {
		w = runtime.NumCPU()
	}
	switch {
	case isRemoteTarget(src) && !isRemoteTarget(dst):
		mirrorRemoteToLocal(src, dst, *overwrite, w)
	case !isRemoteTarget(src) && isRemoteTarget(dst):
		mirrorLocalToRemote(src, dst, *overwrite, w)
	default:
		fatalf("mirror: one side must be a MinIO alias and the other a local path\n")
	}
}

// isRemoteTarget returns true when target starts with a known alias followed by '/'.
func isRemoteTarget(target string) bool {
	i := strings.IndexByte(target, '/')
	if i < 0 {
		return false
	}
	cfg, err := loadConfig()
	if err != nil {
		return false
	}
	_, ok := cfg.Aliases[target[:i]]
	return ok
}

// mirrorRemoteToLocal downloads every object under alias/bucket[/prefix] into dst/.
func mirrorRemoteToLocal(src, dst string, overwrite bool, workers int) {
	aliasName, rest := parseAlias(src)
	if rest == "" {
		fatalf("mirror: source must be alias/bucket[/prefix]\n")
	}
	bucket, prefix, _ := strings.Cut(rest, "/")

	client := newMinioClient(aliasName)
	ctx := context.Background()

	type job struct {
		key       string
		localPath string
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

	recordErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			fatalf("mirror: listing objects: %v\n", obj.Err)
		}

		mu.Lock()
		if firstErr != nil {
			mu.Unlock()
			break
		}
		mu.Unlock()

		// Strip the bucket-level prefix so the local tree mirrors the bucket root.
		relKey := strings.TrimPrefix(obj.Key, prefix)
		relKey = strings.TrimPrefix(relKey, "/")
		j := job{key: obj.Key, localPath: filepath.Join(dst, filepath.FromSlash(relKey))}

		if !overwrite {
			if _, err := os.Stat(j.localPath); err == nil {
				logf("Skipping `%s` (already exists)\n", j.localPath)
				continue
			}
		}

		if err := os.MkdirAll(filepath.Dir(j.localPath), 0755); err != nil {
			fatalf("mirror: mkdir %s: %v\n", filepath.Dir(j.localPath), err)
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()

			reader, err := client.GetObject(ctx, bucket, j.key, minio.GetObjectOptions{})
			if err != nil {
				recordErr(fmt.Errorf("get %s: %w", j.key, err))
				return
			}
			defer reader.Close()

			f, err := os.Create(j.localPath)
			if err != nil {
				recordErr(fmt.Errorf("create %s: %w", j.localPath, err))
				return
			}
			defer f.Close()

			if _, err := io.Copy(f, reader); err != nil {
				recordErr(fmt.Errorf("write %s: %w", j.localPath, err))
				return
			}
			logf("`%s/%s` -> `%s`\n", bucket, j.key, j.localPath)
		}(j)
	}

	wg.Wait()
	if firstErr != nil {
		fatalf("mirror: %v\n", firstErr)
	}
}

// mirrorLocalToRemote uploads every file under src/ to alias/bucket[/prefix].
func mirrorLocalToRemote(src, dst string, overwrite bool, workers int) {
	aliasName, rest := parseAlias(dst)
	if rest == "" {
		fatalf("mirror: destination must be alias/bucket[/prefix]\n")
	}
	bucket, prefix, _ := strings.Cut(rest, "/")

	client := newMinioClient(aliasName)
	ctx := context.Background()

	type job struct {
		path string
		key  string
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

	recordErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	walkErr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		mu.Lock()
		if firstErr != nil {
			mu.Unlock()
			return firstErr
		}
		mu.Unlock()

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		if prefix != "" {
			key = prefix + "/" + key
		}

		if !overwrite {
			if _, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err == nil {
				logf("Skipping `%s` (already exists)\n", key)
				return nil
			}
		}

		j := job{path: path, key: key}
		sem <- struct{}{}
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()

			if _, err := client.FPutObject(ctx, bucket, j.key, j.path, minio.PutObjectOptions{}); err != nil {
				recordErr(fmt.Errorf("upload %s: %w", j.path, err))
				return
			}
			logf("`%s` -> `%s/%s`\n", j.path, bucket, j.key)
		}(j)
		return nil
	})

	wg.Wait()
	if walkErr != nil {
		fatalf("mirror: %v\n", walkErr)
	}
	if firstErr != nil {
		fatalf("mirror: %v\n", firstErr)
	}
}

// newMinioClient builds a minio-go client from a stored alias, exiting on error.
func newMinioClient(aliasName string) *minio.Client {
	a, err := getAlias(aliasName)
	if err != nil {
		fatalf("error: %v\n", err)
	}
	return newMinioClientFromParts(a.URL, a.AccessKey, a.SecretKey)
}

// newMinioClientFromParts builds a minio-go client from a raw URL and credentials.
func newMinioClientFromParts(rawURL, accessKey, secretKey string) *minio.Client {
	endpoint, useSSL, err := aliasEndpoint(Alias{URL: rawURL})
	if err != nil {
		fatalf("error: %v\n", err)
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		fatalf("error creating client: %v\n", err)
	}
	return client
}
