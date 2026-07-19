package retrieve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/scope"
)

// modelFile is one downloadable model file: a source URL, the local filename
// hugot expects in the model directory, the exact byte size, and the sha256 the
// bytes must match. The download is fail-closed — a file is never used
// unverified; the size is the cheap presence check on the hot path (a full
// re-hash on every call would blow the per-prompt latency budget).
type modelFile struct {
	url    string
	name   string
	size   int64
	sha256 string
}

// modelSpec pins one embedding model as a set of files under a directory name.
type modelSpec struct {
	dir   string // directory name under ~/.atl/models/
	files []modelFile
}

// miniLMInt8 is the default embedder model: all-MiniLM-L6-v2 int8 (384-dim,
// ~22 MB) — the cold-start and concurrency winner from the #140 spike (~74-90 ms
// cold on an M2 Max, 2x headroom under the 200 ms budget; int8 is lossless for
// ranking). Sourced from Xenova/all-MiniLM-L6-v2 (its quantized ONNX export) and
// sha256-pinned to the exact files the spike validated.
var miniLMInt8 = modelSpec{
	dir: "all-MiniLM-L6-v2-int8",
	files: []modelFile{
		{
			url:    "https://huggingface.co/Xenova/all-MiniLM-L6-v2/resolve/main/onnx/model_quantized.onnx",
			name:   "model.onnx",
			size:   22972370,
			sha256: "afdb6f1a0e45b715d0bb9b11772f032c399babd23bfc31fed1c170afc848bdb1",
		},
		{
			url:    "https://huggingface.co/Xenova/all-MiniLM-L6-v2/resolve/main/tokenizer.json",
			name:   "tokenizer.json",
			size:   711661,
			sha256: "da0e79933b9ed51798a3ae27893d3c5fa4a201126cef75586296df9b4d2c62a0",
		},
		{
			url:    "https://huggingface.co/Xenova/all-MiniLM-L6-v2/resolve/main/config.json",
			name:   "config.json",
			size:   650,
			sha256: "7135149f7cffa1a573466c6e4d8423ed73b62fd2332c575bf738a0d033f70df7",
		},
	},
}

// modelsRoot is ~/.atl/models — the global cache for downloaded embedder models
// (shared across projects, never committed, and untouched by gc, which only
// scans the .claude asset dirs).
func modelsRoot() (string, error) {
	layer, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(layer, "models"), nil
}

// EnsureModel returns the local directory of the default embedding model,
// downloading and sha256-verifying its files on first use. Idempotent: when
// every file is already present and verifies, it makes no network call.
func EnsureModel(ctx context.Context) (string, error) {
	return ensureModel(ctx, miniLMInt8)
}

func ensureModel(ctx context.Context, spec modelSpec) (string, error) {
	root, err := modelsRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, spec.dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, f := range spec.files {
		dest := filepath.Join(dir, f.name)
		if hasFile(dest, f.size) {
			continue // present at the expected size — the hot path: no hash, no network
		}
		if err := downloadVerified(ctx, f, dest); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// hasFile reports whether path exists as a regular file of exactly size bytes.
// This is the cheap presence check on the per-prompt hot path — a full re-hash of
// a ~22 MB model on every call would blow the latency budget. Integrity is
// enforced by sha256 on the download path (downloadVerified); the size check here
// catches the common truncation case and triggers a re-download. A same-size but
// corrupted file would fail at model load, where retrieval fails open.
func hasFile(path string, size int64) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular() && fi.Size() == size
}

// verifyFile returns nil iff the file at path exists and its sha256 equals want.
// Used by tests to fully confirm the pinned model before asserting on it.
func verifyFile(path, want string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if got := sha256Hex(b); got != want {
		return fmt.Errorf("checksum mismatch for %s (got %s, want %s)", path, got, want)
	}
	return nil
}

// modelHTTPClient has no timeout of its own — the caller bounds each download
// with a context deadline (the models are tens of MB).
var modelHTTPClient = &http.Client{}

// downloadVerified fetches f.url, verifies its sha256, and atomically writes it
// to dest. It never writes an unverified file.
func downloadVerified(ctx context.Context, f modelFile, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return err
	}
	resp, err := modelHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", f.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: %s", f.name, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", f.name, err)
	}
	if got := sha256Hex(b); got != f.sha256 {
		return fmt.Errorf("checksum mismatch for %s (got %s, want %s)", f.name, got, f.sha256)
	}
	// Write to a per-process temp file in the same dir, then atomically rename. A
	// per-process name (not a shared dest+".tmp") is what keeps two workers racing
	// to download the same file — the N-way `atl work dispatch --cap N` cold-start
	// case — from clobbering each other's temp on rename.
	tmp, err := os.CreateTemp(filepath.Dir(dest), f.name+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dest)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
