package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFindingAddInteractive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "findings")
	oldInput, oldOutput := findingInput, findingOutput
	t.Cleanup(func() { findingInput, findingOutput = oldInput, oldOutput })
	findingInput = strings.NewReader("SQL injection in login\nhigh\n\nexample.com/login\n8.8\nCWE-89\nA03:2021 Injection\nUnsanitized input reaches a SQL query.\nAccount data may be exposed.\nDatabase queries can be modified.\nSubmit a quote in the username field.\nUse parameterized queries.\nhttps://owasp.org/\n")
	var output bytes.Buffer
	findingOutput = &output

	if err := findingAdd(dir, []string{"--interactive"}); err != nil {
		t.Fatal(err)
	}
	names, err := findingFiles(dir)
	if err != nil || len(names) != 1 {
		t.Fatalf("finding files=%v err=%v", names, err)
	}
	f, err := parseFinding(filepath.Join(dir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "SQL injection in login" || f.Severity != "high" || f.Status != "open" || f.Asset != "example.com/login" || f.CVSS != "8.8" || f.CWE != "CWE-89" || f.OWASP != "A03:2021 Injection" {
		t.Fatalf("unexpected interactive finding: %+v", f)
	}
	if !strings.Contains(f.Body, "Unsanitized input reaches a SQL query.") || !strings.Contains(f.Body, "Use parameterized queries.") {
		t.Fatalf("interactive report sections were not populated:\n%s", f.Body)
	}
	for _, prompt := range []string{"Title:", "Severity", "Status", "Affected asset", "CVSS", "CWE", "OWASP", "Description", "Business impact", "Technical impact", "Steps to reproduce", "Remediation", "References"} {
		if !strings.Contains(output.String(), prompt) {
			t.Errorf("interactive output missing %q: %s", prompt, output.String())
		}
	}
}

func TestFindingSetInteractive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "findings")
	if err := findingAdd(dir, []string{"SQL injection", "--severity", "high", "--asset", "example.com/login", "--cvss", "8.8", "--cwe", "CWE-89", "--owasp", "A03:2021 Injection"}); err != nil {
		t.Fatal(err)
	}

	oldInput, oldOutput := findingInput, findingOutput
	t.Cleanup(func() { findingInput, findingOutput = oldInput, oldOutput })
	findingInput = strings.NewReader("\nresolved\n-\n\n\n-\nstill-vulnerable\n2026-08-20\n")
	var output bytes.Buffer
	findingOutput = &output

	if err := findingSet(dir, []string{"1", "--interactive"}); err != nil {
		t.Fatal(err)
	}
	names, err := findingFiles(dir)
	if err != nil || len(names) != 1 {
		t.Fatalf("finding files=%v err=%v", names, err)
	}
	f, err := parseFinding(filepath.Join(dir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if f.Severity != "high" || f.Status != "resolved" || f.Asset != "" || f.CVSS != "8.8" || f.CWE != "CWE-89" || f.OWASP != "" || f.RetestStatus != "still-vulnerable" || f.Retested != "2026-08-20" {
		t.Fatalf("unexpected updated finding: %+v", f)
	}
	for _, want := range []string{"Interactive finding update", "[high]", "[example.com/login]", "Retest status", "Retested date"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("interactive output missing %q: %s", want, output.String())
		}
	}
}

func TestFindingSetInteractiveRejectsClearingRequiredValue(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "findings")
	if err := findingAdd(dir, []string{"SQL injection"}); err != nil {
		t.Fatal(err)
	}
	oldInput, oldOutput := findingInput, findingOutput
	t.Cleanup(func() { findingInput, findingOutput = oldInput, oldOutput })
	findingInput = strings.NewReader("-\n\n\n\n\n\n\n\n\n")
	var output bytes.Buffer
	findingOutput = &output

	if err := findingSet(dir, []string{"1", "-i"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "severity cannot be cleared") {
		t.Fatalf("expected validation message, got: %s", output.String())
	}
}

func TestFindingAddListLink(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "findings")

	if err := findingAdd(dir, []string{"SQL", "Injection", "in", "login", "--severity", "high", "--asset", "example.com/login", "--cvss", "8.8", "--cwe", "CWE-89", "--owasp", "A03:2021"}); err != nil {
		t.Fatal(err)
	}
	if err := findingAdd(dir, []string{"Reflected", "XSS"}); err != nil {
		t.Fatal(err)
	}

	names, err := findingFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d finding files, want 2", len(names))
	}
	if names[0][:4] != "0001" || names[1][:4] != "0002" {
		t.Fatalf("unexpected ids: %v", names)
	}

	f, err := parseFinding(filepath.Join(dir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "SQL Injection in login" || f.Severity != "high" || f.Asset != "example.com/login" || f.CVSS != "8.8" || f.Status != "open" {
		t.Fatalf("unexpected parse: %+v", f)
	}
	if err := findingSet(dir, []string{"1", "status", "partially-resolved"}); err != nil {
		t.Fatal(err)
	}
	if err := findingSet(dir, []string{"1", "retest-status", "still-vulnerable"}); err != nil {
		t.Fatal(err)
	}

	evidenceDir := filepath.Join(root, "evidence", "exploitation")
	if err := os.MkdirAll(evidenceDir, 0o750); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(evidenceDir, "shot.png")
	if err := os.WriteFile(evidencePath, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSidecar(evidencePath, sidecar{Timestamp: "now", Host: "host", Type: "exploitation", SHA256: sum}); err != nil {
		t.Fatal(err)
	}
	if err := findingLink(dir, []string{"1", "evidence/exploitation/shot.png"}); err != nil {
		t.Fatal(err)
	}
	retestPath := filepath.Join(evidenceDir, "retest.png")
	if err := os.WriteFile(retestPath, []byte("retest png"), 0o600); err != nil {
		t.Fatal(err)
	}
	retestSum, err := fileSHA256(retestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSidecar(retestPath, sidecar{Timestamp: "later", Host: "host", Type: "exploitation", SHA256: retestSum}); err != nil {
		t.Fatal(err)
	}
	if err := findingLink(dir, []string{"1", "evidence/exploitation/retest.png", "--retest"}); err != nil {
		t.Fatal(err)
	}
	f, err = parseFinding(filepath.Join(dir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Evidence) != 1 || f.Evidence[0] != "evidence/exploitation/shot.png" {
		t.Fatalf("evidence not linked: %+v", f)
	}
	if len(f.RetestEvidence) != 1 || f.RetestEvidence[0] != "evidence/exploitation/retest.png" {
		t.Fatalf("retest evidence not linked: %+v", f)
	}
	if f.Status != "partially-resolved" || f.RetestStatus != "still-vulnerable" {
		t.Fatalf("finding metadata was not updated: %+v", f)
	}
	if f.Body == "" {
		t.Fatal("expected body content to survive the link insert")
	}
}

func TestFindingRejectsBadOptionsAndLinks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "findings")
	if err := findingAdd(dir, []string{"title", "--severity"}); err == nil {
		t.Fatal("expected missing severity value to fail")
	}
	if err := findingAdd(dir, []string{"title", "--unknown"}); err == nil {
		t.Fatal("expected unknown option to fail")
	}
	if err := findingAdd(dir, []string{"title", "--cvss", "11"}); err == nil {
		t.Fatal("expected invalid CVSS to fail")
	}
	if err := findingAdd(dir, []string{"bad\ntitle"}); err == nil {
		t.Fatal("expected multiline title to fail")
	}
	if err := findingAdd(dir, []string{"Valid"}); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := findingLink(dir, []string{"1", outside}); err == nil {
		t.Fatal("expected outside evidence link to fail")
	}
	if err := findingLink(dir, []string{"1", "evidence/missing.png"}); err == nil {
		t.Fatal("expected missing evidence link to fail")
	}
}

func TestConcurrentFindingAddAllocatesUniqueIDs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "findings")
	const count = 20
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- findingAdd(dir, []string{"Concurrent"})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	names, err := findingFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != count {
		t.Fatalf("got %d findings, want %d", len(names), count)
	}
}
