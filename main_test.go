package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRoot(t *testing.T) {
	tmp := t.TempDir()
	eng := filepath.Join(tmp, "acme-2026")
	sub := filepath.Join(eng, "evidence", "recon")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(eng, markerFile), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()

	if err := os.Chdir(eng); err != nil {
		t.Fatal(err)
	}
	wantRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != wantRoot {
		t.Fatalf("got %s, want %s", root, wantRoot)
	}
}

func TestFindRootMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("DOROCAP_CONFIG", filepath.Join(tmp, "no-such-config"))
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	if _, err := findRoot(); err == nil {
		t.Fatal("expected error when no scope.yaml present")
	}
}

func TestFindRootGlobalFallback(t *testing.T) {
	tmp := t.TempDir()
	eng := filepath.Join(tmp, "acme-2026")
	if err := os.MkdirAll(eng, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(eng, markerFile), []byte("engagement: acme-2026\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DOROCAP_CONFIG", filepath.Join(tmp, "config"))
	if err := saveConfigRoot(eng); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(tmp, "elsewhere")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}
	root, err := findRoot()
	if err != nil {
		t.Fatalf("expected global fallback to resolve, got %v", err)
	}
	if filepath.Base(root) != "acme-2026" {
		t.Fatalf("got %s, want engagement acme-2026", root)
	}
}

func TestValidateName(t *testing.T) {
	bad := []string{"", "../etc", "a/b", "..", "foo/../bar", "bad\nname", "has space"}
	for _, s := range bad {
		if err := validateName("test", s); err == nil {
			t.Errorf("validateName(%q) = nil, want error", s)
		}
	}

	good := []string{"acme-2026", "recon", "file", "notes.v2"}
	for _, s := range good {
		if err := validateName("test", s); err != nil {
			t.Errorf("validateName(%q) = %v, want nil", s, err)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"":                   "shot",
		"admin panel login!": "admin-panel-login",
		"  spaced  ":         "spaced",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
