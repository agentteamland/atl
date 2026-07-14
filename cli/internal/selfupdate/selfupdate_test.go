package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestAssetNameAndURLs(t *testing.T) {
	if got := assetName("2.3.1", "darwin", "arm64"); got != "atl_2.3.1_darwin_arm64.tar.gz" {
		t.Errorf("assetName darwin = %q", got)
	}
	if got := assetName("2.3.1", "linux", "amd64"); got != "atl_2.3.1_linux_amd64.tar.gz" {
		t.Errorf("assetName linux = %q", got)
	}
	if got := assetName("2.3.1", "windows", "amd64"); got != "atl_2.3.1_windows_amd64.zip" {
		t.Errorf("assetName windows = %q", got)
	}
	if got := assetURL("v2.3.1", "atl_2.3.1_darwin_arm64.tar.gz"); got != "https://github.com/agentteamland/atl/releases/download/v2.3.1/atl_2.3.1_darwin_arm64.tar.gz" {
		t.Errorf("assetURL = %q", got)
	}
	if got := checksumsURL("v2.3.1", "2.3.1"); got != "https://github.com/agentteamland/atl/releases/download/v2.3.1/atl_2.3.1_checksums.txt" {
		t.Errorf("checksumsURL = %q", got)
	}
}

func TestVerifyChecksum(t *testing.T) {
	archive := []byte("the release tarball bytes")
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])
	asset := "atl_2.3.1_darwin_arm64.tar.gz"
	// goreleaser checksums.txt: "<hex>  <filename>", one per line, multiple assets.
	checksums := []byte(
		"0000000000000000000000000000000000000000000000000000000000000000  atl_2.3.1_linux_amd64.tar.gz\n" +
			hexSum + "  " + asset + "\n",
	)

	if err := verifyChecksum(asset, archive, checksums); err != nil {
		t.Fatalf("valid checksum rejected: %v", err)
	}
	if err := verifyChecksum(asset, []byte("tampered"), checksums); err == nil {
		t.Error("tampered archive passed verification")
	}
	if err := verifyChecksum("atl_2.3.1_windows_amd64.zip", archive, checksums); err == nil {
		t.Error("missing checksum entry passed verification")
	}
}

func TestExtractBinary(t *testing.T) {
	want := []byte("\x7fELF...the atl binary...")
	archive := makeTarGz(t, map[string][]byte{
		"README.md": []byte("readme"),
		"atl":       want,
		"LICENSE":   []byte("license"),
	})

	got, err := extractBinary(archive, "atl")
	if err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted binary = %q, want %q", got, want)
	}

	if _, err := extractBinary(archive, "nope"); err == nil {
		t.Error("extractBinary found a nonexistent file")
	}
}

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o755,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
