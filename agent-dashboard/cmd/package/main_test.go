package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseManifestUsesDownloadFilenames(t *testing.T) {
	dir := t.TempDir()
	var files []string
	var want strings.Builder
	for _, name := range []string{"agent-dashboard-0.1.0-darwin-arm64.zip", "agent-dashboard-0.1.0-windows-amd64.zip"} {
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
