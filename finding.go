package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var findingFileRe = regexp.MustCompile(`^(\d+)-`)

var validSeverities = map[string]bool{"crit": true, "high": true, "med": true, "low": true, "info": true}
var validFindingStatuses = map[string]bool{"open": true, "resolved": true, "partially-resolved": true, "accepted-risk": true, "not-applicable": true}
var validRetestStatuses = map[string]bool{"not-tested": true, "resolved": true, "still-vulnerable": true, "partially-resolved": true}

var (
	findingInput  io.Reader = os.Stdin
	findingOutput io.Writer = os.Stdout
)

type finding struct {
	ID             string
	Title          string
	Severity       string
	Status         string
	Asset          string
	CVSS           string
	CWE            string
	OWASP          string
	Created        string
	RetestStatus   string
	Retested       string
	Evidence       []string
	RetestEvidence []string
	Body           string
}

func cmdFinding(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dorocap finding <add|set|link|list>")
	}

	root, err := findRoot()
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "findings")

	switch args[0] {
	case "add":
		return findingAdd(dir, args[1:])
	case "link":
		return findingLink(dir, args[1:])
	case "set":
		return findingSet(dir, args[1:])
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: dorocap finding list")
		}
		return findingList(dir)
	default:
		return fmt.Errorf("unknown finding subcommand %q", args[0])
	}
}

func findingAdd(dir string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dorocap finding add <title> [--severity value] [--asset value] [--status value] [--cvss value] [--cwe value] [--owasp value]")
	}
	if len(args) == 1 && (args[0] == "--interactive" || args[0] == "-i") {
		return findingAddInteractive(dir)
	}
	for _, arg := range args {
		if arg == "--interactive" || arg == "-i" {
			return fmt.Errorf("--interactive cannot be combined with a title or other options")
		}
	}

	values := map[string]string{"severity": "info", "status": "open", "retest_status": "not-tested"}
	var titleParts []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			option := strings.TrimPrefix(args[i], "--")
			value := ""
			if key, inline, ok := strings.Cut(option, "="); ok {
				option, value = key, inline
			} else {
				if i+1 >= len(args) {
					return fmt.Errorf("--%s requires a value", option)
				}
				i++
				value = args[i]
			}
			key := strings.ReplaceAll(option, "-", "_")
			switch key {
			case "severity", "status", "asset", "cvss", "cwe", "owasp":
			default:
				return fmt.Errorf("unknown option %q", "--"+option)
			}
			if value == "" {
				return fmt.Errorf("--%s requires a value", option)
			}
			values[key] = value
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			return fmt.Errorf("unknown option %q", args[i])
		}
		titleParts = append(titleParts, args[i])
	}
	title := strings.Join(titleParts, " ")
	if title == "" {
		return fmt.Errorf("title required")
	}
	for key, value := range values {
		if err := validateFindingMetadata(key, value); err != nil {
			return err
		}
	}
	if strings.ContainsAny(title, "\r\n\x00") {
		return fmt.Errorf("title must be a single line")
	}

	return writeNewFinding(dir, title, values, nil)
}

func findingAddInteractive(dir string) error {
	reader := bufio.NewReader(findingInput)
	if _, err := fmt.Fprintln(findingOutput, interactivePaint("Interactive finding setup", clrBold+clrBlue)+interactivePaint(" (press Enter to accept a default or skip an optional field).", clrMuted)); err != nil {
		return err
	}

	title, err := promptFindingValue(reader, "Title", "", func(value string) error {
		if value == "" {
			return fmt.Errorf("title is required")
		}
		return nil
	})
	if err != nil {
		return err
	}
	severity, err := promptFindingValue(reader, "Severity (crit/high/med/low/info)", "info", func(value string) error {
		return validateFindingMetadata("severity", value)
	})
	if err != nil {
		return err
	}
	status, err := promptFindingValue(reader, "Status (open/resolved/partially-resolved/accepted-risk/not-applicable)", "open", func(value string) error {
		return validateFindingMetadata("status", value)
	})
	if err != nil {
		return err
	}
	asset, err := promptFindingValue(reader, "Affected asset (host, URL, IP, endpoint, or service)", "", nil)
	if err != nil {
		return err
	}
	cvss, err := promptFindingValue(reader, "CVSS score (0.0-10.0)", "", func(value string) error {
		return validateFindingMetadata("cvss", value)
	})
	if err != nil {
		return err
	}
	cwe, err := promptFindingValue(reader, "CWE (for example CWE-89)", "", nil)
	if err != nil {
		return err
	}
	owasp, err := promptFindingValue(reader, "OWASP category", "", nil)
	if err != nil {
		return err
	}
	sections := map[string]string{}
	for _, section := range []struct {
		key   string
		label string
	}{
		{"description", "Description"},
		{"business_impact", "Business impact"},
		{"technical_impact", "Technical impact"},
		{"steps", "Steps to reproduce"},
		{"remediation", "Remediation"},
		{"references", "References"},
	} {
		value, promptErr := promptFindingValue(reader, section.label, "", nil)
		if promptErr != nil {
			return promptErr
		}
		sections[section.key] = value
	}
	values := map[string]string{
		"severity": severity, "status": status, "asset": asset,
		"cvss": cvss, "cwe": cwe, "owasp": owasp,
	}
	return writeNewFinding(dir, title, values, sections)
}

func promptFindingValue(reader *bufio.Reader, label, defaultValue string, validate func(string) error) (string, error) {
	for {
		if _, err := fmt.Fprint(findingOutput, interactivePaint(label, clrCyan)); err != nil {
			return "", err
		}
		if defaultValue != "" {
			if _, err := fmt.Fprint(findingOutput, interactivePaint(" ["+defaultValue+"]", clrMuted)); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprint(findingOutput, ": "); err != nil {
			return "", err
		}
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			if err == io.EOF {
				return "", fmt.Errorf("interactive input ended while reading %s", strings.ToLower(label))
			}
			return "", err
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = defaultValue
		}
		if validate != nil {
			if validationErr := validate(value); validationErr != nil {
				if _, err := fmt.Fprintf(findingOutput, "  %s %v\n", interactivePaint("Invalid value:", clrRed), validationErr); err != nil {
					return "", err
				}
				continue
			}
		}
		return value, nil
	}
}

func interactivePaint(text, color string) string {
	if findingOutput == os.Stdout {
		return paint(os.Stdout, text, color)
	}
	return text
}

func writeNewFinding(dir, title string, values, sections map[string]string) error {
	if err := mkdirAll(dir, 0o750); err != nil {
		return err
	}
	section := func(key string) string {
		if sections == nil || strings.TrimSpace(sections[key]) == "" {
			return ""
		}
		return strings.TrimSpace(sections[key]) + "\n"
	}

	var path string
	err := withFileLock(filepath.Join(dir, ".finding-ids"), func() error {
		id, err := nextFindingID(dir)
		if err != nil {
			return err
		}
		path = filepath.Join(dir, fmt.Sprintf("%04d-%s.md", id, slugify(title)))
		content := fmt.Sprintf(`id: %04d
title: %s
severity: %s
status: %s
asset: %s
cvss: %s
cwe: %s
owasp: %s
created: %s
retest_status: not-tested
retested:

## Description

%s
## Business Impact

%s
## Technical Impact

%s
## Steps to Reproduce

%s
## Remediation

%s
## References

%s
## Retest Notes

`, id, title, values["severity"], values["status"], values["asset"], values["cvss"], values["cwe"], values["owasp"], time.Now().UTC().Format(time.RFC3339Nano), section("description"), section("business_impact"), section("technical_impact"), section("steps"), section("remediation"), section("references"))
		return writeFileAtomic(path, []byte(content), 0o600)
	})
	if err != nil {
		return err
	}
	printOK("saved %s", path)
	return nil
}

func validateFindingMetadata(key, value string) error {
	if strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("%s must be a single line", strings.ReplaceAll(key, "_", " "))
	}
	switch key {
	case "severity":
		if !validSeverities[value] {
			return fmt.Errorf("invalid severity %q: must be one of crit, high, med, low, info", value)
		}
	case "status":
		if !validFindingStatuses[value] {
			return fmt.Errorf("invalid status %q: must be one of open, resolved, partially-resolved, accepted-risk, not-applicable", value)
		}
	case "retest_status":
		if !validRetestStatuses[value] {
			return fmt.Errorf("invalid retest status %q: must be one of not-tested, resolved, still-vulnerable, partially-resolved", value)
		}
	case "cvss":
		if value == "" {
			return nil
		}
		score, err := strconv.ParseFloat(value, 64)
		if err != nil || score < 0 || score > 10 {
			return fmt.Errorf("invalid CVSS %q: use a score from 0.0 to 10.0", value)
		}
	}
	return nil
}

func findingSet(dir string, args []string) error {
	if len(args) == 2 && (args[1] == "--interactive" || args[1] == "-i") {
		return findingSetInteractive(dir, args[0])
	}
	if len(args) < 3 {
		return fmt.Errorf("usage: dorocap finding set <id> --interactive\n   or: dorocap finding set <id> <severity|status|asset|cvss|cwe|owasp|retest-status|retested> <value>")
	}
	path, err := findingPath(dir, args[0])
	if err != nil {
		return err
	}
	key := strings.ReplaceAll(args[1], "-", "_")
	switch key {
	case "severity", "status", "asset", "cvss", "cwe", "owasp", "retest_status", "retested":
	default:
		return fmt.Errorf("unsupported finding field %q", args[1])
	}
	value := strings.Join(args[2:], " ")
	if value == "-" {
		value = ""
	}
	if err := validateFindingMetadata(key, value); err != nil {
		return err
	}
	if err := updateFindingMetadata(path, map[string]string{key: value}); err != nil {
		return err
	}
	printOK("set %s on %s", strings.ReplaceAll(key, "_", " "), filepath.Base(path))
	return nil
}

func findingSetInteractive(dir, id string) error {
	path, err := findingPath(dir, id)
	if err != nil {
		return err
	}
	current, err := parseFinding(path)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(findingInput)
	if _, err := fmt.Fprintln(findingOutput, interactivePaint("Interactive finding update: ", clrBold+clrBlue)+interactivePaint(current.ID+" - "+current.Title, clrBold)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(findingOutput, interactivePaint("Press Enter to keep the current value. Enter - to clear an optional field.", clrMuted)); err != nil {
		return err
	}

	values := map[string]string{}
	fields := []struct {
		key      string
		label    string
		current  string
		optional bool
	}{
		{"severity", "Severity (crit/high/med/low/info)", current.Severity, false},
		{"status", "Status (open/resolved/partially-resolved/accepted-risk/not-applicable)", current.Status, false},
		{"asset", "Affected asset (host, URL, IP, endpoint, or service)", current.Asset, true},
		{"cvss", "CVSS score (0.0-10.0)", current.CVSS, true},
		{"cwe", "CWE (for example CWE-89)", current.CWE, true},
		{"owasp", "OWASP category", current.OWASP, true},
		{"retest_status", "Retest status (not-tested/resolved/still-vulnerable/partially-resolved)", current.RetestStatus, false},
		{"retested", "Retested date", current.Retested, true},
	}
	for _, field := range fields {
		validate := func(value string) error {
			if value == "-" {
				if field.optional {
					return nil
				}
				return fmt.Errorf("%s cannot be cleared", strings.ReplaceAll(field.key, "_", " "))
			}
			return validateFindingMetadata(field.key, value)
		}
		value, promptErr := promptFindingValue(reader, field.label, field.current, validate)
		if promptErr != nil {
			return promptErr
		}
		if value == "-" {
			value = ""
		}
		values[field.key] = value
	}

	if err := updateFindingMetadata(path, values); err != nil {
		return err
	}
	printOK("updated finding %s", current.ID)
	return nil
}

func updateFindingMetadata(path string, values map[string]string) error {
	return withFileLock(path, func() error {
		b, err := readFileIn(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(b), "\n")
		pending := make(map[string]string, len(values))
		for key, value := range values {
			pending[key] = value
		}
		insertAt := len(lines)
		for i, line := range lines {
			if line == "" {
				insertAt = i
				break
			}
			key, _, found := strings.Cut(line, ":")
			if !found {
				continue
			}
			if value, ok := pending[key]; ok {
				lines[i] = key + ": " + value
				delete(pending, key)
			}
		}
		if len(pending) > 0 {
			keys := make([]string, 0, len(pending))
			for key := range pending {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			insert := make([]string, 0, len(keys))
			for _, key := range keys {
				insert = append(insert, key+": "+pending[key])
			}
			lines = append(lines[:insertAt], append(insert, lines[insertAt:]...)...)
		}
		return writeFileAtomic(path, []byte(strings.Join(lines, "\n")), 0o600)
	})
}

func nextFindingID(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, err
	}

	max := 0
	for _, e := range entries {
		m := findingFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n > max {
			max = n
		}
	}
	return max + 1, nil
}

func findingPath(dir, id string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	padded := id
	if n, err := strconv.Atoi(id); err == nil {
		padded = fmt.Sprintf("%04d", n)
	}
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasPrefix(e.Name(), padded+"-") && strings.HasSuffix(e.Name(), ".md") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no finding with id %s", id)
}

func findingLink(dir string, args []string) error {
	if len(args) < 2 || len(args) > 3 || (len(args) == 3 && args[2] != "--retest") {
		return fmt.Errorf("usage: dorocap finding link <id> <evidence-path> [--retest]")
	}
	retest := len(args) == 3
	path, err := findingPath(dir, args[0])
	if err != nil {
		return err
	}

	root := filepath.Dir(dir)
	evidenceRoot := filepath.Join(root, "evidence")
	evidencePath := args[1]
	if !filepath.IsAbs(evidencePath) {
		evidencePath = filepath.Join(root, filepath.FromSlash(evidencePath))
	}
	realEvidenceRoot, err := filepath.EvalSymlinks(evidenceRoot)
	if err != nil {
		return err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	realEvidencePath, err := filepath.EvalSymlinks(evidencePath)
	if err != nil {
		return fmt.Errorf("invalid evidence path: %w", err)
	}
	if err := ensureWithin(realEvidenceRoot, realEvidencePath); err != nil {
		return fmt.Errorf("invalid evidence path: %w", err)
	}
	info, err := statIn(realEvidencePath)
	isSidecar := false
	if strings.HasSuffix(realEvidencePath, ".json") {
		if original, statErr := statIn(strings.TrimSuffix(realEvidencePath, ".json")); statErr == nil && original.Mode().IsRegular() {
			isSidecar = true
		}
	}
	if err != nil || !info.Mode().IsRegular() || isSidecar {
		return fmt.Errorf("evidence path must be a captured regular file, not a sidecar")
	}
	if _, err := validateEvidenceFile(realEvidencePath); err != nil {
		return fmt.Errorf("evidence failed verification: %w", err)
	}
	rel, err := filepath.Rel(realRoot, realEvidencePath)
	if err != nil {
		return err
	}
	storedEvidence := filepath.ToSlash(rel)

	err = withFileLock(path, func() error {
		b, err := readFileIn(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(b), "\n")
		insertAt := len(lines)
		for i, l := range lines {
			if l == "evidence: "+storedEvidence || l == "retest_evidence: "+storedEvidence {
				return fmt.Errorf("evidence already linked")
			}
			if l == "" && insertAt == len(lines) {
				insertAt = i
			}
		}
		prefix := "evidence: "
		if retest {
			prefix = "retest_evidence: "
		}
		line := prefix + storedEvidence
		lines = append(lines[:insertAt], append([]string{line}, lines[insertAt:]...)...)
		return writeFileAtomic(path, []byte(strings.Join(lines, "\n")), 0o600)
	})
	if err != nil {
		return err
	}
	linkType := "evidence"
	if retest {
		linkType = "retest evidence"
	}
	printOK("linked %s as %s to %s", storedEvidence, linkType, filepath.Base(path))
	return nil
}

func parseFinding(path string) (finding, error) {
	b, err := readFileIn(path)
	if err != nil {
		return finding{}, err
	}
	lines := strings.Split(string(b), "\n")

	var f finding
	f.Status = "open"
	f.RetestStatus = "not-tested"
	i := 0
	for ; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			i++
			break
		}
		switch {
		case strings.HasPrefix(line, "id: "):
			f.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "title: "):
			f.Title = strings.TrimPrefix(line, "title: ")
		case strings.HasPrefix(line, "severity: "):
			f.Severity = strings.TrimPrefix(line, "severity: ")
		case strings.HasPrefix(line, "status: "):
			f.Status = strings.TrimPrefix(line, "status: ")
		case strings.HasPrefix(line, "asset: "):
			f.Asset = strings.TrimPrefix(line, "asset: ")
		case strings.HasPrefix(line, "cvss: "):
			f.CVSS = strings.TrimPrefix(line, "cvss: ")
		case strings.HasPrefix(line, "cwe: "):
			f.CWE = strings.TrimPrefix(line, "cwe: ")
		case strings.HasPrefix(line, "owasp: "):
			f.OWASP = strings.TrimPrefix(line, "owasp: ")
		case strings.HasPrefix(line, "created: "):
			f.Created = strings.TrimPrefix(line, "created: ")
		case strings.HasPrefix(line, "retest_status: "):
			f.RetestStatus = strings.TrimPrefix(line, "retest_status: ")
		case strings.HasPrefix(line, "retested: "):
			f.Retested = strings.TrimPrefix(line, "retested: ")
		case strings.HasPrefix(line, "evidence: "):
			f.Evidence = append(f.Evidence, strings.TrimPrefix(line, "evidence: "))
		case strings.HasPrefix(line, "retest_evidence: "):
			f.RetestEvidence = append(f.RetestEvidence, strings.TrimPrefix(line, "retest_evidence: "))
		}
	}
	f.Body = strings.TrimRight(strings.Join(lines[i:], "\n"), "\n")
	if f.ID == "" || f.Title == "" || !validSeverities[f.Severity] || !validFindingStatuses[f.Status] || !validRetestStatuses[f.RetestStatus] {
		return finding{}, fmt.Errorf("malformed finding %s", filepath.Base(path))
	}
	return f, nil
}

func findingFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func findingList(dir string) error {
	names, err := findingFiles(dir)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		printInfo("no findings yet")
		return nil
	}
	for _, n := range names {
		f, err := parseFinding(filepath.Join(dir, n))
		if err != nil {
			return err
		}
		asset := f.Asset
		if asset == "" {
			asset = "-"
		}
		fmt.Printf("%s  %s  %s  %s  %s  %s\n",
			paint(os.Stdout, fmt.Sprintf("%-4s", f.ID), clrCyan),
			paint(os.Stdout, fmt.Sprintf("%-8s", f.Severity), severityColor(f.Severity)),
			paint(os.Stdout, fmt.Sprintf("%-19s", f.Status), clrMuted),
			paint(os.Stdout, fmt.Sprintf("%-40s", f.Title), clrText),
			paint(os.Stdout, fmt.Sprintf("%-28s", asset), clrCyanDeep),
			paint(os.Stdout, fmt.Sprintf("(%d evidence)", len(f.Evidence)), clrMuted))
	}
	return nil
}
