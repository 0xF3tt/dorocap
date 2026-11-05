package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

const markerFile = "scope.yaml"

var (
	userHomeDir = os.UserHomeDir
	pathOwner   = statPathUID
	currentUID  = os.Getuid
)

type ownerError struct {
	path string
}

func (e *ownerError) Error() string {
	return e.path + " is not owned by the current user"
}

var evidenceTypes = []string{"recon", "staging", "exploitation", "postex", "files"}

var plainNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)

func hasEngagementMarker(dir string) bool {
	info, err := statIn(filepath.Join(dir, markerFile))
	return err == nil && info.Mode().IsRegular()
}

func validateName(kind, s string) error {
	if !plainNameRe.MatchString(s) || s != filepath.Base(s) || strings.Contains(s, "..") {
		return fmt.Errorf("invalid %s %q: use 1-80 letters, numbers, dots, underscores, or hyphens", kind, s)
	}
	return nil
}

func validateCategory(kind, s string, allowFile bool) error {
	if err := validateName(kind, s); err != nil {
		return err
	}
	for _, candidate := range evidenceTypes {
		if candidate == s && (allowFile || s != "files") {
			return nil
		}
	}
	return fmt.Errorf("invalid %s %q: must be one of recon, staging, exploitation, postex%s", kind, s, map[bool]string{true: ", files", false: ""}[allowFile])
}

func cmdInit(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: dorocap init <engagement-name>")
	}
	name := args[0]
	if err := validateName("engagement name", name); err != nil {
		return err
	}
	root, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("resolve engagement path: %w", err)
	}

	if _, err := statIn(name); err == nil {
		return fmt.Errorf("%s already exists", name)
	} else if !os.IsNotExist(err) {
		return err
	}

	dirs := []string{"notes", "findings", "report/draft", "report/final"}
	for _, t := range evidenceTypes {
		dirs = append(dirs, filepath.Join("evidence", t))
	}

	tmpRoot, err := os.MkdirTemp(".", ".dorocap-init-*")
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(tmpRoot)
		}
	}()

	if err := chmodIn(tmpRoot, privateDirPerm); err != nil {
		return err
	}
	for _, d := range dirs {
		if err := mkdirAll(filepath.Join(tmpRoot, d), privateDirPerm); err != nil {
			return err
		}
	}

	scope := `# scope.yaml - engagement scope and rules of engagement
engagement: "` + name + `"
client: ""
assessment_type: "Penetration Test"
testing_mode: ""
classification: "Confidential"
report_version: "0.1"
prepared_by: ""
start_date: ""
end_date: ""
executive_summary: ""
overall_posture: ""
in_scope:
  - ""
out_of_scope:
  - ""
contacts:
  - ""
restrictions:
  - ""
emergency_stop_conditions:
  - ""
limitations:
  - ""
methodology:
  - Reconnaissance
  - Enumeration
  - Vulnerability analysis
  - Exploitation
  - Post-exploitation
  - Evidence verification
  - Reporting
revision_history:
  - version: "0.1"
    date: ""
    author: ""
    changes: "Initial draft"
`
	if err := writeFileAtomic(filepath.Join(tmpRoot, markerFile), []byte(scope), 0o600); err != nil {
		return err
	}
	if err := renameIn(tmpRoot, name); err != nil {
		return err
	}
	committed = true
	if err := saveConfigRoot(root); err != nil {
		return fmt.Errorf("engagement scaffolded at %s, but could not set it as the global path: %w", name, err)
	}

	printOK("engagement scaffolded at %s and set as the global path", name)
	fmt.Print(engagementTree(name, root))
	return nil
}

func engagementTree(name, root string) string {
	type treeEntry struct {
		branch      string
		name        string
		description string
	}
	entries := []treeEntry{
		{"├── ", "scope.yaml", "engagement name, client, dates, scope, and contacts"},
		{"├── ", "evidence/", "captured artifacts with SHA-256 JSON sidecars"},
		{"│   ├── ", "recon/", "discovery, enumeration, and mapping evidence"},
		{"│   ├── ", "staging/", "setup, payload preparation, and pre-exploitation evidence"},
		{"│   ├── ", "exploitation/", "proof of exploitation and vulnerability evidence"},
		{"│   ├── ", "postex/", "impact, privilege, persistence, and cleanup evidence"},
		{"│   └── ", "files/", "files imported with \"dorocap ss file\""},
		{"├── ", "notes/", "timestamped notes grouped by engagement phase"},
		{"├── ", "findings/", "editable Markdown vulnerability records"},
		{"└── ", "report/", ""},
		{"    ├── ", "draft/", "generated report from \"dorocap export\""},
		{"    └── ", "final/", "finalized deliverables managed by the operator"},
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(paint(os.Stdout, "Engagement layout:", clrBold+clrBlue))
	b.WriteString("\n\n")
	b.WriteString(paint(os.Stdout, name+"/", clrBold+clrViolet))
	b.WriteString("\n")
	for _, entry := range entries {
		branch := entry.branch
		if !unicodeEnabled(os.Stdout) {
			branch = strings.NewReplacer("│", "|", "├──", "|--", "└──", "`--").Replace(branch)
		}
		b.WriteString(paint(os.Stdout, branch, clrBlue))
		if entry.description == "" {
			b.WriteString(paint(os.Stdout, entry.name, clrCyan))
			b.WriteString("\n")
			continue
		}
		nameWidth := 28 - utf8.RuneCountInString(branch)
		if nameWidth <= utf8.RuneCountInString(entry.name) {
			nameWidth = utf8.RuneCountInString(entry.name) + 1
		}
		b.WriteString(paint(os.Stdout, fmt.Sprintf("%-*s", nameWidth, entry.name), clrCyan))
		b.WriteString(paint(os.Stdout, entry.description, clrMuted))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(paint(os.Stdout, "Active global engagement:", clrCyanDeep))
	b.WriteString(" ")
	b.WriteString(paint(os.Stdout, root, clrText))
	b.WriteString("\n")
	b.WriteString(paint(os.Stdout, "Next:", clrYellow))
	b.WriteString(" edit ")
	b.WriteString(paint(os.Stdout, name+"/scope.yaml", clrCyan))
	b.WriteString(", then capture evidence or add notes and findings.\n\n")
	return b.String()
}

func findRoot() (string, error) {
	dir, err := findRootFromCwd()
	if err == nil {
		return dir, nil
	}
	var owned *ownerError
	if errors.As(err, &owned) {
		return "", err
	}
	cfg, err := loadConfigRoot()
	if err != nil {
		return "", fmt.Errorf("load global path: %w", err)
	}
	if cfg != "" && hasEngagementMarker(cfg) {
		if err := ensureOwnedVault(cfg); err != nil {
			return "", err
		}
		return cfg, nil
	}
	return "", fmt.Errorf("not inside an engagement (no %s found) and no valid global path set; run `dorocap init`, then `dorocap path <dir>`", markerFile)
}

func findRootFromCwd() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err = canonicalPath(dir)
	if err != nil {
		return "", err
	}
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	home, err = canonicalPath(home)
	if err != nil {
		return "", err
	}

	// Walk parents, but never above the home directory. Outside home, only
	// step into a parent the current user owns, so a scope.yaml planted in
	// /tmp cannot capture a command run from a subdirectory.
	for {
		if hasEngagementMarker(dir) {
			if err := ensureOwnedVault(dir); err != nil {
				return "", err
			}
			return dir, nil
		}
		if pathsEqual(dir, home) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if !withinPath(home, dir) {
			if runtime.GOOS == "windows" {
				break
			}
			owned, err := isOwnedByCurrentUser(parent)
			if err != nil || !owned {
				break
			}
		}
		dir = parent
	}
	return "", fmt.Errorf("not inside an engagement (no %s found)", markerFile)
}

func ensureOwnedVault(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	uid, err := pathOwner(dir)
	if err != nil {
		return err
	}
	if uid != currentUID() {
		return &ownerError{path: dir}
	}
	return nil
}

func isOwnedByCurrentUser(path string) (bool, error) {
	uid, err := pathOwner(path)
	if err != nil {
		return false, err
	}
	return uid == currentUID(), nil
}

func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return resolved, nil
}

func pathsEqual(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func withinPath(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
