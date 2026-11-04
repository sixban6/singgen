package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// checkTimeout bounds how long `sing-box check` may run
const checkTimeout = 30 * time.Second

// outputWriter writes the generated config atomically and, when enabled and
// possible, validates it with `sing-box check` before replacing the target.
type outputWriter struct {
	logger *slog.Logger
	check  bool
	// lookPath is overridable for tests; defaults to exec.LookPath
	lookPath func(string) (string, error)
}

// write stores data at outFile ("-" for stdout). The data is first written to a
// temporary file in the target directory; the target is only replaced (atomic
// rename) after the optional check succeeds, so a bad config never overwrites
// a working one.
func (w outputWriter) write(outFile string, data []byte) error {
	dir := os.TempDir()
	if outFile != "-" {
		dir = filepath.Dir(outFile)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory failed: %w", err)
		}
	}

	tmp, err := os.CreateTemp(dir, ".singgen-*.json")
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file failed: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file failed: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file failed: %w", err)
	}

	if err := w.validate(tmpPath, data); err != nil {
		return err
	}

	if outFile == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("chmod temp file failed: %w", err)
	}
	if err := os.Rename(tmpPath, outFile); err != nil {
		return fmt.Errorf("replace output file failed: %w", err)
	}
	return nil
}

// validate runs `sing-box check` on path when enabled and applicable.
func (w outputWriter) validate(path string, data []byte) error {
	if !w.check {
		return nil
	}
	if !looksLikeJSON(data) {
		w.logger.Info("Skipping sing-box check: only JSON output can be checked")
		return nil
	}
	if linuxOnly(data) && runtime.GOOS != "linux" {
		w.logger.Warn("Skipping sing-box check: config uses linux-only options (default_mark/tproxy) that cannot be checked on " + runtime.GOOS)
		return nil
	}

	lookPath := w.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	singbox, err := lookPath("sing-box")
	if err != nil {
		w.logger.Info("Skipping sing-box check: sing-box not found in PATH")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, singbox, "check", "-c", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("generated config failed `sing-box check` (output file left unchanged; use -check=false to skip): %v\n%s",
			err, strings.TrimSpace(string(output)))
	}
	w.logger.Info("sing-box check passed", "sing_box", singbox)
	return nil
}

func looksLikeJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// linuxOnly reports whether the config uses options sing-box only accepts on linux
func linuxOnly(data []byte) bool {
	var cfg struct {
		Route    map[string]any   `json:"route"`
		Inbounds []map[string]any `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	if _, ok := cfg.Route["default_mark"]; ok {
		return true
	}
	for _, in := range cfg.Inbounds {
		if in["type"] == "tproxy" || in["type"] == "redirect" {
			return true
		}
	}
	return false
}
