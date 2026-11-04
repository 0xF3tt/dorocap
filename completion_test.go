package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionScriptsCoverCommandsAndDynamicLinking(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish", "powershell"} {
		script, err := completionScript(shell)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"finding", "completion candidates", "findings", "evidence"} {
			if !strings.Contains(script, want) {
				t.Errorf("%s completion missing %q", shell, want)
			}
		}
	}
	if _, err := completionScript("nushell"); err == nil {
		t.Fatal("expected unsupported shell error")
	}
}

func TestLinkCompletionOffersOnlyUnlinkedVerifiedEvidence(t *testing.T) {
	root := scaffoldTestEngagement(t)
	dir := filepath.Join(root, "findings")
	if err := findingAdd(dir, []string{"TLS", "issue", "--severity", "med"}); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "proof.txt")
	if err := os.WriteFile(src, []byte("proof"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileEvidence(root, src, "proof"); err != nil {
		t.Fatal(err)
	}

	ids, err := findingCompletionCandidates(root)
	if err != nil || len(ids) != 1 || ids[0] != "0001" {
		t.Fatalf("finding candidates=%v err=%v", ids, err)
	}
	paths, err := evidenceCompletionCandidates(root, "0001")
	if err != nil || len(paths) != 1 {
		t.Fatalf("evidence candidates=%v err=%v", paths, err)
	}
	if err := findingLink(dir, []string{"0001", paths[0]}); err != nil {
		t.Fatal(err)
	}
	paths, err = evidenceCompletionCandidates(root, "0001")
	if err != nil || len(paths) != 0 {
		t.Fatalf("linked evidence was still offered: %v err=%v", paths, err)
	}
}
