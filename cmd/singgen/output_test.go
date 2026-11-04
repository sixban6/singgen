package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const validJSON = `{"outbounds":[{"type":"direct","tag":"d"}]}`

// fakeSingBox creates an executable script that mimics `sing-box check`
func fakeSingBox(t *testing.T, exitCode int, message string) func(string) (string, error) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake not supported on windows")
	}
	path := filepath.Join(t.TempDir(), "sing-box")
	script := "#!/bin/sh\necho '" + message + "'\nexit " + string(rune('0'+exitCode)) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return func(string) (string, error) { return path, nil }
}

func missingSingBox(string) (string, error) { return "", os.ErrNotExist }

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestWriteReplacesFileWhenCheckPasses(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "config.json")
	os.WriteFile(out, []byte("old"), 0o644)

	w := outputWriter{logger: quietLogger(), check: true, lookPath: fakeSingBox(t, 0, "ok")}
	if err := w.write(out, []byte(validJSON)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if got, _ := os.ReadFile(out); string(got) != validJSON {
		t.Errorf("output not replaced: %s", got)
	}
	if st, _ := os.Stat(out); st.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", st.Mode().Perm())
	}
	if names := listDir(t, dir); len(names) != 1 {
		t.Errorf("temp files left behind: %v", names)
	}
}

func TestWriteKeepsOldFileWhenCheckFails(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "config.json")
	os.WriteFile(out, []byte("working-config"), 0o644)

	w := outputWriter{logger: quietLogger(), check: true, lookPath: fakeSingBox(t, 1, "FATAL unknown field")}
	err := w.write(out, []byte(validJSON))
	if err == nil || !strings.Contains(err.Error(), "FATAL unknown field") || !strings.Contains(err.Error(), "-check=false") {
		t.Fatalf("expected check failure with sing-box output and hint, got %v", err)
	}
	if got, _ := os.ReadFile(out); string(got) != "working-config" {
		t.Errorf("existing config was overwritten: %s", got)
	}
	if names := listDir(t, dir); len(names) != 1 {
		t.Errorf("temp files left behind: %v", names)
	}
}

func TestWriteSkipsCheckWhenDisabledOrUnavailable(t *testing.T) {
	failing := fakeSingBox(t, 1, "should not run")
	cases := map[string]outputWriter{
		"disabled":          {logger: quietLogger(), check: false, lookPath: failing},
		"sing-box missing":  {logger: quietLogger(), check: true, lookPath: missingSingBox},
		"yaml output":       {logger: quietLogger(), check: true, lookPath: failing},
		"new file, no prev": {logger: quietLogger(), check: true, lookPath: missingSingBox},
	}
	for name, w := range cases {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "sub", "config.json")
			data := []byte(validJSON)
			if name == "yaml output" {
				data = []byte("outbounds: []\n")
			}
			if err := w.write(out, data); err != nil {
				t.Fatalf("write failed: %v", err)
			}
			if got, _ := os.ReadFile(out); string(got) != string(data) {
				t.Errorf("unexpected content: %s", got)
			}
		})
	}
}

func TestLinuxOnlyConfigCheckSkippedOffLinux(t *testing.T) {
	linuxCfg := []byte(`{"route":{"default_mark":1},"inbounds":[{"type":"tproxy"}]}`)
	if !linuxOnly(linuxCfg) || linuxOnly([]byte(validJSON)) {
		t.Fatal("linuxOnly detection wrong")
	}

	out := filepath.Join(t.TempDir(), "config.json")
	w := outputWriter{logger: quietLogger(), check: true, lookPath: fakeSingBox(t, 1, "default_mark is only supported on linux")}
	err := w.write(out, linuxCfg)
	if runtime.GOOS == "linux" {
		if err == nil {
			t.Error("on linux the check must run (and the fake fails)")
		}
	} else if err != nil {
		t.Errorf("linux-only config must not be checked on %s: %v", runtime.GOOS, err)
	}
}

func TestWriteStdoutCheckFailureWritesNothing(t *testing.T) {
	r, wPipe, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = wPipe
	defer func() { os.Stdout = orig }()

	w := outputWriter{logger: quietLogger(), check: true, lookPath: fakeSingBox(t, 1, "bad")}
	err := w.write("-", []byte(validJSON))
	wPipe.Close()
	written, _ := io.ReadAll(r)

	if err == nil {
		t.Fatal("expected check failure")
	}
	if len(written) != 0 {
		t.Errorf("nothing should be written to stdout on failure, got %q", written)
	}
}
