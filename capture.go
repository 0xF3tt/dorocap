package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type sidecar struct {
	Timestamp string `json:"timestamp"`
	Host      string `json:"host"`
	Type      string `json:"type"`
	Note      string `json:"note,omitempty"`
	Source    string `json:"source,omitempty"`
	SHA256    string `json:"sha256"`
}

var (
	currentGOOS    = runtime.GOOS
	findExecutable = exec.LookPath
	statTool       = os.Stat
	toolVersion    = func(bin string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, bin, "-version").CombinedOutput()
	}
	runScreenshot = func(cmd string, args []string, dest string) error {
		bin, err := trustedBinary(cmd)
		if err != nil {
			return err
		}
		c := &exec.Cmd{Path: bin, Args: append(append([]string{bin}, args...), dest)}
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		return c.Run()
	}
)

func trustedBinary(name string) (string, error) {
	if name == "screencapture" {
		const path = "/usr/sbin/screencapture"
		if _, err := statTool(path); err != nil {
			return "", fmt.Errorf("screencapture not found at %s", path)
		}
		return path, nil
	}
	found, err := findExecutable(name)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(found)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(abs) {
		return "", fmt.Errorf("%s resolved to a non-absolute path", name)
	}
	if name == "import" {
		out, err := toolVersion(abs)
		if err != nil || !bytes.Contains(out, []byte("ImageMagick")) {
			return "", fmt.Errorf("refusing to run %s: expected ImageMagick import", abs)
		}
	}
	return abs, nil
}

func fileSHA256(path string) (string, error) {
	f, err := openIn(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func slugify(s string) string {
	s = slugRe.ReplaceAllString(strings.TrimSpace(s), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "shot"
	}
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func cmdScreenshot(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dorocap ss <type> [note...] | dorocap ss file <src-path> [note...]")
	}
	typ := args[0]
	rest := args[1:]

	root, err := findRoot()
	if err != nil {
		return err
	}

	if typ == "file" {
		if len(rest) < 1 {
			return fmt.Errorf("usage: dorocap ss file <src-path> [note...]")
		}
		return copyFileEvidence(root, rest[0], strings.Join(rest[1:], " "))
	}
	if err := validateCategory("evidence type", typ, false); err != nil {
		return err
	}

	return takeScreenshot(root, typ, strings.Join(rest, " "))
}

func writeSidecar(destPath string, sc sidecar) error {
	b, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(destPath+".json", append(b, '\n'), 0o600)
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func takeScreenshot(root, typ, note string) error {
	cmd, cmdArgs, err := screenshotCommand()
	if err != nil {
		return err
	}

	dir := filepath.Join(root, "evidence", typ)
	if err := mkdirAll(dir, privateDirPerm); err != nil {
		return err
	}

	now := time.Now().UTC()
	ts := now.Format(time.RFC3339Nano)
	suffix, err := uniqueSuffix()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s_%s_%s.png", now.Format("20060102T150405.000000000Z"), slugify(note), suffix)
	dest := filepath.Join(dir, name)

	captureDir, err := os.MkdirTemp("", "dorocap-capture-*")
	if err != nil {
		return err
	}
	tmpPath := filepath.Join(captureDir, "capture.png")
	committed := false
	defer func() {
		_ = os.RemoveAll(captureDir)
		if !committed {
			_ = removeIn(dest)
			_ = removeIn(dest + ".json")
		}
	}()

	if err := runScreenshot(cmd, cmdArgs, tmpPath); err != nil {
		return fmt.Errorf("screenshot capture failed: %w", err)
	}

	if _, err := statIn(tmpPath); err != nil {
		return fmt.Errorf("no screenshot saved (selection cancelled, or screen recording permission denied?)")
	}

	sum, err := fileSHA256(tmpPath)
	if err != nil {
		return err
	}
	if err := copyPathAtomic(tmpPath, dest, 0o600); err != nil {
		return err
	}
	if err := writeSidecar(dest, sidecar{Timestamp: ts, Host: hostname(), Type: typ, Note: note, SHA256: sum}); err != nil {
		return err
	}
	committed = true

	printOK("saved %s", dest)
	return nil
}

func screenshotCommand() (string, []string, error) {
	switch currentGOOS {
	case "darwin":
		return "screencapture", []string{"-i"}, nil
	case "linux":
		opts := []struct {
			bin  string
			args []string
		}{
			{"scrot", []string{"-s"}},
			{"gnome-screenshot", []string{"-a", "-f"}},
			{"import", nil},
		}
		for _, o := range opts {
			if _, err := findExecutable(o.bin); err == nil {
				return o.bin, o.args, nil
			}
		}
		return "", nil, fmt.Errorf("no screenshot tool found (install scrot, gnome-screenshot, or imagemagick)")
	default:
		return "", nil, fmt.Errorf("screenshot capture not supported on %s; use `dorocap ss file <path>` to import one instead", currentGOOS)
	}
}

func copyFileEvidence(root, src, note string) error {
	realSrc, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	info, err := statIn(realSrc)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file: %s", src)
	}

	dir := filepath.Join(root, "evidence", "files")
	if err := mkdirAll(dir, privateDirPerm); err != nil {
		return err
	}

	now := time.Now().UTC()
	ts := now.Format(time.RFC3339Nano)
	suffix, err := uniqueSuffix()
	if err != nil {
		return err
	}
	dest := filepath.Join(dir, fmt.Sprintf("%s_%s_%s", now.Format("20060102T150405.000000000Z"), suffix, filepath.Base(src)))

	committed := false
	defer func() {
		if !committed {
			_ = removeIn(dest)
			_ = removeIn(dest + ".json")
		}
	}()
	if err := copyPathAtomic(realSrc, dest, 0o600); err != nil {
		return err
	}
	sum, err := fileSHA256(dest)
	if err != nil {
		return err
	}
	if err := writeSidecar(dest, sidecar{Timestamp: ts, Host: hostname(), Type: "file", Note: note, Source: filepath.Base(src), SHA256: sum}); err != nil {
		return err
	}
	committed = true

	printOK("saved %s", dest)
	return nil
}
