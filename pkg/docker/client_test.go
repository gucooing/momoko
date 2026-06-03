package docker

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	v1 "momoko/api/gen/v1"
)

func TestDockerConfigTestIgnoresEnabled(t *testing.T) {
	server := httptest.NewServer(fakeDockerAPIHandler())
	defer server.Close()

	host := "tcp://" + strings.TrimPrefix(server.URL, "http://")
	status, err := (&Manager{}).Test(context.Background(), &v1.DockerConfigInfo{
		Enabled:               false,
		Host:                  host,
		RequestTimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatalf("Test returned error: %v", err)
	}
	if !status.Connected {
		t.Fatalf("expected docker config test to connect when disabled, got status: %+v", status)
	}
}

func TestUnixDockerHostUsesSocketTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket listener is platform-specific")
	}

	dir, err := os.MkdirTemp(os.TempDir(), "momoko-docker-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	socketPath := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	server := &http.Server{Handler: fakeDockerAPIHandler()}
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Close()

	cli, err := newClient(&v1.DockerConfigInfo{
		Enabled:               true,
		Host:                  "unix://" + socketPath,
		RequestTimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	if _, err := cli.Ping(context.Background()); err != nil {
		t.Fatalf("expected ping over unix socket to succeed: %v", err)
	}
}

func fakeDockerAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_ping":
			w.Header().Set("Api-Version", "1.51")
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/info"):
			writeJSON(w, map[string]any{
				"ID":                "test-engine",
				"Name":              "test-docker",
				"ServerVersion":     "28.5.2",
				"OperatingSystem":   "Linux",
				"OSType":            "linux",
				"Architecture":      "x86_64",
				"DockerRootDir":     "/var/lib/docker",
				"Containers":        1,
				"ContainersRunning": 1,
				"Images":            1,
				"Driver":            "overlay2",
				"NCPU":              2,
				"MemTotal":          1024,
			})
		case strings.HasSuffix(r.URL.Path, "/version"):
			writeJSON(w, map[string]any{
				"Version":    "28.5.2",
				"ApiVersion": "1.51",
			})
		default:
			http.NotFound(w, r)
		}
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
