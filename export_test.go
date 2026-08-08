package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scaffoldTestEngagement(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "acme-2026")
	for _, dir := range []string{"findings", "notes", "report/draft", "evidence/files", "evidence/recon", "evidence/staging", "evidence/exploitation", "evidence/postex"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	scope := `engagement: acme-2026
client: Acme Corp
start_date: "2026-08-01"
end_date: "2026-08-05"
in_scope: ["example.com"]
out_of_scope: ["billing.example.com"]
contacts: ["Security Team"]
assessment_type: Web Application Penetration Test
testing_mode: Gray-box
classification: Confidential
report_version: "0.9"
prepared_by: "@0xF3tt"
executive_summary: "Acme requested an assessment of its external application."
overall_posture: "Elevated risk"
restrictions: ["No denial-of-service testing"]
emergency_stop_conditions: ["Stop on service instability"]
limitations: ["Payment processing was unavailable"]
methodology: ["Reconnaissance", "Exploitation", "Reporting"]
revision_history:
  - version: "0.9"
    date: "2026-08-06"
    author: "@0xF3tt"
    changes: "Initial draft"
`
	if err := os.WriteFile(filepath.Join(root, markerFile), []byte(scope), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(t.TempDir(), "config")
	t.Setenv("DOROCAP_CONFIG", config)
	if err := saveConfigRoot(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestExportIncludesScopeNotesHashesAndWarnings(t *testing.T) {
	root := scaffoldTestEngagement(t)
	src := filepath.Join(t.TempDir(), "request.txt")
	if err := os.WriteFile(src, []byte("GET /admin"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileEvidence(root, src, "admin request"); err != nil {
		t.Fatal(err)
	}
	evidence, issues := inspectEvidence(root)
	if len(evidence) != 1 || len(issues) != 0 {
		t.Fatalf("evidence=%v issues=%v", evidence, issues)
	}
	findingsDir := filepath.Join(root, "findings")
	if err := findingAdd(findingsDir, []string{"Admin", "exposure", "--severity", "high", "--asset", "example.com/admin", "--cvss", "8.1", "--cwe", "CWE-200", "--owasp", "A01:2021"}); err != nil {
		t.Fatal(err)
	}
	if err := findingLink(findingsDir, []string{"1", evidence[0].Path}); err != nil {
		t.Fatal(err)
	}
	if err := cmdNote([]string{"recon", "Discovered", "admin", "endpoint"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdExport(nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "report", "draft", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	report := string(b)
	for _, want := range []string{"Web Application Penetration Test Report", "Document Control", "Executive Summary", "Elevated risk", "Risk Summary", "Gray-box", "Stop on service instability", "example.com/admin", "8.1", "CWE-200", "A01:2021", "Methodology", "Payment processing was unavailable", "Remediation Roadmap", "Admin exposure", "HIGH", evidence[0].Sidecar.SHA256, "Discovered admin endpoint", "No integrity, link, or orphan warnings"} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q", want)
		}
	}
	if err := cmdFinalize(nil); err != nil {
		t.Fatal(err)
	}
	final, err := os.ReadFile(filepath.Join(root, "report", "final", "report.md"))
	if err != nil || string(final) != report {
		t.Fatalf("final report mismatch err=%v", err)
	}
}

func TestExportReportsBrokenAndUnlinkedEvidence(t *testing.T) {
	root := scaffoldTestEngagement(t)
	src := filepath.Join(t.TempDir(), "proof.txt")
	if err := os.WriteFile(src, []byte("proof"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileEvidence(root, src, ""); err != nil {
		t.Fatal(err)
	}
	if err := findingAdd(filepath.Join(root, "findings"), []string{"Broken"}); err != nil {
		t.Fatal(err)
	}
	names, _ := findingFiles(filepath.Join(root, "findings"))
	path := filepath.Join(root, "findings", names[0])
	b, _ := os.ReadFile(path)
	b = []byte(strings.Replace(string(b), "\n\n", "\nevidence: evidence/files/missing.txt\n\n", 1))
	if err := writeFileAtomic(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdExport(nil); err != nil {
		t.Fatal(err)
	}
	reportBytes, _ := os.ReadFile(filepath.Join(root, "report", "draft", "report.md"))
	report := string(reportBytes)
	if !strings.Contains(report, "Broken evidence link") || !strings.Contains(report, "Unlinked evidence") {
		t.Fatalf("warnings missing from report:\n%s", report)
	}
}

func TestExportEmbedsImagesAndLinksOtherEvidence(t *testing.T) {
	root := scaffoldTestEngagement(t)
	imagePath := filepath.Join(root, "evidence", "exploitation", "cookie[admin].png")
	if err := os.WriteFile(imagePath, []byte("png evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	imageSum, err := fileSHA256(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSidecar(imagePath, sidecar{Timestamp: "2026-08-07T00:00:00Z", Host: "host", Type: "exploitation", Note: "Admin cookie", SHA256: imageSum}); err != nil {
		t.Fatal(err)
	}

	textSource := filepath.Join(t.TempDir(), "request.txt")
	if err := os.WriteFile(textSource, []byte("GET /admin"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileEvidence(root, textSource, "request"); err != nil {
		t.Fatal(err)
	}
	if err := findingAdd(filepath.Join(root, "findings"), []string{"Session", "exposure"}); err != nil {
		t.Fatal(err)
	}
	evidence, issues := inspectEvidence(root)
	if len(issues) != 0 || len(evidence) != 2 {
		t.Fatalf("evidence=%v issues=%v", evidence, issues)
	}
	for _, item := range evidence {
		if err := findingLink(filepath.Join(root, "findings"), []string{"0001", item.Path}); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmdExport(nil); err != nil {
		t.Fatal(err)
	}
	reportBytes, err := os.ReadFile(filepath.Join(root, "report", "draft", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	report := string(reportBytes)
	if !strings.Contains(report, "![Admin cookie](../../evidence/exploitation/cookie%5Badmin%5D.png)") {
		t.Fatalf("image was not embedded:\n%s", report)
	}
	if !strings.Contains(report, "](../../evidence/files/") {
		t.Fatalf("non-image evidence was not linked:\n%s", report)
	}
}

func TestCommandArgumentValidation(t *testing.T) {
	_ = scaffoldTestEngagement(t)
	for name, fn := range map[string]func() error{
		"export":   func() error { return cmdExport([]string{"extra"}) },
		"finalize": func() error { return cmdFinalize([]string{"extra"}) },
		"verify":   func() error { return cmdVerify([]string{"extra"}) },
		"info":     func() error { return cmdInfo([]string{"extra"}) },
		"list":     func() error { return cmdFinding([]string{"list", "extra"}) },
		"bad note": func() error { return cmdNote([]string{"custom", "text"}) },
	} {
		if err := fn(); err == nil {
			t.Errorf("%s accepted invalid arguments", name)
		}
	}
}
