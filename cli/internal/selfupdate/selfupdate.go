// Package selfupdate updates the running atl binary to the latest stable
// release. It is the shared machinery behind the manual `atl upgrade` command
// and the automatic session-start auto-apply: resolve the latest stable release,
// compare it to the running build (only-upgrade, never downgrade), download the
// matching release asset, verify its sha256 against the published checksums, and
// atomically replace the running executable.
//
// Safety rails, all mandatory:
//   - the ATL_NO_SELF_UPDATE env var disables it entirely (the only opt-out);
//   - an un-stamped "dev" build is never replaced (Check returns Upgrade=false);
//   - the downloaded asset is sha256-verified before it touches the install dir;
//   - Windows can't overwrite a running .exe, so Apply returns ErrWindowsManual.
package selfupdate

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/agentteamland/atl/cli/internal/semver"
)

const (
	// EnvDisable turns self-update off entirely (the only opt-out — the feature is
	// otherwise mandatory, per the v2 "automation is mandatory" stance).
	EnvDisable = "ATL_NO_SELF_UPDATE"

	// devVersion is buildinfo.Version for an un-stamped `go build` — never updated.
	devVersion = "dev"

	apiLatest    = "https://api.github.com/repos/agentteamland/atl/releases/latest"
	downloadBase = "https://github.com/agentteamland/atl/releases/download"
	binaryName   = "atl"
)

// ErrWindowsManual is returned by Apply on Windows: a running .exe cannot be
// overwritten in place, so the user must rerun the install script instead.
var ErrWindowsManual = errors.New("windows binaries can't self-replace in place; rerun the install script to upgrade")

// httpClient has no timeout of its own — callers bound each call with a context
// deadline (a short one for the check, a longer one for the download).
var httpClient = &http.Client{}

// Status is the result of a version Check.
type Status struct {
	Current string // the running build's version (buildinfo.Version), e.g. "2.3.1" or "dev"
	Latest  string // the resolved latest stable tag, e.g. "v2.3.2" (empty when skipped)
	Upgrade bool   // true iff Latest is strictly newer than Current
	Reason  string // when Upgrade is false: why (disabled / dev build / already up to date)
}

// Check resolves the latest stable release and compares it to current. It makes
// at most one network call. Disabled/dev cases are not errors — they return
// Upgrade=false with a Reason and no network call.
func Check(ctx context.Context, current string) (Status, error) {
	st := Status{Current: current}
	if os.Getenv(EnvDisable) != "" {
		st.Reason = "disabled via " + EnvDisable
		return st, nil
	}
	if current == devVersion || current == "" {
		st.Reason = "dev build — self-update skipped"
		return st, nil
	}
	tag, err := latestStableTag(ctx)
	if err != nil {
		return st, err
	}
	st.Latest = tag
	// semver.Less tolerates the v-prefix asymmetry ("2.3.1" vs "v2.3.1") and treats
	// the current build as an upgrade target only when it is strictly older.
	st.Upgrade = semver.Less(current, tag)
	if !st.Upgrade {
		st.Reason = "already up to date"
	}
	return st, nil
}

// Apply downloads the release asset for tag matching this OS/arch, verifies its
// sha256 against the published checksums, and atomically replaces the running
// binary. Unix only — on Windows it returns ErrWindowsManual without touching disk.
func Apply(ctx context.Context, tag string) error {
	if runtime.GOOS == "windows" {
		return ErrWindowsManual
	}
	version := strings.TrimPrefix(tag, "v")
	asset := assetName(version, runtime.GOOS, runtime.GOARCH)

	archive, err := download(ctx, assetURL(tag, asset))
	if err != nil {
		return fmt.Errorf("downloading %s: %w", asset, err)
	}
	checksums, err := download(ctx, checksumsURL(tag, version))
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	if err := verifyChecksum(asset, archive, checksums); err != nil {
		return err
	}
	bin, err := extractBinary(archive, binaryName)
	if err != nil {
		return err
	}
	return replaceExecutable(bin)
}

// latestStableTag returns the tag_name of the latest stable release. GitHub's
// /releases/latest excludes prereleases, which is exactly the stable-only channel.
func latestStableTag(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiLatest, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release check: %s", resp.Status)
	}
	var out struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.TagName == "" {
		return "", errors.New("release check: no tag_name in latest release")
	}
	return out.TagName, nil
}

// assetName mirrors the goreleaser archive name_template:
// atl_<version>_<os>_<arch>.<ext> with version v-stripped and windows → zip.
func assetName(version, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s.%s", binaryName, version, goos, goarch, ext)
}

func assetURL(tag, asset string) string {
	return downloadBase + "/" + tag + "/" + asset
}

func checksumsURL(tag, version string) string {
	return downloadBase + "/" + tag + "/" + fmt.Sprintf("%s_%s_checksums.txt", binaryName, version)
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum confirms the asset's sha256 matches its line in a goreleaser
// checksums.txt ("<hex>  <filename>" per line). A missing or mismatched entry is
// an error — the binary is never swapped on an unverified download.
func verifyChecksum(asset string, archive, checksums []byte) error {
	sum := sha256.Sum256(archive)
	want := hex.EncodeToString(sum[:])
	sc := bufio.NewScanner(bytes.NewReader(checksums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		if fields[1] == asset {
			if !strings.EqualFold(fields[0], want) {
				return fmt.Errorf("checksum mismatch for %s (got %s, want %s)", asset, want, fields[0])
			}
			return nil
		}
	}
	return fmt.Errorf("no checksum entry for %s", asset)
}

// extractBinary pulls the named regular file out of a .tar.gz archive.
func extractBinary(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("untar: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", name)
}

// replaceExecutable atomically swaps the running binary for newBin. It resolves
// the real on-disk path (through symlinks), writes newBin to a temp file in the
// SAME directory (so the rename is atomic on one filesystem), makes it
// executable, then renames it over the current binary. Overwriting a running
// binary is permitted on Unix; the current process keeps executing the old inode.
func replaceExecutable(newBin []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating the running binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)

	tmp, err := os.CreateTemp(dir, ".atl-upgrade-*")
	if err != nil {
		return fmt.Errorf("install dir %s is not writable (rerun the install script, or fix permissions): %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename below succeeds

	if _, err := tmp.Write(newBin); err != nil {
		tmp.Close()
		return fmt.Errorf("writing the new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing the new binary: %w", err)
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return fmt.Errorf("setting executable bit: %w", err)
	}
	if err := os.Rename(tmpName, exe); err != nil {
		return fmt.Errorf("replacing %s: %w", exe, err)
	}
	return nil
}
