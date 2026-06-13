// Package source fetches a team's files from its GitHub repo as a tarball and
// extracts the requested subpath — the deterministic fetch behind `atl install`.
//
// No git binary is required (decision doc item 9, script-only distribution): a
// single HTTPS GET of the ref tarball, gunzip + untar in-process. A standalone
// team uses subpath ""; a first-party team in the atl monorepo uses its
// teams/<name> subpath. Extraction strips the archive's top-level
// "<repo>-<ref>/" directory and is hardened against path traversal.
package source

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// httpClient is used for tarball fetches; a timeout keeps install from hanging.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// TarballURL is the GitHub archive URL for a repo at a ref (tag, branch, or SHA).
func TarballURL(repo, ref string) string {
	return fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", repo, ref)
}

// Fetch downloads repo@ref, extracts subpath into a fresh temp directory, and
// returns that directory. The caller owns it and should remove it when done.
// subpath "" means the repo root.
func Fetch(repo, subpath, ref string) (dir string, err error) {
	url := TarballURL(repo, ref)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	dest, err := os.MkdirTemp("", "atl-src-*")
	if err != nil {
		return "", err
	}
	if err := Extract(resp.Body, subpath, dest); err != nil {
		os.RemoveAll(dest)
		return "", err
	}
	return dest, nil
}

// Extract reads a gzipped tar from r, strips the archive's single top-level
// directory, keeps only entries under subpath, and writes them under dest
// (so subpath becomes dest's root). Hardened against path traversal.
func Extract(r io.Reader, subpath, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gunzip: %w", err)
	}
	defer gz.Close()

	subpath = path.Clean("/" + strings.Trim(subpath, "/"))[1:] // normalized, no leading slash
	tr := tar.NewReader(gz)
	wrote := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("untar: %w", err)
		}

		rel := stripTop(hdr.Name)
		if rel == "" {
			continue // the top-level dir itself
		}
		if subpath != "" {
			if rel != subpath && !strings.HasPrefix(rel, subpath+"/") {
				continue
			}
			rel = strings.TrimPrefix(strings.TrimPrefix(rel, subpath), "/")
			if rel == "" {
				continue
			}
		}

		target, err := safeJoin(dest, rel)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
			wrote++
		}
	}
	if wrote == 0 {
		return fmt.Errorf("no files found under subpath %q", subpath)
	}
	return nil
}

// stripTop removes the archive's leading "<repo>-<ref>/" path segment.
func stripTop(name string) string {
	name = strings.TrimPrefix(name, "./")
	i := strings.IndexByte(name, '/')
	if i < 0 {
		return ""
	}
	return name[i+1:]
}

// safeJoin joins rel onto dest, refusing any path that escapes dest.
func safeJoin(dest, rel string) (string, error) {
	target := filepath.Join(dest, rel)
	if target != dest && !strings.HasPrefix(target, dest+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe path in archive: %q", rel)
	}
	return target, nil
}
