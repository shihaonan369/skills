// Build-time only: run from the skill root with `go run ./cmd/package`.
// Produces unsigned binaries and one self-contained skill zip per target.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if e := build(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func build() error {
	if _, e := os.Stat("SKILL.md"); e != nil {
		return fmt.Errorf("run from agent-dashboard/: %w", e)
	}
	targets := []string{"darwin/arm64", "darwin/amd64", "linux/arm64", "linux/amd64", "windows/arm64", "windows/amd64"}
	goexe := filepath.Join(os.Getenv("GOROOT"), "bin", "go")
	if _, e := os.Stat(goexe); e != nil {
		var e error
		goexe, e = exec.LookPath("go")
		if e != nil {
			return e
		}
	}
	if e := notices(goexe); e != nil {
		return e
	}
	if e := os.MkdirAll("target", 0755); e != nil {
		return e
	}
	var sums strings.Builder
	for _, target := range targets {
		parts := strings.Split(target, "/")
		name := strings.Join(parts, "-")
		binary := "agent-dashboard"
		if parts[0] == "windows" {
			binary += ".exe"
		}
		path := filepath.Join("bin", name, binary)
		if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
			return e
		}
		cmd := exec.Command(goexe, "build", "-trimpath", "-ldflags=-s -w -buildid=", "-o", path, "./cmd/agent-dashboard")
		cmd.Env = append(os.Environ(), "GOOS="+parts[0], "GOARCH="+parts[1], "CGO_ENABLED=0")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e := cmd.Run(); e != nil {
			return e
		}
		info, e := os.Stat(path)
		if e != nil {
			return e
		}
		fmt.Printf("%s · %d bytes\n", name, info.Size())
		zipPath := filepath.Join("target", "agent-dashboard-0.1.0-"+name+".zip")
		files := []string{"SKILL.md", "README.md", "docs/ARCHITECTURE.md", "docs/VALIDATION.md", "docs/PANELS.md", "docs/DAG.md", path, "LICENSES.txt"}
		if e := filepath.WalkDir("examples", func(p string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && (filepath.Ext(p) == ".json" || filepath.Ext(p) == ".md") {
				files = append(files, p)
			}
			return nil
		}); e != nil {
			return e
		}
		if e := archive(zipPath, files); e != nil {
			return e
		}
		for _, p := range []string{path, zipPath} {
			raw, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(raw), filepath.ToSlash(p))
		}
	}
	return os.WriteFile("target/SHA256SUMS", []byte(sums.String()), 0644)
}
func archive(path string, files []string) (err error) {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer func() {
		ce := f.Close()
		if err == nil {
			err = ce
		}
	}()
	z := zip.NewWriter(f)
	for _, p := range files {
		in, e := os.Open(p)
		if e != nil {
			return e
		}
		info, e := in.Stat()
		if e != nil {
			in.Close()
			return e
		}
		header, e := zip.FileInfoHeader(info)
		if e != nil {
			in.Close()
			return e
		}
		header.Name = "agent-dashboard/" + filepath.ToSlash(p)
		header.Method = zip.Deflate
		if strings.HasPrefix(p, "bin"+string(filepath.Separator)) {
			header.SetMode(0755)
		}
		w, e := z.CreateHeader(header)
		if e == nil {
			_, e = io.Copy(w, in)
		}
		in.Close()
		if e != nil {
			return e
		}
	}
	return z.Close()
}

func notices(goexe string) error {
	cmd := exec.Command(goexe, "list", "-m", "-json", "all")
	raw, e := cmd.Output()
	if e != nil {
		return e
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var out strings.Builder
	out.WriteString("Third-party dependency notices for local builds. No license is assigned here to the user's own repository code.\n\n")
	for {
		var m struct {
			Path, Version, Dir string
			Main               bool
		}
		if e := dec.Decode(&m); e == io.EOF {
			break
		} else if e != nil {
			return e
		}
		if m.Main {
			continue
		}
		if m.Dir == "" {
			return fmt.Errorf("module %s not cached; run go mod download all before packaging", m.Path)
		}
		fmt.Fprintf(&out, "\n===== %s %s =====\n", m.Path, m.Version)
		found := false
		entries, e := os.ReadDir(m.Dir)
		if e != nil {
			return e
		}
		for _, entry := range entries {
			name := entry.Name()
			upper := strings.ToUpper(name)
			if entry.IsDir() || !(strings.HasPrefix(upper, "LICENSE") || strings.HasPrefix(upper, "LICENCE") || strings.HasPrefix(upper, "COPYING") || strings.HasPrefix(upper, "NOTICE")) {
				continue
			}
			raw, e := os.ReadFile(filepath.Join(m.Dir, name))
			if e == nil {
				found = true
				out.Write(raw)
				out.WriteString("\n")
			}
		}
		if !found {
			raw, e := os.ReadFile(filepath.Join(m.Dir, "README.md"))
			if e == nil && strings.Contains(strings.ToLower(string(raw)), "license") {
				out.WriteString("Upstream ships its license declaration in README.md (reproduced verbatim):\n")
				out.Write(raw)
				found = true
			}
		}
		if !found {
			return fmt.Errorf("no dependency license declaration found for %s", m.Path)
		}
	}
	cmd = exec.Command(goexe, "env", "GOROOT")
	raw, e = cmd.Output()
	if e != nil {
		return e
	}
	raw, e = os.ReadFile(filepath.Join(strings.TrimSpace(string(raw)), "LICENSE"))
	if e != nil {
		return e
	}
	out.WriteString("\n===== Go runtime/toolchain =====\n")
	out.Write(raw)
	return os.WriteFile("LICENSES.txt", []byte(out.String()), 0644)
}
