package modelmanager

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type rewriteHostTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.base.Scheme
	cloned.URL.Host = t.base.Host
	return t.rt.RoundTrip(cloned)
}

func newTestHFProvider(serverURL string) *HuggingFaceProvider {
	base, _ := url.Parse(serverURL)
	provider := NewHuggingFaceProvider("")
	provider.client = &http.Client{
		Timeout: 5 * time.Second,
		Transport: rewriteHostTransport{
			base: base,
			rt:   http.DefaultTransport,
		},
	}
	return provider
}

func TestFilterDownloadFiles_NoHintReturnsAllFilesSorted(t *testing.T) {
	files := []HFModelFile{
		{Type: "file", Path: "b/b.txt"},
		{Type: "file", Path: "a/a.safetensors"},
		{Type: "directory", Path: "c"},
	}

	got, err := filterDownloadFiles(files, "")
	if err != nil {
		t.Fatalf("filterDownloadFiles returned error: %v", err)
	}

	paths := make([]string, 0, len(got))
	for _, f := range got {
		paths = append(paths, f.Path)
	}

	want := []string{"a/a.safetensors", "b/b.txt"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("unexpected file list, got %v want %v", paths, want)
	}
}

func TestHuggingFaceDownload_RecursivelyDownloadsAllFiles(t *testing.T) {
	h := http.NewServeMux()

	h.HandleFunc("/api/models/org/repo/tree/main", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"type":"directory","path":"subdir"},
			{"type":"file","path":"top.txt","size":3}
		]`))
	})

	h.HandleFunc("/api/models/org/repo/tree/main/subdir", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"type":"file","path":"subdir/model.bin","size":5},
			{"type":"file","path":"subdir/config.json","size":2}
		]`))
	})

	h.HandleFunc("/org/repo/resolve/main/top.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("top"))
	})
	h.HandleFunc("/org/repo/resolve/main/subdir/model.bin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("model"))
	})
	h.HandleFunc("/org/repo/resolve/main/subdir/config.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{}"))
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	provider := newTestHFProvider(srv.URL)
	dest := t.TempDir()

	err := provider.Download(context.Background(), "org/repo", dest, nil, DownloadOptions{})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	modelDir := filepath.Join(dest, "org_repo")
	checks := []struct {
		path string
		want string
	}{
		{path: filepath.Join(modelDir, "top.txt"), want: "top"},
		{path: filepath.Join(modelDir, "subdir", "model.bin"), want: "model"},
		{path: filepath.Join(modelDir, "subdir", "config.json"), want: "{}"},
	}

	for _, c := range checks {
		data, readErr := os.ReadFile(c.path)
		if readErr != nil {
			t.Fatalf("failed to read %s: %v", c.path, readErr)
		}
		if string(data) != c.want {
			t.Fatalf("unexpected content in %s, got %q want %q", c.path, string(data), c.want)
		}
	}

	metaPath := filepath.Join(modelDir, "metadata.json")
	meta, readErr := os.ReadFile(metaPath)
	if readErr != nil {
		t.Fatalf("failed to read metadata file: %v", readErr)
	}
	if !strings.Contains(string(meta), fmt.Sprintf("\"id\": %q", "org/repo")) {
		t.Fatalf("metadata.json does not contain expected model id, got: %s", string(meta))
	}
}
