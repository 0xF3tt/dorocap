package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestInitPathInfoAndScope(t *testing.T) {
	parent := t.TempDir()
	withWorkingDirectory(t, parent)
	config := filepath.Join(t.TempDir(), "config")
	t.Setenv("DOROCAP_CONFIG", config)
	if err := cmdInit([]string{"acme-2026"}); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "acme-2026")
	scope, err := loadScope(root)
	if err != nil || scope.Engagement != "acme-2026" || scope.PreparedBy != "@0xF3tt" || len(scope.Methodology) == 0 {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
	if engagementName(root) != "acme-2026" {
		t.Fatal("engagement name was not parsed through YAML")
	}
	configuredRoot, err := loadConfigRoot()
	if err != nil {
		t.Fatal(err)
	}
	configuredInfo, err := os.Stat(configuredRoot)
	if err != nil {
		t.Fatal(err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(configuredInfo, rootInfo) {
		t.Fatalf("init configured global root %q, want %q", configuredRoot, root)
	}
	if err := cmdPath([]string{root}); err != nil {
		t.Fatal(err)
	}
	if err := cmdPath(nil); err != nil {
		t.Fatal(err)
	}
	if err := cmdInfo(nil); err != nil {
		t.Fatal(err)
	}
	if err := cmdInit([]string{"acme-2026"}); err == nil {
		t.Fatal("expected duplicate engagement to fail")
	}
	if err := cmdPath([]string{parent}); err == nil {
		t.Fatal("expected non-engagement path to fail")
	}
}

func TestScreenshotImportCommandAndVerify(t *testing.T) {
	root := scaffoldTestEngagement(t)
	src := filepath.Join(t.TempDir(), "capture.bin")
	if err := os.WriteFile(src, []byte("capture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdScreenshot([]string{"file", src, "packet", "capture"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdVerify(nil); err != nil {
		t.Fatal(err)
	}
	if err := cmdScreenshot(nil); err == nil {
		t.Fatal("expected empty screenshot command to fail")
	}
	if err := cmdScreenshot([]string{"custom"}); err == nil {
		t.Fatal("expected unsupported category to fail")
	}
	evidence, _ := inspectEvidence(root)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(evidence[0].Path)), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdVerify(nil); err == nil {
		t.Fatal("expected verification failure after tampering")
	}
}

func TestFindingCommandsAndList(t *testing.T) {
	_ = scaffoldTestEngagement(t)
	if err := cmdFinding([]string{"list"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdFinding([]string{"add", "TLS", "issue", "--severity=med"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdFinding([]string{"set", "1", "asset", "example.com/login"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdFinding([]string{"list"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdFinding([]string{"unknown"}); err == nil {
		t.Fatal("expected unknown subcommand to fail")
	}
	if err := cmdFinding(nil); err == nil {
		t.Fatal("expected missing subcommand to fail")
	}
}

func TestMalformedScopeAndNonColorBanner(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, markerFile), []byte("engagement: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadScope(root); err == nil {
		t.Fatal("expected malformed YAML to fail")
	}
	t.Setenv("NO_COLOR", "1")
	if got := banner(); !strings.HasPrefix(got, "+---") || !strings.Contains(got, "D O R O C A P") || !strings.Contains(got, "@0xF3tt") {
		t.Fatalf("unexpected banner %q", got)
	}
}

func TestEngagementTreeDescribesScaffold(t *testing.T) {
	t.Setenv("DOROCAP_ASCII", "1")
	tree := engagementTree("acme-2026", "/engagements/acme-2026")
	for _, want := range []string{
		"acme-2026/",
		"scope.yaml",
		"evidence/",
		"recon/",
		"staging/",
		"exploitation/",
		"postex/",
		"files/",
		"notes/",
		"findings/",
		"draft/",
		"final/",
		"Active global engagement: /engagements/acme-2026",
	} {
		if !strings.Contains(tree, want) {
			t.Errorf("tree missing %q:\n%s", want, tree)
		}
	}
	if !strings.Contains(tree, "|-- scope.yaml") || strings.Contains(tree, "├") {
		t.Fatalf("ASCII tree fallback was not used:\n%s", tree)
	}
}

func TestTerminalColorModeCompatibility(t *testing.T) {
	tests := []struct {
		name string
		tty  bool
		goos string
		env  map[string]string
		want terminalColorMode
	}{
		{"redirected", false, "linux", map[string]string{"COLORTERM": "truecolor"}, colorNone},
		{"no color", true, "linux", map[string]string{"NO_COLOR": ""}, colorNone},
		{"dumb terminal", true, "linux", map[string]string{"TERM": "dumb"}, colorNone},
		{"unix true color", true, "linux", map[string]string{"COLORTERM": "truecolor"}, colorTrue},
		{"unix ansi fallback", true, "darwin", map[string]string{"TERM": "xterm-256color"}, colorANSI16},
		{"windows terminal", true, "windows", map[string]string{"WT_SESSION": "session"}, colorTrue},
		{"windows ansi console", true, "windows", map[string]string{"ANSICON": "1"}, colorANSI16},
		{"legacy windows cmd", true, "windows", map[string]string{}, colorNone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := terminalColorModeFor(test.tty, test.goos, test.env); got != test.want {
				t.Fatalf("mode=%v, want %v", got, test.want)
			}
		})
	}
}

func TestTerminalUnicodeCompatibility(t *testing.T) {
	tests := []struct {
		name string
		tty  bool
		goos string
		env  map[string]string
		want bool
	}{
		{"modern unix", true, "linux", map[string]string{"LANG": "en_US.UTF-8"}, true},
		{"ascii locale", true, "linux", map[string]string{"LANG": "C"}, false},
		{"forced ascii", true, "darwin", map[string]string{"DOROCAP_ASCII": ""}, false},
		{"redirected", false, "linux", map[string]string{"LANG": "en_US.UTF-8"}, false},
		{"windows terminal", true, "windows", map[string]string{"WT_SESSION": "session"}, true},
		{"legacy windows", true, "windows", map[string]string{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := terminalUnicodeFor(test.tty, test.goos, test.env); got != test.want {
				t.Fatalf("unicode=%v, want %v", got, test.want)
			}
		})
	}
}
