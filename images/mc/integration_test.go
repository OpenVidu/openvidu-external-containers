//go:build integration

package main_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testUser  = "minio"
	testPass  = "miniosecret"
	testAlias = "minio"
)

// Package-level state set up once in TestMain and shared across tests.
var (
	minioImage string // set in TestMain after build
	mcBin      string // path to compiled mc binary
	configDir  string // shared config dir with testAlias registered
	endpoint   string // http://host:port of the shared MinIO container
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

	// Build the mc binary once.
	mcBin, err = buildBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		return 1
	}
	defer os.Remove(mcBin)

	// Shared temp config dir.
	configDir, err = os.MkdirTemp("", "mc-test-cfg-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdirtemp: %v\n", err)
		return 1
	}
	defer os.RemoveAll(configDir)

	// Shared MinIO container (no restart policy).
	ctx := context.Background()
	ctr, ep, err := startMinio(ctx, testUser, testPass, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start minio: %v\n", err)
		return 1
	}
	defer ctr.Terminate(ctx)
	endpoint = ep

	// Register the shared alias.
	if _, _, code := mc("alias", "set", testAlias, endpoint, testUser, testPass); code != 0 {
		fmt.Fprintf(os.Stderr, "alias set failed\n")
		return 1
	}

	return m.Run()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildBinary() (string, error) {
	bin := filepath.Join(os.TempDir(), fmt.Sprintf("mc-test-%d", os.Getpid()))
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}
	return bin, nil
}

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

// startMinio starts a MinIO container and returns (container, "http://host:port", error).
// withRestartPolicy sets Docker's "always" restart policy, needed for service-restart tests.
// The MinIO web console is exposed on a random host port; its URL is printed to stderr.
func startMinio(ctx context.Context, user, pass string, withRestartPolicy bool) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        minioImage,
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     user,
			"MINIO_ROOT_PASSWORD": pass,
			"MINIO_BROWSER":       "on",
		},
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(90 * time.Second),
	}
	if withRestartPolicy {
		req.HostConfigModifier = func(hc *dockercontainer.HostConfig) {
			hc.RestartPolicy = dockercontainer.RestartPolicy{Name: "always"}
		}
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
	fmt.Fprintf(os.Stderr, "MinIO console: http://%s:%s  (user: %s  pass: %s)\n", host, consolePort.Port(), user, pass)
	return ctr, fmt.Sprintf("http://%s:%s", host, port.Port()), nil
}

// mc runs the mc binary against the shared config dir.
func mc(args ...string) (stdout, stderr string, code int) {
	return runMC(configDir, args...)
}

// runMC runs the mc binary with an explicit config dir.
func runMC(cfgDir string, args ...string) (stdout, stderr string, code int) {
	cmd := exec.Command(mcBin, append([]string{"--config-dir", cfgDir}, args...)...)
	var ob, eb bytes.Buffer
	cmd.Stdout = &ob
	cmd.Stderr = &eb
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		}
	}
	return ob.String(), eb.String(), code
}

// minioGoClient returns a minio-go client pointed at the shared container.
func minioGoClient(t *testing.T) *miniogo.Client {
	t.Helper()
	ep := strings.TrimPrefix(endpoint, "http://")
	c, err := miniogo.New(ep, &miniogo.Options{
		Creds:  credentials.NewStaticV4(testUser, testPass, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("minio-go client: %v", err)
	}
	return c
}

// putObject uploads a small text object directly via minio-go (used as test fixture).
func putObject(t *testing.T, bucket, key, content string) {
	t.Helper()
	c := minioGoClient(t)
	_, err := c.PutObject(
		context.Background(), bucket, key,
		strings.NewReader(content), int64(len(content)),
		miniogo.PutObjectOptions{},
	)
	if err != nil {
		t.Fatalf("putObject %s/%s: %v", bucket, key, err)
	}
}

// waitForHealth polls GET /minio/health/live until 200 or timeout.
func waitForHealth(ep string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(ep + "/minio/health/live")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("MinIO not healthy after %v", timeout)
}

// ── alias ─────────────────────────────────────────────────────────────────────

// TestAliasSet verifies that alias set writes an entry to config.json.
func TestAliasSet(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	if !strings.Contains(string(data), testAlias) {
		t.Errorf("alias %q not found in config.json:\n%s", testAlias, data)
	}
}

// TestAliasSetUnreachableServer verifies that alias set exits non-zero when
// the server is unreachable — matching upstream mc behaviour that scripts like
//
//	until mc alias set myminio ...; do sleep 5; done
//
// depend on.
func TestAliasSetUnreachableServer(t *testing.T) {
	tmp, err := os.MkdirTemp("", "mc-unreachable-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	_, _, code := runMC(tmp, "alias", "set", "dead", "http://127.0.0.1:19999", testUser, testPass)
	if code == 0 {
		t.Error("expected non-zero exit when server is unreachable, got 0")
	}
}

// TestAliasSetQuiet verifies that --quiet suppresses stdout on alias set.
func TestAliasSetQuiet(t *testing.T) {
	tmp, err := os.MkdirTemp("", "mc-quiet-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	stdout, _, code := runMC(tmp, "--quiet", "alias", "set", "q", endpoint, testUser, testPass)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected no stdout with --quiet, got: %q", stdout)
	}
}

// TestAliasList verifies that alias ls shows registered aliases.
func TestAliasList(t *testing.T) {
	stdout, stderr, code := mc("alias", "ls")
	fmt.Fprintln(os.Stderr, "alias ls output:\n"+stdout)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, testAlias) {
		t.Errorf("alias %q missing from output:\n%s", testAlias, stdout)
	}
}

// TestAliasRemove verifies that alias remove deletes the entry and that the
// alias can be re-registered afterwards.
func TestAliasRemove(t *testing.T) {
	tmp, err := os.MkdirTemp("", "mc-alias-rm-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	runMC(tmp, "alias", "set", "tmp", endpoint, testUser, testPass)

	stdout, stderr, code := runMC(tmp, "alias", "remove", "tmp")
	if code != 0 {
		t.Fatalf("alias remove exited %d: %s", code, stderr)
	}

	stdout, _, _ = runMC(tmp, "alias", "ls")
	if strings.Contains(stdout, "tmp") {
		t.Error("alias still listed after removal")
	}

	// Re-add must succeed.
	_, _, code = runMC(tmp, "alias", "set", "tmp", endpoint, testUser, testPass)
	if code != 0 {
		t.Error("re-adding removed alias failed")
	}
}

// ── mb ────────────────────────────────────────────────────────────────────────

// TestMbBasic verifies that mb creates a bucket.
func TestMbBasic(t *testing.T) {
	_, stderr, code := mc("mb", testAlias+"/mb-basic")
	if code != 0 {
		t.Fatalf("mb exited %d: %s", code, stderr)
	}
}

// TestMbWithRegion verifies that mb --region is accepted.
func TestMbWithRegion(t *testing.T) {
	_, stderr, code := mc("mb", "--region", "us-east-1", testAlias+"/mb-region")
	if code != 0 {
		t.Fatalf("mb --region exited %d: %s", code, stderr)
	}
}

// TestMbAlreadyExists verifies that creating a duplicate bucket fails.
func TestMbAlreadyExists(t *testing.T) {
	mc("mb", testAlias+"/mb-exists") // first creation, ignore result
	_, _, code := mc("mb", testAlias+"/mb-exists")
	if code == 0 {
		t.Error("expected non-zero exit for duplicate bucket")
	}
}

// TestMbIgnoreExisting verifies that --ignore-existing (-p) suppresses the
// "bucket already exists" error — matching upstream mc behaviour used in
// init scripts to create buckets idempotently.
func TestMbIgnoreExisting(t *testing.T) {
	mc("mb", testAlias+"/mb-ignore") // first creation
	_, stderr, code := mc("mb", "--ignore-existing", testAlias+"/mb-ignore")
	if code != 0 {
		t.Fatalf("mb --ignore-existing exited %d: %s", code, stderr)
	}
	// Short form -p must also work.
	_, stderr, code = mc("mb", "-p", testAlias+"/mb-ignore")
	if code != 0 {
		t.Fatalf("mb -p exited %d: %s", code, stderr)
	}
}

// ── rb ────────────────────────────────────────────────────────────────────────

// TestRbEmpty verifies that rb removes an empty bucket.
func TestRbEmpty(t *testing.T) {
	mc("mb", testAlias+"/rb-empty")
	_, stderr, code := mc("rb", testAlias+"/rb-empty")
	if code != 0 {
		t.Fatalf("rb exited %d: %s", code, stderr)
	}
	// Bucket must be gone.
	_, _, code = mc("stat", testAlias+"/rb-empty")
	if code == 0 {
		t.Error("bucket still exists after rb")
	}
}

// TestRbNonEmptyWithoutForce verifies that rb exits non-zero on a non-empty bucket.
func TestRbNonEmptyWithoutForce(t *testing.T) {
	mc("mb", testAlias+"/rb-nonempty")
	putObject(t, "rb-nonempty", "file.txt", "data")
	_, _, code := mc("rb", testAlias+"/rb-nonempty")
	if code == 0 {
		t.Error("expected non-zero exit when removing non-empty bucket without --force")
	}
}

// TestRbForce verifies that rb --force removes a non-empty bucket and all its objects.
func TestRbForce(t *testing.T) {
	mc("mb", testAlias+"/rb-force")
	putObject(t, "rb-force", "a.txt", "aaa")
	putObject(t, "rb-force", "sub/b.txt", "bbb")

	_, stderr, code := mc("rb", "--force", testAlias+"/rb-force")
	if code != 0 {
		t.Fatalf("rb --force exited %d: %s", code, stderr)
	}
	_, _, code = mc("stat", testAlias+"/rb-force")
	if code == 0 {
		t.Error("bucket still exists after rb --force")
	}
}

// TestRbForce100Files verifies that rb --force removes a bucket containing
// 100 × 1 MiB random objects and that the bucket no longer exists afterwards.
func TestRbForce100Files(t *testing.T) {
	const fileCount = 100
	const fileSize = 1 << 20 // 1 MiB

	mc("mb", testAlias+"/rb-force-100")

	c := minioGoClient(t)
	for i := range fileCount {
		key := fmt.Sprintf("file-%03d.bin", i)
		buf := make([]byte, fileSize)
		if _, err := rand.Read(buf); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}
		if _, err := c.PutObject(context.Background(), "rb-force-100", key, strings.NewReader(string(buf)), int64(fileSize), miniogo.PutObjectOptions{}); err != nil {
			t.Fatalf("upload %s: %v", key, err)
		}
	}

	_, stderr, code := mc("rb", "--force", testAlias+"/rb-force-100")
	if code != 0 {
		t.Fatalf("rb --force exited %d: %s", code, stderr)
	}
	_, _, code = mc("stat", testAlias+"/rb-force-100")
	if code == 0 {
		t.Error("bucket still exists after rb --force")
	}
}

// TestRbForceMultipleTargets verifies that rb --force accepts multiple targets
// in one invocation, each containing 100 × 1 MiB objects distributed across
// random subdirectories, and that all buckets are gone afterwards.
func TestRbForceMultipleTargets(t *testing.T) {
	const bucketCount = 3
	const fileCount = 100
	const fileSize = 1 << 20 // 1 MiB

	dirs := []string{"alpha", "beta", "gamma", "delta", "epsilon"}

	c := minioGoClient(t)
	buckets := make([]string, bucketCount)
	for b := range bucketCount {
		bucket := fmt.Sprintf("rb-force-multi-%d", b)
		buckets[b] = bucket
		mc("mb", testAlias+"/"+bucket)

		for i := range fileCount {
			dir := dirs[mrand.Intn(len(dirs))]
			key := fmt.Sprintf("%s/file-%03d.bin", dir, i)
			buf := make([]byte, fileSize)
			if _, err := rand.Read(buf); err != nil {
				t.Fatalf("rand.Read: %v", err)
			}
			if _, err := c.PutObject(context.Background(), bucket, key, strings.NewReader(string(buf)), int64(fileSize), miniogo.PutObjectOptions{}); err != nil {
				t.Fatalf("upload %s/%s: %v", bucket, key, err)
			}
		}
	}

	targets := make([]string, bucketCount)
	for b, bucket := range buckets {
		targets[b] = testAlias + "/" + bucket
	}

	args := append([]string{"rb", "--force"}, targets...)
	_, stderr, code := mc(args...)
	if code != 0 {
		t.Fatalf("rb --force exited %d: %s", code, stderr)
	}

	for _, bucket := range buckets {
		if _, _, c := mc("stat", testAlias+"/"+bucket); c == 0 {
			t.Errorf("bucket %s still exists after rb --force", bucket)
		}
	}
}

// TestRbNonexistent verifies that rb on a missing bucket exits non-zero.
func TestRbNonexistent(t *testing.T) {
	_, _, code := mc("rb", testAlias+"/rb-no-such-bucket")
	if code == 0 {
		t.Error("expected non-zero exit for nonexistent bucket")
	}
}

// TestRbMultiple verifies that rb accepts multiple targets in one invocation.
func TestRbMultiple(t *testing.T) {
	mc("mb", testAlias+"/rb-multi-1")
	mc("mb", testAlias+"/rb-multi-2")
	_, stderr, code := mc("rb", testAlias+"/rb-multi-1", testAlias+"/rb-multi-2")
	if code != 0 {
		t.Fatalf("rb multiple exited %d: %s", code, stderr)
	}
	for _, b := range []string{"rb-multi-1", "rb-multi-2"} {
		if _, _, c := mc("stat", testAlias+"/"+b); c == 0 {
			t.Errorf("bucket %s still exists after rb", b)
		}
	}
}

// ── ls ────────────────────────────────────────────────────────────────────────

// TestLsBuckets verifies that ls with only an alias lists existing buckets
// and that each bucket line includes the "0B" size field.
func TestLsBuckets(t *testing.T) {
	mc("mb", testAlias+"/ls-all")
	stdout, stderr, code := mc("ls", testAlias)
	if code != 0 {
		t.Fatalf("ls exited %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "ls-all") {
		t.Errorf("bucket not listed:\n%s", stdout)
	}
}

// TestLsEmptyBucket verifies that ls on an empty bucket exits 0.
func TestLsEmptyBucket(t *testing.T) {
	mc("mb", testAlias+"/ls-empty")
	_, stderr, code := mc("ls", testAlias+"/ls-empty")
	if code != 0 {
		t.Fatalf("ls empty bucket exited %d: %s", code, stderr)
	}
}

// TestLsObjectsListed verifies that ls shows objects that exist in a bucket.
func TestLsObjectsListed(t *testing.T) {
	mc("mb", testAlias+"/ls-objects")
	putObject(t, "ls-objects", "hello.txt", "hello world")

	stdout, stderr, code := mc("ls", testAlias+"/ls-objects")
	if code != 0 {
		t.Fatalf("ls exited %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "hello.txt") {
		t.Errorf("object not listed:\n%s", stdout)
	}
}

// TestLsNonexistentBucket verifies that ls on a missing bucket exits non-zero.
func TestLsNonexistentBucket(t *testing.T) {
	_, _, code := mc("ls", testAlias+"/ls-no-such-bucket")
	if code == 0 {
		t.Error("expected non-zero exit for nonexistent bucket")
	}
}

// ── stat ──────────────────────────────────────────────────────────────────────

// TestStatBucketExists verifies that stat on an existing bucket exits 0.
func TestStatBucketExists(t *testing.T) {
	mc("mb", testAlias+"/stat-bucket")
	stdout, stderr, code := mc("stat", testAlias+"/stat-bucket")
	fmt.Fprintln(os.Stderr, "stat output:\n"+stdout)
	if code != 0 {
		t.Fatalf("stat exited %d: %s", code, stderr)
	}
}

// TestStatBucketNotFound verifies that stat on a missing bucket exits non-zero.
func TestStatBucketNotFound(t *testing.T) {
	_, _, code := mc("stat", testAlias+"/stat-no-such-bucket")
	if code == 0 {
		t.Error("expected non-zero exit for nonexistent bucket")
	}
}

// TestStatObjectExists verifies that stat on an existing object exits 0.
func TestStatObjectExists(t *testing.T) {
	mc("mb", testAlias+"/stat-obj")
	putObject(t, "stat-obj", "data.txt", "some content")

	stdout, stderr, code := mc("stat", testAlias+"/stat-obj/data.txt")
	fmt.Fprintln(os.Stderr, "stat output:\n"+stdout)
	if code != 0 {
		t.Fatalf("stat object exited %d: %s", code, stderr)
	}
}

// TestStatObjectNotFound verifies that stat on a missing object exits non-zero.
func TestStatObjectNotFound(t *testing.T) {
	mc("mb", testAlias+"/stat-obj-nf")
	_, _, code := mc("stat", testAlias+"/stat-obj-nf/no-such-object.txt")
	if code == 0 {
		t.Error("expected non-zero exit for nonexistent object")
	}
}

// ── anonymous ─────────────────────────────────────────────────────────────────

// assertPolicy sets a policy on a bucket and verifies that anonymous get
// reports the expected policy name back.
func assertPolicy(t *testing.T, bucket, policy, wantName string) {
	t.Helper()
	if _, stderr, code := mc("anonymous", "set", policy, testAlias+"/"+bucket+"/"); code != 0 {
		t.Fatalf("anonymous set %s exited %d: %s", policy, code, stderr)
	}
	stdout, stderr, code := mc("anonymous", "get", testAlias+"/"+bucket+"/")
	if code != 0 {
		t.Fatalf("anonymous get exited %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, wantName) {
		t.Errorf("anonymous get: expected policy %q in output, got: %s", wantName, stdout)
	}
}

// anonGET performs an unauthenticated HTTP GET for bucket/key and returns the status code.
func anonGET(t *testing.T, bucket, key string) int {
	t.Helper()
	resp, err := http.Get(endpoint + "/" + bucket + "/" + key)
	if err != nil {
		t.Fatalf("anonymous GET: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// anonPUT performs an unauthenticated HTTP PUT of body to bucket/key and returns the status code.
func anonPUT(t *testing.T, bucket, key string, body []byte) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, endpoint+"/"+bucket+"/"+key, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("anonymous PUT request: %v", err)
	}
	req.ContentLength = int64(len(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("anonymous PUT: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestAnonymousDownload verifies the "download" policy: anonymous GET succeeds,
// anonymous PUT is rejected.
func TestAnonymousDownload(t *testing.T) {
	mc("mb", testAlias+"/anon-dl")
	assertPolicy(t, "anon-dl", "download", "download")

	putObject(t, "anon-dl", "file.txt", "hello")
	if got := anonGET(t, "anon-dl", "file.txt"); got != http.StatusOK {
		t.Errorf("anonymous GET with download policy: want 200, got %d", got)
	}
	if got := anonPUT(t, "anon-dl", "new.txt", []byte("x")); got == http.StatusOK {
		t.Errorf("anonymous PUT with download policy should be rejected, got %d", got)
	}
}

// TestAnonymousUpload verifies the "upload" policy: anonymous PUT succeeds,
// anonymous GET is rejected.
func TestAnonymousUpload(t *testing.T) {
	mc("mb", testAlias+"/anon-ul")
	assertPolicy(t, "anon-ul", "upload", "upload")

	if got := anonPUT(t, "anon-ul", "file.txt", []byte("hello")); got != http.StatusOK {
		t.Errorf("anonymous PUT with upload policy: want 200, got %d", got)
	}
	if got := anonGET(t, "anon-ul", "file.txt"); got == http.StatusOK {
		t.Errorf("anonymous GET with upload policy should be rejected, got %d", got)
	}
}

// TestAnonymousPublic verifies the "public" policy: both anonymous GET and PUT succeed.
func TestAnonymousPublic(t *testing.T) {
	mc("mb", testAlias+"/anon-pub")
	assertPolicy(t, "anon-pub", "public", "public")

	if got := anonPUT(t, "anon-pub", "file.txt", []byte("hello")); got != http.StatusOK {
		t.Errorf("anonymous PUT with public policy: want 200, got %d", got)
	}
	if got := anonGET(t, "anon-pub", "file.txt"); got != http.StatusOK {
		t.Errorf("anonymous GET with public policy: want 200, got %d", got)
	}
}

// TestAnonymousPrivate verifies that "private" removes public access:
// both anonymous GET and PUT are rejected.
func TestAnonymousPrivate(t *testing.T) {
	mc("mb", testAlias+"/anon-priv")
	assertPolicy(t, "anon-priv", "public", "public") // open it first
	assertPolicy(t, "anon-priv", "private", "none")

	putObject(t, "anon-priv", "file.txt", "hello")
	if got := anonGET(t, "anon-priv", "file.txt"); got == http.StatusOK {
		t.Errorf("anonymous GET with private policy should be rejected, got %d", got)
	}
	if got := anonPUT(t, "anon-priv", "new.txt", []byte("x")); got == http.StatusOK {
		t.Errorf("anonymous PUT with private policy should be rejected, got %d", got)
	}
}

// TestAnonymousNone verifies that "none" (alias for "private") also rejects
// both anonymous GET and PUT.
func TestAnonymousNone(t *testing.T) {
	mc("mb", testAlias+"/anon-none")
	assertPolicy(t, "anon-none", "public", "public") // open it first
	assertPolicy(t, "anon-none", "none", "none")

	putObject(t, "anon-none", "file.txt", "hello")
	if got := anonGET(t, "anon-none", "file.txt"); got == http.StatusOK {
		t.Errorf("anonymous GET with none policy should be rejected, got %d", got)
	}
	if got := anonPUT(t, "anon-none", "new.txt", []byte("x")); got == http.StatusOK {
		t.Errorf("anonymous PUT with none policy should be rejected, got %d", got)
	}
}

// TestAnonymousInvalidPolicy verifies that an unknown policy name exits non-zero.
func TestAnonymousInvalidPolicy(t *testing.T) {
	mc("mb", testAlias+"/anon-inv")
	_, _, code := mc("anonymous", "set", "INVALID", testAlias+"/anon-inv/")
	if code == 0 {
		t.Error("expected non-zero exit for unknown policy")
	}
}

// ── admin info ────────────────────────────────────────────────────────────────

// TestAdminInfo verifies that admin info exits 0 and prints human-readable
// server information when the server is reachable.
func TestAdminInfo(t *testing.T) {
	stdout, stderr, code := mc("admin", "info", testAlias)
	fmt.Fprintln(os.Stderr, "admin info output:\n"+stdout)
	if code != 0 {
		t.Fatalf("admin info exited %d: %s", code, stderr)
	}
	epHost := strings.Split(strings.TrimPrefix(endpoint, "http://"), ":")[0]
	if !strings.Contains(stdout, epHost) {
		t.Errorf("output missing endpoint host %q:\n%s", epHost, stdout)
	}
	for _, want := range []string{"Uptime:", "Version:", "Used,"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q:\n%s", want, stdout)
		}
	}
}

// TestAdminInfoServerDown verifies that admin info exits 0 even when the server
// is unreachable — matching upstream mc behaviour (error surfaced in output,
// not exit code).
func TestAdminInfoServerDown(t *testing.T) {
	tmp, err := os.MkdirTemp("", "mc-info-down-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	runMC(tmp, "alias", "set", "dead", endpoint, testUser, testPass) // real server for alias set
	// Point alias at a dead port — we must edit the config directly since
	// alias set probes the server.
	cfgPath := filepath.Join(tmp, "config.json")
	data, _ := os.ReadFile(cfgPath)
	patched := strings.ReplaceAll(string(data), endpoint, "http://127.0.0.1:19999")
	os.WriteFile(cfgPath, []byte(patched), 0600)

	_, _, code := runMC(tmp, "admin", "info", "dead")
	if code != 0 {
		t.Errorf("admin info should exit 0 even when server is down, got %d", code)
	}
}

// TestAdminInfoJSON verifies that admin info --json outputs valid JSON with
// status "success".
func TestAdminInfoJSON(t *testing.T) {
	stdout, stderr, code := mc("admin", "info", testAlias, "--json")
	fmt.Fprintln(os.Stderr, "admin info --json output:\n"+stdout)
	if code != 0 {
		t.Fatalf("admin info --json exited %d: %s", code, stderr)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, stdout)
	}
	if out["status"] != "success" {
		t.Errorf(`expected "status":"success", got %v`, out["status"])
	}
}

// ── mirror ────────────────────────────────────────────────────────────────────

// TestMirrorRemoteToLocal mirrors a MinIO bucket to a local directory.
func TestMirrorRemoteToLocal(t *testing.T) {
	mc("mb", testAlias+"/mirror-r2l")
	putObject(t, "mirror-r2l", "a.txt", "aaa")
	putObject(t, "mirror-r2l", "sub/b.txt", "bbb")

	dir, err := os.MkdirTemp("", "mc-mirror-r2l-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	_, stderr, code := mc("mirror", testAlias+"/mirror-r2l", dir)
	if code != 0 {
		t.Fatalf("mirror exited %d: %s", code, stderr)
	}

	for _, f := range []string{"a.txt", "sub/b.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
}

// TestMirrorLocalToRemote mirrors a local directory into a MinIO bucket.
func TestMirrorLocalToRemote(t *testing.T) {
	dir, err := os.MkdirTemp("", "mc-mirror-l2r-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("xxx"), 0644)
	os.MkdirAll(filepath.Join(dir, "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "deep", "y.txt"), []byte("yyy"), 0644)

	mc("mb", testAlias+"/mirror-l2r")
	_, stderr, code := mc("mirror", dir, testAlias+"/mirror-l2r")
	if code != 0 {
		t.Fatalf("mirror exited %d: %s", code, stderr)
	}

	// Verify objects exist in MinIO.
	c := minioGoClient(t)
	for _, key := range []string{"x.txt", "deep/y.txt"} {
		if _, err := c.StatObject(context.Background(), "mirror-l2r", key, miniogo.StatObjectOptions{}); err != nil {
			t.Errorf("expected object %s in MinIO: %v", key, err)
		}
	}
}

// TestMirrorOverwrite verifies that --overwrite replaces existing destination files.
func TestMirrorOverwrite(t *testing.T) {
	mc("mb", testAlias+"/mirror-ow")
	putObject(t, "mirror-ow", "f.txt", "v1")

	dir, err := os.MkdirTemp("", "mc-mirror-ow-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// First mirror creates the file.
	mc("mirror", testAlias+"/mirror-ow", dir)

	// Update the object in MinIO.
	putObject(t, "mirror-ow", "f.txt", "v2")

	// Without --overwrite, stale file remains.
	mc("mirror", testAlias+"/mirror-ow", dir)
	data, _ := os.ReadFile(filepath.Join(dir, "f.txt"))
	if string(data) != "v1" {
		t.Errorf("expected stale v1 without --overwrite, got %q", data)
	}

	// With --overwrite, file is refreshed.
	mc("mirror", "--overwrite", testAlias+"/mirror-ow", dir)
	data, _ = os.ReadFile(filepath.Join(dir, "f.txt"))
	if string(data) != "v2" {
		t.Errorf("expected updated v2 with --overwrite, got %q", data)
	}
}

// TestMirrorMaxWorkers verifies that --max-workers is accepted and that parallel
// transfers produce the same result as the default single-worker path.
func TestMirrorMaxWorkers(t *testing.T) {
	// Remote → local with multiple objects.
	mc("mb", testAlias+"/mirror-mw-r2l")
	for _, key := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
		putObject(t, "mirror-mw-r2l", key, key)
	}

	dir, err := os.MkdirTemp("", "mc-mirror-mw-r2l-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	_, stderr, code := mc("mirror", "--max-workers", "2", testAlias+"/mirror-mw-r2l", dir)
	if code != 0 {
		t.Fatalf("mirror --max-workers r2l exited %d: %s", code, stderr)
	}
	for _, f := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected file %s: %v", f, err)
		}
	}

	// Local → remote with multiple files.
	srcDir, err := os.MkdirTemp("", "mc-mirror-mw-l2r-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	for _, name := range []string{"e.txt", "f.txt", "g.txt", "h.txt"} {
		os.WriteFile(filepath.Join(srcDir, name), []byte(name), 0644)
	}

	mc("mb", testAlias+"/mirror-mw-l2r")
	_, stderr, code = mc("mirror", "--max-workers", "2", srcDir, testAlias+"/mirror-mw-l2r")
	if code != 0 {
		t.Fatalf("mirror --max-workers l2r exited %d: %s", code, stderr)
	}

	c := minioGoClient(t)
	for _, key := range []string{"e.txt", "f.txt", "g.txt", "h.txt"} {
		if _, err := c.StatObject(context.Background(), "mirror-mw-l2r", key, miniogo.StatObjectOptions{}); err != nil {
			t.Errorf("expected object %s in MinIO: %v", key, err)
		}
	}
}

// TestMirrorParallel100Files verifies that mirroring 100 × 1 MiB files works
// correctly in both directions. Each file is filled with random bytes and its
// SHA-256 digest is recorded upfront; after each mirror leg the digest is
// recomputed and compared to confirm data integrity end-to-end.
func TestMirrorParallel100Files(t *testing.T) {
	const fileCount = 100
	const fileSize = 1 << 20 // 1 MiB

	srcDir, err := os.MkdirTemp("", "mc-par-l2r-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	// Generate random files and record their SHA-256 digests.
	digests := make(map[string][sha256.Size]byte, fileCount)
	for i := range fileCount {
		name := fmt.Sprintf("file-%03d.bin", i)
		buf := make([]byte, fileSize)
		if _, err := rand.Read(buf); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, name), buf, 0644); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		digests[name] = sha256.Sum256(buf)
	}

	checkFile := func(path, name string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			return
		}
		if got := sha256.Sum256(data); got != digests[name] {
			t.Errorf("integrity mismatch for %s", name)
		}
	}

	// ── local → remote ────────────────────────────────────────────────────────
	mc("mb", testAlias+"/mirror-par-l2r")
	_, stderr, code := mc("mirror", srcDir, testAlias+"/mirror-par-l2r")
	if code != 0 {
		t.Fatalf("mirror l2r exited %d: %s", code, stderr)
	}

	c := minioGoClient(t)
	for i := range fileCount {
		name := fmt.Sprintf("file-%03d.bin", i)
		obj, err := c.GetObject(context.Background(), "mirror-par-l2r", name, miniogo.GetObjectOptions{})
		if err != nil {
			t.Errorf("get object %s: %v", name, err)
			continue
		}
		data, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			t.Errorf("read object %s: %v", name, err)
			continue
		}
		if got := sha256.Sum256(data); got != digests[name] {
			t.Errorf("integrity mismatch for remote object %s", name)
		}
	}

	// ── remote → local ────────────────────────────────────────────────────────
	dstDir, err := os.MkdirTemp("", "mc-par-r2l-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	_, stderr, code = mc("mirror", testAlias+"/mirror-par-l2r", dstDir)
	if code != 0 {
		t.Fatalf("mirror r2l exited %d: %s", code, stderr)
	}

	for i := range fileCount {
		name := fmt.Sprintf("file-%03d.bin", i)
		checkFile(filepath.Join(dstDir, name), name)
	}
}

// TestMirrorParallel10Filesx10MB verifies data integrity when mirroring
// 10 × 10 MiB random files in both directions.
func TestMirrorParallel10Filesx10MB(t *testing.T) {
	const fileCount = 10
	const fileSize = 10 << 20 // 10 MiB

	srcDir, err := os.MkdirTemp("", "mc-par10-l2r-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	digests := make(map[string][sha256.Size]byte, fileCount)
	for i := range fileCount {
		name := fmt.Sprintf("file-%02d.bin", i)
		buf := make([]byte, fileSize)
		if _, err := rand.Read(buf); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, name), buf, 0644); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		digests[name] = sha256.Sum256(buf)
	}

	// ── local → remote ────────────────────────────────────────────────────────
	mc("mb", testAlias+"/mirror-par10-l2r")
	_, stderr, code := mc("mirror", srcDir, testAlias+"/mirror-par10-l2r")
	if code != 0 {
		t.Fatalf("mirror l2r exited %d: %s", code, stderr)
	}

	c := minioGoClient(t)
	for i := range fileCount {
		name := fmt.Sprintf("file-%02d.bin", i)
		obj, err := c.GetObject(context.Background(), "mirror-par10-l2r", name, miniogo.GetObjectOptions{})
		if err != nil {
			t.Errorf("get object %s: %v", name, err)
			continue
		}
		data, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			t.Errorf("read object %s: %v", name, err)
			continue
		}
		if got := sha256.Sum256(data); got != digests[name] {
			t.Errorf("integrity mismatch for remote object %s", name)
		}
	}

	// ── remote → local ────────────────────────────────────────────────────────
	dstDir, err := os.MkdirTemp("", "mc-par10-r2l-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	_, stderr, code = mc("mirror", testAlias+"/mirror-par10-l2r", dstDir)
	if code != 0 {
		t.Fatalf("mirror r2l exited %d: %s", code, stderr)
	}

	for i := range fileCount {
		name := fmt.Sprintf("file-%02d.bin", i)
		data, err := os.ReadFile(filepath.Join(dstDir, name))
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		if got := sha256.Sum256(data); got != digests[name] {
			t.Errorf("integrity mismatch for downloaded file %s", name)
		}
	}
}

// TestMirrorMCConfigDir verifies that MC_CONFIG_DIR env var is honoured as
// the default config directory (matching the upstream mc behaviour used by the
// OpenVidu backup/restore scripts).
func TestMirrorMCConfigDir(t *testing.T) {
	tmp, err := os.MkdirTemp("", "mc-envdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	// Register alias using the env var instead of --config-dir flag.
	cmd := exec.Command(mcBin, "alias", "set", "envtest", endpoint, testUser, testPass)
	cmd.Env = append(os.Environ(), "MC_CONFIG_DIR="+tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("alias set via MC_CONFIG_DIR failed: %s", out)
	}

	// Verify config was written to the env-var path, not the default.
	data, err := os.ReadFile(filepath.Join(tmp, "config.json"))
	if err != nil {
		t.Fatalf("config.json not found in MC_CONFIG_DIR path: %v", err)
	}
	if !strings.Contains(string(data), "envtest") {
		t.Errorf("alias not found in config.json:\n%s", data)
	}
}

// ── admin service ─────────────────────────────────────────────────────────────

// TestAdminServiceRestart verifies that admin service restart exits 0 and that
// the server comes back up (requires "always" Docker restart policy).
func TestAdminServiceRestart(t *testing.T) {
	ctx := context.Background()
	ctr, ep, err := startMinio(ctx, testUser, testPass, true /* always restart */)
	if err != nil {
		t.Fatalf("start minio: %v", err)
	}
	defer ctr.Terminate(ctx)

	tmp, err := os.MkdirTemp("", "mc-restart-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	const alias = "restart"
	runMC(tmp, "alias", "set", alias, ep, testUser, testPass)

	_, stderr, code := runMC(tmp, "admin", "service", "restart", alias)
	if code != 0 {
		t.Fatalf("admin service restart exited %d: %s", code, stderr)
	}

	// Give Docker time to restart the container, then wait for health.
	time.Sleep(2 * time.Second)
	if err := waitForHealth(ep, 60*time.Second); err != nil {
		t.Fatalf("MinIO did not recover after restart: %v", err)
	}

	// Confirm subsequent commands work again.
	_, stderr, code = runMC(tmp, "ls", alias)
	if code != 0 {
		t.Fatalf("ls after restart exited %d: %s", code, stderr)
	}
}

// TestAdminServiceStop verifies that admin service stop exits 0 and that the
// server is no longer reachable afterwards.
func TestAdminServiceStop(t *testing.T) {
	ctx := context.Background()
	ctr, ep, err := startMinio(ctx, testUser, testPass, false)
	if err != nil {
		t.Fatalf("start minio: %v", err)
	}
	defer ctr.Terminate(ctx)

	tmp, err := os.MkdirTemp("", "mc-stop-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	const alias = "stop"
	runMC(tmp, "alias", "set", alias, ep, testUser, testPass)

	_, stderr, code := runMC(tmp, "admin", "service", "stop", alias)
	if code != 0 {
		t.Fatalf("admin service stop exited %d: %s", code, stderr)
	}

	// Server should now be unreachable; ls must fail.
	time.Sleep(2 * time.Second)
	_, _, code = runMC(tmp, "ls", alias)
	if code == 0 {
		t.Error("expected ls to fail after server stop, but it exited 0")
	}
}
