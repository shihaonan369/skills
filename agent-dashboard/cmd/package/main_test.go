package main

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestPlatformPackageIncludesBootstrapWithoutDeveloperRuntime(t *testing.T) {
	t.Chdir("../..")
	binary := filepath.Join(t.TempDir(), "synthetic-binary")
	if err := os.WriteFile(binary, []byte("not executed"), 0700); err != nil {
		t.Fatal(err)
	}
	files, err := packageFiles(binary)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"SKILL.md", "README.md", "bootstrap.sh", "bootstrap.ps1", "bootstrap.sha256", "docs/REX.md", "docs/OBSERVATIONS.md", "docs/DYNAMIC.md", "examples/rex/overview.layout.json", "examples/rex/records.template.layout.json", "LICENSES.txt", binary} {
		if !slices.Contains(files, required) {
			t.Fatalf("missing bootstrap package file %s", required)
		}
	}
	for _, path := range files {
		if strings.HasSuffix(path, ".py") || strings.HasPrefix(path, "tests/") || strings.HasPrefix(path, "legacy/") {
			t.Fatalf("developer runtime or legacy file shipped: %s", path)
		}
	}
	archivePath := filepath.Join(t.TempDir(), "skill.zip")
	if err := archive(archivePath, files); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	for _, name := range []string{"bootstrap.sh", "bootstrap.ps1", "bootstrap.sha256", ".prebundled"} {
		f, err := z.Open("agent-dashboard/" + name)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

func TestBundledBootstrapPinsCoverPublishedPlatforms(t *testing.T) {
	raw, err := os.ReadFile("../../bootstrap.sha256")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^[0-9a-f]{64}  agent-dashboard-0\.2\.0-((?:darwin|linux|windows)-(?:arm64|amd64))\.zip$`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	seen := map[string]bool{}
	for _, match := range matches {
		if seen[match[1]] {
			t.Fatal("duplicate platform pin", match[1])
		}
		seen[match[1]] = true
	}
	if len(matches) != 6 || len(strings.Fields(string(raw))) != 12 {
		t.Fatal("expected exactly six pinned release archives")
	}
}

func TestReleaseManifestUsesDownloadFilenames(t *testing.T) {
	dir := t.TempDir()
	var files []string
	var want strings.Builder
	for _, name := range []string{"agent-dashboard-0.2.0-darwin-arm64.zip", "agent-dashboard-0.2.0-windows-amd64.zip"} {
		content := []byte("synthetic archive: " + name)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0600); err != nil {
			t.Fatal(err)
		}
		files = append(files, path)
		fmt.Fprintf(&want, "%x  %s\n", sha256.Sum256(content), name)
	}
	got, err := releaseManifest(files)
	if err != nil || string(got) != want.String() {
		t.Fatalf("release manifest must reference downloadable filenames, not local bin/target paths: %s, %v", got, err)
	}
	if _, err := releaseManifest([]string{filepath.Join(dir, "missing.zip")}); err == nil {
		t.Fatal("missing archive accepted")
	}
}
