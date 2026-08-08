package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFileEvidenceUniqueAndVerifiable(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(t.TempDir(), "proof.txt")
	if err := os.WriteFile(src, []byte("proof"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileEvidence(root, src, "first"); err != nil {
		t.Fatal(err)
	}
	if err := copyFileEvidence(root, src, "second"); err != nil {
		t.Fatal(err)
	}
	evidence, issues := inspectEvidence(root)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	if len(evidence) != 2 || evidence[0].Path == evidence[1].Path {
		t.Fatalf("expected two unique files: %+v", evidence)
	}
	for _, item := range evidence {
		if item.Sidecar.Source != "proof.txt" {
			t.Fatalf("source leaked or changed: %q", item.Sidecar.Source)
		}
	}
}

func TestInspectEvidenceDetectsTamperMissingAndOrphan(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "evidence", "files")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	tampered := filepath.Join(dir, "tampered.txt")
	if err := os.WriteFile(tampered, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeSidecar(tampered, sidecar{Timestamp: "now", Host: "host", Type: "file", SHA256: strings.Repeat("0", 64)}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "missing.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	orphan, _ := json.Marshal(sidecar{Timestamp: "now", Host: "host", Type: "file", SHA256: strings.Repeat("a", 64)})
	if err := os.WriteFile(filepath.Join(dir, "gone.txt.json"), orphan, 0o600); err != nil {
		t.Fatal(err)
	}
	_, issues := inspectEvidence(root)
	joined := strings.Join(issues, "\n")
	for _, want := range []string{"SHA-256 mismatch", "missing or unreadable sidecar", "orphaned sidecar"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in issues: %v", want, issues)
		}
	}
}

func TestTakeScreenshotUsesAtomicTemporaryTarget(t *testing.T) {
	root := t.TempDir()
	oldOS, oldFind, oldRun := currentGOOS, findExecutable, runScreenshot
	t.Cleanup(func() { currentGOOS, findExecutable, runScreenshot = oldOS, oldFind, oldRun })
	currentGOOS = "darwin"
	runScreenshot = func(_ string, _ []string, dest string) error {
		if filepath.Base(dest) != "capture.png" || !strings.Contains(filepath.Base(filepath.Dir(dest)), "dorocap-capture-") {
			t.Fatalf("runner did not receive temporary path: %s", dest)
		}
		return os.WriteFile(dest, []byte("png"), 0o600)
	}
	if err := takeScreenshot(root, "recon", "login"); err != nil {
		t.Fatal(err)
	}
	evidence, issues := inspectEvidence(root)
	if len(issues) != 0 || len(evidence) != 1 {
		t.Fatalf("evidence=%v issues=%v", evidence, issues)
	}
}

func TestScreenshotCommandSelection(t *testing.T) {
	oldOS, oldFind := currentGOOS, findExecutable
	t.Cleanup(func() { currentGOOS, findExecutable = oldOS, oldFind })
	currentGOOS = "linux"
	findExecutable = func(name string) (string, error) {
		if name == "gnome-screenshot" {
			return "/bin/gnome-screenshot", nil
		}
		return "", errors.New("missing")
	}
	cmd, args, err := screenshotCommand()
	if err != nil || cmd != "gnome-screenshot" || len(args) != 2 {
		t.Fatalf("cmd=%q args=%v err=%v", cmd, args, err)
	}
	currentGOOS = "windows"
	if _, _, err := screenshotCommand(); err == nil {
		t.Fatal("expected unsupported platform error")
	}
}
