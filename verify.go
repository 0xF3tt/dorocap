package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type verifiedEvidence struct {
	Path    string
	Sidecar sidecar
}

func validateEvidenceFile(path string) (sidecar, error) {
	info, err := statIn(path)
	if err != nil || !info.Mode().IsRegular() {
		return sidecar{}, fmt.Errorf("not a regular file")
	}
	b, err := readFileIn(path + ".json")
	if err != nil {
		return sidecar{}, fmt.Errorf("missing or unreadable sidecar")
	}
	var sc sidecar
	if err := json.Unmarshal(b, &sc); err != nil {
		return sidecar{}, fmt.Errorf("invalid sidecar JSON")
	}
	if sc.Timestamp == "" || sc.Host == "" || sc.Type == "" || sc.SHA256 == "" {
		return sidecar{}, fmt.Errorf("incomplete sidecar metadata")
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return sidecar{}, err
	}
	if !strings.EqualFold(sum, sc.SHA256) {
		return sidecar{}, fmt.Errorf("SHA-256 mismatch")
	}
	return sc, nil
}

func inspectEvidence(root string) ([]verifiedEvidence, []string) {
	evidenceRoot := filepath.Join(root, "evidence")
	var evidence []verifiedEvidence
	var issues []string
	err := filepath.WalkDir(evidenceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			issues = append(issues, walkErr.Error())
			return nil
		}
		if path == evidenceRoot || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			issues = append(issues, err.Error())
			return nil
		}
		rel = filepath.ToSlash(rel)
		if entry.Type()&os.ModeSymlink != 0 {
			issues = append(issues, rel+": symlinks are not valid evidence")
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".dorocap-") || strings.HasSuffix(entry.Name(), ".lock") {
			issues = append(issues, rel+": incomplete temporary file")
			return nil
		}
		if strings.HasSuffix(path, ".json") {
			if info, err := statIn(strings.TrimSuffix(path, ".json")); err == nil && info.Mode().IsRegular() {
				return nil
			}
			if b, err := readFileIn(path); err == nil {
				var possibleSidecar sidecar
				if json.Unmarshal(b, &possibleSidecar) == nil && possibleSidecar.SHA256 != "" {
					issues = append(issues, rel+": orphaned sidecar")
					return nil
				}
			}
		}
		sc, err := validateEvidenceFile(path)
		if err != nil {
			issues = append(issues, rel+": "+err.Error())
			return nil
		}
		evidence = append(evidence, verifiedEvidence{Path: rel, Sidecar: sc})
		return nil
	})
	if err != nil {
		issues = append(issues, err.Error())
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].Path < evidence[j].Path })
	sort.Strings(issues)
	return evidence, issues
}

func cmdVerify(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: dorocap verify")
	}
	root, err := findRoot()
	if err != nil {
		return err
	}
	evidence, issues := inspectEvidence(root)
	verifiedPaths := make(map[string]bool, len(evidence))
	for _, item := range evidence {
		verifiedPaths[item.Path] = true
	}
	names, findingErr := findingFiles(filepath.Join(root, "findings"))
	if findingErr != nil {
		issues = append(issues, findingErr.Error())
	}
	for _, name := range names {
		f, parseErr := parseFinding(filepath.Join(root, "findings", name))
		if parseErr != nil {
			issues = append(issues, parseErr.Error())
			continue
		}
		seen := map[string]bool{}
		allLinks := append(append([]string{}, f.Evidence...), f.RetestEvidence...)
		for _, linked := range allLinks {
			if seen[linked] {
				issues = append(issues, fmt.Sprintf("finding %s: duplicate evidence link %s", f.ID, linked))
			}
			seen[linked] = true
			if !verifiedPaths[linked] {
				issues = append(issues, fmt.Sprintf("finding %s: broken or unverified evidence link %s", f.ID, linked))
			}
		}
	}
	sort.Strings(issues)
	if len(issues) > 0 {
		for _, issue := range issues {
			printError(os.Stderr, "%s", issue)
		}
		return fmt.Errorf("verification failed with %d issue(s)", len(issues))
	}
	printOK("verified %d evidence file(s)", len(evidence))
	return nil
}
