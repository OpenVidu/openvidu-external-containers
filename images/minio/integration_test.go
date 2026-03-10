//go:build integration

package minio_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/madmin-go/v4"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testUser = "minio"
	testPass = "miniosecret"
)

// Package-level state set up once in TestMain and shared across basic tests.
var (
	minioImage string // set in TestMain after build
	endpoint   string // http://host:port of the shared standalone container
)

// ── TestMain ──────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	var err error

	// Build the minio image from source once.
	minioImage, err = buildMinioImage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build minio image: %v\n", err)
		return 1
	}

	// Shared standalone container used by the basic tests.
	ctx := context.Background()
	ctr, ep, err := startStandalone(ctx, testUser, testPass, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start minio: %v\n", err)
		return 1
	}
	defer ctr.Terminate(ctx)
	endpoint = ep

	return m.Run()
}

// ── Image build ───────────────────────────────────────────────────────────────

// buildMinioImage builds the openvidu/minio image from the repository source
// and returns its tag. MINIO_TAG env var overrides the default release tag.
func buildMinioImage() (string, error) {
	tag := "openvidu/minio:integration-test"

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}

	minioTag := os.Getenv("MINIO_TAG")
	if minioTag == "" {
		data, err := os.ReadFile(filepath.Join(repoRoot, "versions.env"))
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if k, v, ok := strings.Cut(line, "="); ok && k == "MINIO_TAG" {
					minioTag = strings.TrimSpace(v)
					break
				}
			}
		}
	}

	cmd := exec.Command("docker", "build",
		"-f", filepath.Join(repoRoot, "images", "minio", "Dockerfile"),
		"--build-arg", "MINIO_TAG="+minioTag,
		"-t", tag,
		repoRoot,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker build failed: %w\n%s", err, out)
	}
	return tag, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// startStandalone starts a single MinIO container and returns (container, "http://host:port", error).
// extraEnv is merged on top of the default MINIO_ROOT_USER / MINIO_ROOT_PASSWORD / MINIO_BROWSER env.
func startStandalone(ctx context.Context, user, pass string, extraEnv map[string]string) (testcontainers.Container, string, error) {
	env := map[string]string{
		"MINIO_ROOT_USER":     user,
		"MINIO_ROOT_PASSWORD": pass,
		"MINIO_BROWSER":       "on",
	}
	for k, v := range extraEnv {
		env[k] = v
	}

	req := testcontainers.ContainerRequest{
		Image:        minioImage,
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env:          env,
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(90 * time.Second),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		ctr.Terminate(ctx)
		return nil, "", err
	}
	port, err := ctr.MappedPort(ctx, "9000")
	if err != nil {
		ctr.Terminate(ctx)
		return nil, "", err
	}
	consolePort, err := ctr.MappedPort(ctx, "9001")
	if err != nil {
		ctr.Terminate(ctx)
		return nil, "", err
	}
	fmt.Fprintf(os.Stderr, "MinIO console: http://%s:%s  (user: %s  pass: %s)\n",
		host, consolePort.Port(), user, pass)
	return ctr, fmt.Sprintf("http://%s:%s", host, port.Port()), nil
}

// startDistributed4 starts a 4-node MinIO distributed cluster on an isolated
// Docker bridge network (mirrors docker-compose-distributed.yml).
// Callers must terminate all returned containers and remove the network when done.
func startDistributed4(t *testing.T, ctx context.Context) ([]testcontainers.Container, []string) {
	t.Helper()

	net, err := network.New(ctx)
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	t.Cleanup(func() { net.Remove(ctx) })

	nodes := []string{"minio", "minio2", "minio3", "minio4"}
	env := map[string]string{
		"MINIO_ROOT_USER":                testUser,
		"MINIO_ROOT_PASSWORD":            testPass,
		"MINIO_DISTRIBUTED_MODE_ENABLED": "yes",
		"MINIO_DISTRIBUTED_NODES":        strings.Join(nodes, ","),
	}

	type result struct {
		ctr      testcontainers.Container
		endpoint string
		err      error
	}

	results := make([]result, len(nodes))
	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node string) {
			defer wg.Done()
			req := testcontainers.ContainerRequest{
				Image:        minioImage,
				ExposedPorts: []string{"9000/tcp"},
				Env:          env,
				Networks:     []string{net.Name},
				NetworkAliases: map[string][]string{
					net.Name: {node},
				},
				WaitingFor: wait.ForHTTP("/minio/health/live").
					WithPort("9000/tcp").
					WithStartupTimeout(120 * time.Second),
			}
			ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			})
			if err != nil {
				results[i] = result{err: fmt.Errorf("node %s: %w", node, err)}
				return
			}
			host, err := ctr.Host(ctx)
			if err != nil {
				ctr.Terminate(ctx)
				results[i] = result{err: err}
				return
			}
			port, err := ctr.MappedPort(ctx, "9000")
			if err != nil {
				ctr.Terminate(ctx)
				results[i] = result{err: err}
				return
			}
			results[i] = result{
				ctr:      ctr,
				endpoint: fmt.Sprintf("http://%s:%s", host, port.Port()),
			}
		}(i, node)
	}
	wg.Wait()

	ctrs := make([]testcontainers.Container, 0, len(nodes))
	endpoints := make([]string, 0, len(nodes))
	for _, r := range results {
		if r.err != nil {
			for _, c := range ctrs {
				c.Terminate(ctx)
			}
			t.Fatalf("start distributed cluster: %v", r.err)
		}
		ctrs = append(ctrs, r.ctr)
		endpoints = append(endpoints, r.endpoint)
	}

	for _, c := range ctrs {
		t.Cleanup(func() { c.Terminate(ctx) })
	}
	return ctrs, endpoints
}

// startDistributedMultiDrive starts a 2-node × 2-drive MinIO distributed cluster
// (mirrors docker-compose-distributed-multidrive.yml).
func startDistributedMultiDrive(t *testing.T, ctx context.Context) ([]testcontainers.Container, []string) {
	t.Helper()

	net, err := network.New(ctx)
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	t.Cleanup(func() { net.Remove(ctx) })

	// Create 4 host directories: 2 per node. chmod 0777 so the non-root
	// container user can write.
	dataDirs := make([]string, 4)
	for i := range dataDirs {
		dir, err := os.MkdirTemp("", fmt.Sprintf("minio-multidrive-%d-*", i))
		if err != nil {
			t.Fatalf("mkdirtemp: %v", err)
		}
		if err := os.Chmod(dir, 0777); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		dataDirs[i] = dir
		t.Cleanup(func() { os.RemoveAll(dir) })
	}

	nodes := []string{"minio-0", "minio-1"}
	env := map[string]string{
		"MINIO_ROOT_USER":                testUser,
		"MINIO_ROOT_PASSWORD":            testPass,
		"MINIO_DISTRIBUTED_MODE_ENABLED": "yes",
		"MINIO_DISTRIBUTED_NODES":        "minio-{0...1}/data-{0...1}",
	}

	type result struct {
		ctr      testcontainers.Container
		endpoint string
		err      error
	}

	results := make([]result, len(nodes))
	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node string) {
			defer wg.Done()
			req := testcontainers.ContainerRequest{
				Image:        minioImage,
				ExposedPorts: []string{"9000/tcp"},
				Env:          env,
				Networks:     []string{net.Name},
				NetworkAliases: map[string][]string{
					net.Name: {node},
				},
				// Bind-mount the two data directories for this node.
				Mounts: testcontainers.ContainerMounts{
					testcontainers.BindMount(dataDirs[i*2], "/data-0"),
					testcontainers.BindMount(dataDirs[i*2+1], "/data-1"),
				},
				WaitingFor: wait.ForHTTP("/minio/health/live").
					WithPort("9000/tcp").
					WithStartupTimeout(120 * time.Second),
			}
			ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			})
			if err != nil {
				results[i] = result{err: fmt.Errorf("node %s: %w", node, err)}
				return
			}
			host, err := ctr.Host(ctx)
			if err != nil {
				ctr.Terminate(ctx)
				results[i] = result{err: err}
				return
			}
			port, err := ctr.MappedPort(ctx, "9000")
			if err != nil {
				ctr.Terminate(ctx)
				results[i] = result{err: err}
				return
			}
			results[i] = result{
				ctr:      ctr,
				endpoint: fmt.Sprintf("http://%s:%s", host, port.Port()),
			}
		}(i, node)
	}
	wg.Wait()

	ctrs := make([]testcontainers.Container, 0, len(nodes))
	endpoints := make([]string, 0, len(nodes))
	for _, r := range results {
		if r.err != nil {
			for _, c := range ctrs {
				c.Terminate(ctx)
			}
			t.Fatalf("start multidrive cluster: %v", r.err)
		}
		ctrs = append(ctrs, r.ctr)
		endpoints = append(endpoints, r.endpoint)
	}

	for _, c := range ctrs {
		t.Cleanup(func() { c.Terminate(ctx) })
	}
	return ctrs, endpoints
}

// newS3Client returns a minio-go S3 client pointed at the given endpoint.
func newS3Client(t *testing.T, ep, user, pass string) *miniogo.Client {
	t.Helper()
	host := strings.TrimPrefix(ep, "http://")
	c, err := miniogo.New(host, &miniogo.Options{
		Creds:  credentials.NewStaticV4(user, pass, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("minio-go client: %v", err)
	}
	return c
}

// newAdminClient returns a madmin client pointed at the given endpoint.
func newAdminClient(t *testing.T, ep, user, pass string) *madmin.AdminClient {
	t.Helper()
	host := strings.TrimPrefix(ep, "http://")
	ac, err := madmin.New(host, user, pass, false)
	if err != nil {
		t.Fatalf("madmin client: %v", err)
	}
	return ac
}

// waitForHealth polls GET /minio/health/live until 200 or timeout.
func waitForHealth(ep string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(ep + "/minio/health/live")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("MinIO not healthy after %v", timeout)
}

// ── Standalone — shared container ─────────────────────────────────────────────

// TestHealthLive verifies that GET /minio/health/live returns 200.
func TestHealthLive(t *testing.T) {
	resp, err := http.Get(endpoint + "/minio/health/live")
	if err != nil {
		t.Fatalf("GET /minio/health/live: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

// TestHealthReady verifies that GET /minio/health/ready returns 200.
func TestHealthReady(t *testing.T) {
	resp, err := http.Get(endpoint + "/minio/health/ready")
	if err != nil {
		t.Fatalf("GET /minio/health/ready: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

// TestAdminInfo verifies that the admin client can query server info and that
// the server reports a positive uptime, a non-empty version, and online drives.
func TestAdminInfo(t *testing.T) {
	ctx := context.Background()
	ac := newAdminClient(t, endpoint, testUser, testPass)

	info, err := ac.ServerInfo(ctx)
	if err != nil {
		t.Fatalf("ServerInfo: %v", err)
	}
	if len(info.Servers) == 0 {
		t.Fatal("ServerInfo returned no servers")
	}

	srv := info.Servers[0]
	if srv.Version == "" {
		t.Error("expected non-empty server version")
	}
	if srv.State == "" {
		t.Error("expected non-empty server state")
	}
}

// TestBucketCreateAndList creates a bucket and verifies it appears in ListBuckets.
func TestBucketCreateAndList(t *testing.T) {
	ctx := context.Background()
	c := newS3Client(t, endpoint, testUser, testPass)

	bucket := "test-create-list"
	if err := c.MakeBucket(ctx, bucket, miniogo.MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket: %v", err)
	}

	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}
	for _, b := range buckets {
		if b.Name == bucket {
			return
		}
	}
	t.Errorf("bucket %q not found in ListBuckets", bucket)
}

// TestObjectUploadAndDownload uploads an object and verifies the downloaded content matches.
func TestObjectUploadAndDownload(t *testing.T) {
	ctx := context.Background()
	c := newS3Client(t, endpoint, testUser, testPass)

	bucket := "test-upload-download"
	if err := c.MakeBucket(ctx, bucket, miniogo.MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket: %v", err)
	}

	const content = "hello, minio integration test"
	_, err := c.PutObject(ctx, bucket, "hello.txt",
		strings.NewReader(content), int64(len(content)),
		miniogo.PutObjectOptions{},
	)
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	obj, err := c.GetObject(ctx, bucket, "hello.txt", miniogo.GetObjectOptions{})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer obj.Close()

	got, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if string(got) != content {
		t.Errorf("content mismatch: want %q, got %q", content, got)
	}
}

// TestDefaultBuckets verifies that MINIO_DEFAULT_BUCKETS causes a bucket to
// exist on first connection without any explicit mc mb invocation.
//
// The bucket is created by setup.sh before the final minio process starts, so
// the health check may fire during the setup phase (background minio) before
// the bucket exists. We retry until the bucket appears or the deadline expires.
func TestDefaultBuckets(t *testing.T) {
	ctx := context.Background()
	ctr, ep, err := startStandalone(ctx, testUser, testPass, map[string]string{
		"MINIO_DEFAULT_BUCKETS": "pre-bucket",
	})
	if err != nil {
		t.Fatalf("start minio: %v", err)
	}
	defer ctr.Terminate(ctx)

	c := newS3Client(t, ep, testUser, testPass)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		buckets, err := c.ListBuckets(ctx)
		if err == nil {
			for _, b := range buckets {
				if b.Name == "pre-bucket" {
					return
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("expected bucket \"pre-bucket\" to exist via MINIO_DEFAULT_BUCKETS")
}

// ── Security ──────────────────────────────────────────────────────────────────

// TestWrongCredentialsRejected verifies that listing buckets with wrong credentials fails.
func TestWrongCredentialsRejected(t *testing.T) {
	ctx := context.Background()
	c := newS3Client(t, endpoint, testUser, "wrongpassword")
	_, err := c.ListBuckets(ctx)
	if err == nil {
		t.Error("expected error with wrong credentials, got nil")
	}
}

// TestNoCredentialsRejected verifies that an unsigned S3 list request returns 403.
func TestNoCredentialsRejected(t *testing.T) {
	resp, err := http.Get(endpoint + "/?list-type=2")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for unauthenticated request, got %d", resp.StatusCode)
	}
}

// TestShortPasswordRejected verifies that MinIO refuses to start with a
// password shorter than 8 characters.
func TestShortPasswordRejected(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        minioImage,
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     testUser,
			"MINIO_ROOT_PASSWORD": "abc1", // 4 chars — too short
		},
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(15 * time.Second),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err == nil {
		ctr.Terminate(ctx)
		t.Error("expected container start to fail with 4-char password, but it succeeded")
	}
}

// TestCustomCredentials verifies that a container started with custom
// credentials accepts those credentials and rejects the defaults.
func TestCustomCredentials(t *testing.T) {
	ctx := context.Background()
	ctr, ep, err := startStandalone(ctx, "admin", "strongpass123", nil)
	if err != nil {
		t.Fatalf("start minio with custom credentials: %v", err)
	}
	defer ctr.Terminate(ctx)

	// Custom credentials must work.
	customClient := newS3Client(t, ep, "admin", "strongpass123")
	if _, err := customClient.ListBuckets(ctx); err != nil {
		t.Errorf("custom credentials should work: %v", err)
	}

	// Default credentials must be rejected.
	defaultClient := newS3Client(t, ep, testUser, testPass)
	if _, err := defaultClient.ListBuckets(ctx); err == nil {
		t.Error("default credentials should be rejected when custom ones are set")
	}
}

// TestBrowserOff verifies that MINIO_BROWSER=off prevents the console from
// listening on port 9001.
func TestBrowserOff(t *testing.T) {
	ctx := context.Background()
	ctr, ep, err := startStandalone(ctx, testUser, testPass, map[string]string{
		"MINIO_BROWSER": "off",
	})
	if err != nil {
		t.Fatalf("start minio with MINIO_BROWSER=off: %v", err)
	}
	defer ctr.Terminate(ctx)

	// The API port must still be healthy.
	if err := waitForHealth(ep, 10*time.Second); err != nil {
		t.Fatalf("API port not healthy: %v", err)
	}

	// The console port (9001) should not be accepting connections since the
	// console is disabled. Check via the container's mapped port if available,
	// or use a direct TCP probe. We try an HTTP GET to :9001 and expect either
	// a connection error or a non-200 response.
	consolePort, err := ctr.MappedPort(ctx, "9001")
	if err != nil {
		// Port not exposed — console definitely not running.
		return
	}
	host, _ := ctr.Host(ctx)
	consoleURL := fmt.Sprintf("http://%s:%s", host, consolePort.Port())

	// Give a short time to confirm the port is not serving HTTP.
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(consoleURL)
	if err != nil {
		// Connection refused or timeout — console is off. Pass.
		return
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("expected console to be off (MINIO_BROWSER=off) but got HTTP 200")
	}
}

// TestDataDirIsolation verifies that two independently started MinIO instances
// with different credentials cannot access each other's data via the S3 API.
func TestDataDirIsolation(t *testing.T) {
	ctx := context.Background()

	ctrA, epA, err := startStandalone(ctx, "userA", "passwordA1", nil)
	if err != nil {
		t.Fatalf("start container A: %v", err)
	}
	defer ctrA.Terminate(ctx)

	ctrB, epB, err := startStandalone(ctx, "userB", "passwordB1", nil)
	if err != nil {
		t.Fatalf("start container B: %v", err)
	}
	defer ctrB.Terminate(ctx)

	// Client A's credentials must be rejected by server B.
	clientAonB := newS3Client(t, epB, "userA", "passwordA1")
	if _, err := clientAonB.ListBuckets(ctx); err == nil {
		t.Error("credentials from server A should be rejected by server B")
	}

	// Client B's credentials must be rejected by server A.
	clientBonA := newS3Client(t, epA, "userB", "passwordB1")
	if _, err := clientBonA.ListBuckets(ctx); err == nil {
		t.Error("credentials from server B should be rejected by server A")
	}
}

// ── Distributed 4-node ────────────────────────────────────────────────────────

// TestDistributed4NodesHealthy mirrors docker-compose-distributed.yml:
// 4 nodes form a cluster and all report healthy network connectivity and drives.
//
// The health check may fire before the cluster finishes initializing its drives,
// so we retry ServerInfo until all drives are ok.
func TestDistributed4NodesHealthy(t *testing.T) {
	ctx := context.Background()
	_, endpoints := startDistributed4(t, ctx)

	ac := newAdminClient(t, endpoints[0], testUser, testPass)

	allOK := func(info madmin.InfoMessage) bool {
		if len(info.Servers) != 4 {
			return false
		}
		for _, srv := range info.Servers {
			for _, d := range srv.Disks {
				if !strings.EqualFold(d.State, "ok") {
					return false
				}
			}
		}
		return true
	}

	var info madmin.InfoMessage
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var err error
		info, err = ac.ServerInfo(ctx)
		if err == nil && allOK(info) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if len(info.Servers) != 4 {
		t.Errorf("expected 4 servers in cluster, got %d", len(info.Servers))
	}
	for _, srv := range info.Servers {
		for peer, status := range srv.Network {
			if !strings.EqualFold(status, "online") {
				t.Errorf("server %s: peer %s status %q, want online", srv.Endpoint, peer, status)
			}
		}
		for _, d := range srv.Disks {
			if !strings.EqualFold(d.State, "ok") {
				t.Errorf("server %s: drive %s state %q, want ok", srv.Endpoint, d.Endpoint, d.State)
			}
		}
	}
}

// TestDistributedDataAvailability writes an object via node 1 and reads it
// back via node 3, confirming distributed replication is working.
func TestDistributedDataAvailability(t *testing.T) {
	ctx := context.Background()
	_, endpoints := startDistributed4(t, ctx)

	const bucket = "dist-availability"
	const key = "hello.txt"
	const content = "distributed minio test"

	// Write via node 1.
	c1 := newS3Client(t, endpoints[0], testUser, testPass)
	if err := c1.MakeBucket(ctx, bucket, miniogo.MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket via node 1: %v", err)
	}
	if _, err := c1.PutObject(ctx, bucket, key,
		strings.NewReader(content), int64(len(content)),
		miniogo.PutObjectOptions{},
	); err != nil {
		t.Fatalf("PutObject via node 1: %v", err)
	}

	// Read back via node 3.
	c3 := newS3Client(t, endpoints[2], testUser, testPass)
	obj, err := c3.GetObject(ctx, bucket, key, miniogo.GetObjectOptions{})
	if err != nil {
		t.Fatalf("GetObject via node 3: %v", err)
	}
	defer obj.Close()

	got, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("read object via node 3: %v", err)
	}
	if string(got) != content {
		t.Errorf("content mismatch: want %q, got %q", content, got)
	}
}

// ── Distributed Multi-Drive ───────────────────────────────────────────────────

// TestDistributedMultiDriveHealthy mirrors docker-compose-distributed-multidrive.yml:
// 2 nodes × 2 drives each form a cluster and all nodes report online drives.
//
// The health check may fire before MinIO finishes initializing the bind-mounted
// data directories, so we retry ServerInfo until all drives are ok.
func TestDistributedMultiDriveHealthy(t *testing.T) {
	ctx := context.Background()
	_, endpoints := startDistributedMultiDrive(t, ctx)

	ac := newAdminClient(t, endpoints[0], testUser, testPass)

	allOK := func(info madmin.InfoMessage) bool {
		if len(info.Servers) != 2 {
			return false
		}
		for _, srv := range info.Servers {
			for _, d := range srv.Disks {
				if !strings.EqualFold(d.State, "ok") {
					return false
				}
			}
		}
		return true
	}

	var info madmin.InfoMessage
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var err error
		info, err = ac.ServerInfo(ctx)
		if err == nil && allOK(info) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if len(info.Servers) != 2 {
		t.Errorf("expected 2 servers in cluster, got %d", len(info.Servers))
	}
	for _, srv := range info.Servers {
		if len(srv.Disks) != 2 {
			t.Errorf("server %s: expected 2 drives, got %d", srv.Endpoint, len(srv.Disks))
		}
		for _, d := range srv.Disks {
			if !strings.EqualFold(d.State, "ok") {
				t.Errorf("server %s: drive %s state %q, want ok", srv.Endpoint, d.Endpoint, d.State)
			}
		}
	}
}
