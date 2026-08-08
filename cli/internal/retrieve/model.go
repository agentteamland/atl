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

// miniLMInt8 is the PREVIOUS embedder model, kept for the record: all-MiniLM-L6-v2 int8 (384-dim,
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

// multilingualInt8 is the ACTIVE embedder model: paraphrase-multilingual-MiniLM-L12-v2
// int8 (384-dim, ~135 MB), replacing the English-only miniLMInt8 above.
//
// The English model had no signal at all on this user's majority language — not
// "weaker", none. Top-1 cosine by band against the live corpus put on-topic
// Turkish (mean 0.140) BELOW off-topic English (0.152): the same question
// measured 0.631 in English against 0.138 in Turkish. The lexical arm is dead
// there too (Turkish coverage 0.00-0.17 against English 0.67-1.00), so both
// halves of the hybrid ranker were failing together on ~87% of prompts.
//
// Chosen by measurement, not by reputation. Four candidates were screened over a
// real-corpus sample, and the quantity that decides it is the GAP between
// on-topic and off-topic — not the absolute score, which is not comparable
// across models:
//
//	model                          sep EN   sep TR   ms/chunk
//	all-MiniLM-L6-v2 (was)          0.264    0.091      535
//	paraphrase-multilingual-L12     0.250    0.274     1457   <- this
//	multilingual-e5-small           0.059    0.029     1160
//	granite-97m (fp32 + CLS)        0.083    0.060      596
//
// e5-small is 20% faster and its absolute scores look far better (on-topic 0.837
// vs L12's 0.382) — and it is rejected, because 0.837 against an off-topic 0.808
// is a 0.029 gap. What a threshold consumes is the gap. granite-97m is 2.4×
// faster and loses by 4.5×; its usable weights are fp32 at 372 MB, and its int8
// export is silently wrong on the pure-Go backend (0.003 separation, no error) —
// which is why a quantized export is only accepted when it preserves separation
// against its fp32 twin.
//
// Same 384 dimensions, so the stored vector format is unchanged — but the
// vectors themselves are not comparable across models, which is what
// indexFormatVersion guards.
var multilingualInt8 = modelSpec{
	dir: "paraphrase-multilingual-MiniLM-L12-v2-int8",
	files: []modelFile{
		{
			url:    "https://huggingface.co/Xenova/paraphrase-multilingual-MiniLM-L12-v2/resolve/main/onnx/model_quantized.onnx",
			name:   "model.onnx",
			size:   118308126,
			sha256: "66fc00f5f29afcaff34092e1bdd20008ca3918265a82fb9695a551e510cc4ebc",
		},
		{
			url:    "https://huggingface.co/Xenova/paraphrase-multilingual-MiniLM-L12-v2/resolve/main/tokenizer.json",
			name:   "tokenizer.json",
			size:   17082913,
			sha256: "b60b6b43406a48bf3638526314f3d232d97058bc93472ff2de930d43686fa441",
		},
		{
			url:    "https://huggingface.co/Xenova/paraphrase-multilingual-MiniLM-L12-v2/resolve/main/config.json",
			name:   "config.json",
			size:   673,
			sha256: "05b570bff786faa5c4604152aa16f19f77ed6dfc31e47dd0f3dd987078693ac7",
		},
	},
}

// activeModel is the one model the retriever uses. Everything reads it rather
// than a spec directly, so a swap is one line here and cannot leave the download
// path and the presence check pointing at different models.
var activeModel = multilingualInt8

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
	return ensureModel(ctx, activeModel)
}

// ModelDirIfPresent returns the default model's local directory and true only if
// every file is already present at its expected size — the cheap check (no hash,
// no network) the retrieval hook uses to decide whether it can run the semantic
// half without triggering a multi-second download on a prompt.
func ModelDirIfPresent() (string, bool) {
	root, err := modelsRoot()
	if err != nil {
		return "", false
	}
	dir := filepath.Join(root, activeModel.dir)
	for _, f := range activeModel.files {
		if !hasFile(filepath.Join(dir, f.name), f.size) {
			return "", false
		}
	}
	return dir, true
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
	// per-process name (not a shared dest+".tmp") is what keeps two processes racing
	// to download the same file — several sessions starting cold at once — from
	// clobbering each other's temp on rename.
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
