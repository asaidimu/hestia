package update_e2e

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/asaidimu/updater"
)

const (
	e2eAppName  = "hestia-e2e"
	e2eToken    = "e2e-token"
	e2eClientID = "e2e-client"
	initialVer  = "1.0.0"
	latestVer   = "1.1.0"
)

// TestSelfUpdateEndToEnd drives the full update loop against two built
// binaries and a local update server:
//
//	build v1.0.0 and v1.1.0 -> boot v1.0.0 with the server provider ->
//	check+stage (downloads v1.1.0 into DataDir) -> apply (spawns the staged
//	binary and exits) -> new process swaps itself over the original executable
//	-> verify the restarted server reports version 1.1.0 and the binary file at
//	the original path is now the v1.1.0 artifact.
func TestSelfUpdateEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping self-update E2E in -short mode")
	}

	repoRoot := repoRoot(t)
	work := t.TempDir()
	binV100 := filepath.Join(work, "app-"+initialVer)
	binV110 := filepath.Join(work, "app-"+latestVer)
	buildTestServer(t, repoRoot, binV100, initialVer)
	buildTestServer(t, repoRoot, binV110, latestVer)

	// In-process update server: checks the client version and serves the new
	// binary with an RSA-signed response (mirrors updater's wireUpdateInfo).
	srv, pubKey := startUpdateServer(t, binV110)
	defer srv.Close()

	port := freePort(t)
	base := "http://127.0.0.1:" + port
	dbPath := filepath.Join(work, "e2e.db")

	_ = spawnApp(t, binV100, port, srv.URL, pubPEM(t, pubKey), dbPath)

	waitForStatus(t, base, initialVer)

	// Check: discovers and stages v1.1.0.
	check := postJSON(t, base+"/api/system/updates/check/create")
	assertField(t, check, "checked", true)
	assertField(t, check, "staged", true)
	assertField(t, check, "version", latestVer)

	// Status now reflects the staged update.
	status := getJSON(t, base+"/api/system/updates/status/get")
	assertField(t, status, "staged_version", latestVer)
	assertField(t, status, "prepared", true)

	// Changelog surfaces the staged release notes.
	changelog := getJSON(t, base+"/api/system/updates/changelog/get")
	assertField(t, changelog, "version", latestVer)
	assertField(t, changelog, "changelog", "E2E release")

	// Apply: spawns the staged binary and the old process exits (the request
	// dies mid-flight, so the response is intentionally ignored).
	_, _, _ = postJSONAllowErr(base + "/api/system/updates/update/apply")

	// The swapped process restarts as v1.1.0 on the same port.
	waitForStatus(t, base, latestVer)

	// The original executable path now holds the v1.1.0 binary.
	if !filesEqual(binV100, binV110) {
		t.Fatal("original executable was not replaced by the v1.1.0 binary")
	}

	// And the new process reports a clean state (nothing staged).
	status = getJSON(t, base+"/api/system/updates/status/get")
	assertField(t, status, "version", latestVer)
	assertField(t, status, "staged_version", "")
	assertField(t, status, "prepared", false)
}

// --- harness ---

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(dir))) // tests/e2e/update -> repo root
	info, err := os.Stat(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("cannot locate repo root from %s: %v", dir, err)
	}
	if info.IsDir() {
		t.Fatalf("%s/go.mod is a directory", root)
	}
	return root
}

func buildTestServer(t *testing.T, repoRoot, out, version string) {
	t.Helper()
	cmd := exec.Command("go", "build",
		"-o", out,
		"-ldflags", "-X main.version="+version,
		"github.com/asaidimu/hestia/cmd/test-server",
	)
	cmd.Dir = repoRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build v%s: %v\n%s", version, err, b)
	}
}

// wireUpdateInfo mirrors updater's (unexported) server response payload. The
// RSA signature is computed over the JSON encoding of this struct with the
// signature field cleared — identical bytes to updater.verifySignature.
type wireUpdateInfo struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	TTL       int    `json:"ttl"`
	Changelog string `json:"changelog"`
	Checksum  string `json:"checksum"`
	Signature string `json:"signature"`
}

// startUpdateServer hosts POST /api/update (check contract) and GET /binary
// (the v1.1.0 artifact).
func startUpdateServer(t *testing.T, binV110 string) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	binBytes, err := os.ReadFile(binV110)
	if err != nil {
		t.Fatalf("read v1.1.0 binary: %v", err)
	}
	sum := sha256.Sum256(binBytes)
	checksum := "SHA256:" + hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req updater.CheckRequestBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Token != e2eToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if req.Name != e2eAppName {
			http.Error(w, "unknown app", http.StatusNotFound)
			return
		}
		cur, err := semver.NewVersion(strings.TrimPrefix(req.Version, "v"))
		if err != nil {
			http.Error(w, "bad version", http.StatusBadRequest)
			return
		}
		target, _ := semver.NewVersion(latestVer)
		if !target.GreaterThan(cur) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		wire := wireUpdateInfo{
			Version:   latestVer,
			URL:       "/binary",
			Changelog: "E2E release",
			Checksum:  checksum,
		}
		wire.Signature = signUpdate(t, key, &wire)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&wire)
	})
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(binBytes)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, key
}

func signUpdate(t *testing.T, key *rsa.PrivateKey, wire *wireUpdateInfo) string {
	t.Helper()
	sig := wire.Signature
	wire.Signature = ""
	defer func() { wire.Signature = sig }()
	data, err := json.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal wire: %v", err)
	}
	hash := sha256.Sum256(data)
	s, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return base64.StdEncoding.EncodeToString(s)
}

// pubPEM renders an RSA public key as a PEM PKIX block for the
// UPDATE_SERVER_PUBLIC_KEY env var.
func pubPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	b, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b}))
}

func spawnApp(t *testing.T, bin, port, srvURL, pubKeyPEM, dbPath string) *exec.Cmd {
	t.Helper()
	env := append(os.Environ(),
		"PORT="+port,
		"DB_PATH="+dbPath,
		"UPDATE_ENABLED=true",
		"UPDATE_SERVER_URL="+srvURL,
		"UPDATE_SERVER_APP_NAME="+e2eAppName,
		"UPDATE_SERVER_CLIENT_TOKEN="+e2eToken,
		"UPDATE_SERVER_CLIENT_ID="+e2eClientID,
		"UPDATE_SERVER_PUBLIC_KEY="+pubKeyPEM,
	)
	cmd := exec.Command(bin)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start app: %v", err)
	}
	// Reap the old process as soon as it exits. updater's swap waits for the
	// old PID via signal(pid,0), which sees an unreaped zombie as still
	// running — so the E2E must call Wait promptly or the swap times out.
	go cmd.Wait()
	t.Cleanup(func() { killProcessGroup(cmd) })
	return cmd
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()
	return port
}

// waitForStatus polls the status endpoint until it serves the expected
// version, with a generous timeout for boots and the restart.
func waitForStatus(t *testing.T, base, wantVersion string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		body, status, err := getJSONAllowErr(base + "/api/system/updates/status/get")
		if err == nil && status == http.StatusOK {
			if v, ok := body["version"].(string); ok && v == wantVersion {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for version %q", wantVersion)
}

func getJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	body, status, err := getJSONAllowErr(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	if status != http.StatusOK {
		t.Fatalf("GET %s: status %d, body %v", url, status, body)
	}
	return body
}

func getJSONAllowErr(url string) (map[string]any, int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var m map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode %q: %w", data, err)
		}
	}
	// hestia wraps single-document responses in a {"data":{...}} envelope.
	if inner, ok := m["data"].(map[string]any); ok {
		m = inner
	}
	return m, resp.StatusCode, nil
}

func postJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	body, status, err := postJSONAllowErr(url)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		t.Fatalf("POST %s: status %d, body %v", url, status, body)
	}
	return body
}

func postJSONAllowErr(url string) (map[string]any, int, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var m map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode %q: %w", data, err)
		}
	}
	if inner, ok := m["data"].(map[string]any); ok {
		m = inner
	}
	return m, resp.StatusCode, nil
}

func assertField(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	got, ok := body[key]
	if !ok {
		t.Fatalf("response missing %q; body: %v", key, body)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%q = %v (%T), want %v; body: %v", key, got, got, want, body)
	}
}

func filesEqual(a, b string) bool {
	fa, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	fb, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(fa, fb)
}