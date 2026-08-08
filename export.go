package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func cmdExport(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: dorocap export")
	}
	root, err := findRoot()
	if err != nil {
		return err
	}
	scope, err := loadScope(root)
	if err != nil {
		return err
	}
	names, err := findingFiles(filepath.Join(root, "findings"))
	if err != nil {
		return err
	}
	findings := make([]finding, 0, len(names))
	counts := map[string]int{}
	for _, name := range names {
		f, err := parseFinding(filepath.Join(root, "findings", name))
		if err != nil {
			return err
		}
		findings = append(findings, f)
		counts[f.Severity]++
	}
	evidence, integrityIssues := inspectEvidence(root)
	byPath := make(map[string]verifiedEvidence, len(evidence))
	for _, item := range evidence {
		byPath[item.Path] = item
	}
	linked := map[string]bool{}
	var brokenLinks []string
	generated := time.Now().UTC()

	var b strings.Builder
	assessmentType := valueOr(scope.AssessmentType, "Penetration Test")
	classification := valueOr(scope.Classification, "Confidential")
	version := valueOr(scope.ReportVersion, "0.1")
	preparedBy := valueOr(scope.PreparedBy, "@0xF3tt")
	fmt.Fprintf(&b, "# %s Report\n\n", assessmentType)
	fmt.Fprintf(&b, "## %s\n\n", scope.Engagement)
	fmt.Fprintf(&b, "> **Classification:** %s\n\n", classification)
	fmt.Fprintf(&b, "- **Client:** %s\n", displayValue(scope.Client))
	fmt.Fprintf(&b, "- **Prepared by:** %s\n", preparedBy)
	fmt.Fprintf(&b, "- **Report version:** %s\n", version)
	fmt.Fprintf(&b, "- **Report date:** %s\n", generated.Format("2006-01-02"))
	fmt.Fprintf(&b, "- **Testing period:** %s to %s\n", displayValue(scope.StartDate), displayValue(scope.EndDate))

	b.WriteString("\n## Document Control\n\n")
	b.WriteString("| Version | Date | Author | Changes |\n|---|---|---|---|\n")
	if len(scope.RevisionHistory) == 0 {
		fmt.Fprintf(&b, "| %s | %s | %s | Generated draft |\n", markdownCell(version), generated.Format("2006-01-02"), markdownCell(preparedBy))
	} else {
		for _, revision := range scope.RevisionHistory {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", markdownCell(revision.Version), markdownCell(revision.Date), markdownCell(revision.Author), markdownCell(revision.Changes))
		}
	}

	b.WriteString("\n## Table of Contents\n\n")
	b.WriteString("1. [Executive Summary](#executive-summary)\n2. [Risk Summary](#risk-summary)\n3. [Engagement and Scope](#engagement-and-scope)\n4. [Methodology](#methodology)\n5. [Limitations](#limitations)\n6. [Findings](#findings)\n7. [Remediation Roadmap](#remediation-roadmap)\n8. [Notes](#notes)\n9. [Evidence Integrity](#evidence-integrity)\n")

	b.WriteString("\n## Executive Summary\n\n")
	if strings.TrimSpace(scope.ExecutiveSummary) != "" {
		fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(scope.ExecutiveSummary))
	} else {
		fmt.Fprintf(&b, "The assessment identified %d finding(s) across the defined scope. Prioritize the key risks and remediation roadmap below, then validate fixes through retesting.\n\n", len(findings))
	}
	fmt.Fprintf(&b, "**Overall security posture:** %s\n\n", valueOr(scope.OverallPosture, derivedPosture(counts)))
	b.WriteString("### Key Risks and Attack Paths\n\n")
	keyRisks := findingsBySeverity(findings, "crit", "high")
	if len(keyRisks) == 0 {
		b.WriteString("No critical or high-severity findings were recorded.\n")
	} else {
		for _, f := range keyRisks {
			fmt.Fprintf(&b, "- [%s — %s](#finding-%s)%s\n", f.ID, markdownInline(f.Title), f.ID, assetSuffix(f.Asset))
		}
	}
	b.WriteString("\n### Severity Distribution\n\n")
	b.WriteString("| Critical | High | Medium | Low | Informational |\n|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| %d | %d | %d | %d | %d |\n", counts["crit"], counts["high"], counts["med"], counts["low"], counts["info"])

	b.WriteString("\n## Risk Summary\n\n")
	b.WriteString("| ID | Finding | Severity | Affected asset | CVSS | Status |\n|---|---|---|---|---:|---|\n")
	if len(findings) == 0 {
		b.WriteString("| — | No findings recorded | — | — | — | — |\n")
	}
	for _, f := range findings {
		fmt.Fprintf(&b, "| [%s](#finding-%s) | %s | %s | %s | %s | %s |\n", f.ID, f.ID, markdownCell(f.Title), strings.ToUpper(f.Severity), markdownCell(valueOr(f.Asset, "—")), markdownCell(valueOr(f.CVSS, "—")), markdownCell(f.Status))
	}

	b.WriteString("\n## Engagement and Scope\n\n")
	fmt.Fprintf(&b, "- **Assessment type:** %s\n", assessmentType)
	fmt.Fprintf(&b, "- **Testing mode:** %s\n", displayValue(scope.TestingMode))
	fmt.Fprintf(&b, "- **Client:** %s\n- **Start date:** %s\n- **End date:** %s\n", displayValue(scope.Client), displayValue(scope.StartDate), displayValue(scope.EndDate))
	writeStringList(&b, "In scope", scope.InScope)
	writeStringList(&b, "Out of scope", scope.OutOfScope)
	writeStringList(&b, "Contacts", scope.Contacts)
	writeStringList(&b, "Restrictions", scope.Restrictions)
	writeStringList(&b, "Emergency stop conditions", scope.EmergencyStop)

	b.WriteString("\n## Methodology\n\n")
	methodology := cleanStrings(scope.Methodology)
	if len(methodology) == 0 {
		methodology = []string{"Reconnaissance", "Enumeration", "Vulnerability analysis", "Exploitation", "Post-exploitation", "Evidence verification", "Reporting"}
	}
	for i, phase := range methodology {
		fmt.Fprintf(&b, "%d. %s\n", i+1, phase)
	}

	b.WriteString("\n## Limitations\n\n")
	limitations := cleanStrings(scope.Limitations)
	if len(limitations) == 0 {
		b.WriteString("No assessment limitations were documented.\n")
	} else {
		for _, limitation := range limitations {
			fmt.Fprintf(&b, "- %s\n", limitation)
		}
	}

	b.WriteString("\n## Findings\n\n")
	if len(findings) == 0 {
		b.WriteString("_No findings recorded yet._\n")
	}
	figure := 0
	for _, f := range findings {
		fmt.Fprintf(&b, "<a id=\"finding-%s\"></a>\n\n", f.ID)
		fmt.Fprintf(&b, "### [%s] %s\n\n", f.ID, f.Title)
		b.WriteString("| Field | Value |\n|---|---|\n")
		fmt.Fprintf(&b, "| Severity | **%s** |\n", strings.ToUpper(f.Severity))
		fmt.Fprintf(&b, "| Status | %s |\n", markdownCell(f.Status))
		fmt.Fprintf(&b, "| Affected asset | %s |\n", markdownCell(valueOr(f.Asset, "_(not specified)_")))
		fmt.Fprintf(&b, "| CVSS | %s |\n", markdownCell(valueOr(f.CVSS, "_(not specified)_")))
		fmt.Fprintf(&b, "| CWE | %s |\n", markdownCell(valueOr(f.CWE, "_(not specified)_")))
		fmt.Fprintf(&b, "| OWASP | %s |\n", markdownCell(valueOr(f.OWASP, "_(not specified)_")))
		fmt.Fprintf(&b, "| Created | %s |\n", markdownCell(valueOr(f.Created, "_(not specified)_")))
		fmt.Fprintf(&b, "| Retest status | %s |\n", markdownCell(f.RetestStatus))
		fmt.Fprintf(&b, "| Retested | %s |\n\n", markdownCell(valueOr(f.Retested, "_(not tested)_")))
		if f.Body != "" {
			fmt.Fprintf(&b, "%s\n\n", nestedFindingHeadings(f.Body))
		}
		brokenLinks = append(brokenLinks, writeEvidenceSection(&b, "Evidence", f.ID, f.Evidence, byPath, linked, &figure)...)
		brokenLinks = append(brokenLinks, writeEvidenceSection(&b, "Retest Evidence", f.ID, f.RetestEvidence, byPath, linked, &figure)...)
	}

	b.WriteString("\n## Remediation Roadmap\n\n")
	writeRemediationGroup(&b, "Immediate — Critical and High", findingsBySeverity(findings, "crit", "high"))
	writeRemediationGroup(&b, "Short-term — Medium", findingsBySeverity(findings, "med"))
	writeRemediationGroup(&b, "Long-term — Low and Informational", findingsBySeverity(findings, "low", "info"))

	writeNotes(&b, root)
	b.WriteString("\n## Evidence Integrity\n\n")
	fmt.Fprintf(&b, "Verified evidence files: %d\n\n", len(evidence))
	var unlinked []string
	for _, item := range evidence {
		if !linked[item.Path] {
			unlinked = append(unlinked, item.Path)
		}
	}
	if len(integrityIssues) == 0 && len(brokenLinks) == 0 && len(unlinked) == 0 {
		b.WriteString("No integrity, link, or orphan warnings.\n")
	} else {
		for _, issue := range integrityIssues {
			fmt.Fprintf(&b, "- **Integrity:** %s\n", issue)
		}
		for _, link := range brokenLinks {
			fmt.Fprintf(&b, "- **Broken evidence link:** %s\n", link)
		}
		for _, path := range unlinked {
			fmt.Fprintf(&b, "- **Unlinked evidence:** `%s`\n", path)
		}
	}

	dir := filepath.Join(root, "report", "draft")
	path := filepath.Join(dir, "report.md")
	if err := writeFileAtomic(path, []byte(b.String()), 0o600); err != nil {
		return err
	}
	printOK("report saved %s", path)
	return nil
}

func isMarkdownImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func markdownAlt(text string) string {
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]").Replace(text)
}

func markdownInline(text string) string {
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`").Replace(text)
}

func markdownCell(text string) string {
	return strings.NewReplacer("|", "\\|", "\r", " ", "\n", " ").Replace(text)
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func cleanStrings(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			clean = append(clean, value)
		}
	}
	return clean
}

func derivedPosture(counts map[string]int) string {
	switch {
	case counts["crit"] > 0:
		return "Critical risk — immediate remediation is required"
	case counts["high"] > 0:
		return "High risk — prioritize remediation of significant attack paths"
	case counts["med"] > 0:
		return "Moderate risk — address material weaknesses in the short term"
	case counts["low"] > 0:
		return "Low risk — complete hardening improvements through normal planning"
	default:
		return "No material risk was identified from the recorded findings"
	}
}

func findingsBySeverity(findings []finding, severities ...string) []finding {
	wanted := make(map[string]bool, len(severities))
	for _, severity := range severities {
		wanted[severity] = true
	}
	var selected []finding
	for _, f := range findings {
		if wanted[f.Severity] {
			selected = append(selected, f)
		}
	}
	return selected
}

func assetSuffix(asset string) string {
	if strings.TrimSpace(asset) == "" {
		return ""
	}
	return " — `" + strings.ReplaceAll(asset, "`", "\\`") + "`"
}

func nestedFindingHeadings(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			lines[i] = "### " + strings.TrimPrefix(line, "## ")
		}
	}
	return strings.Join(lines, "\n")
}

func writeEvidenceSection(b *strings.Builder, heading, findingID string, paths []string, byPath map[string]verifiedEvidence, linked map[string]bool, figure *int) []string {
	if len(paths) == 0 {
		return nil
	}
	fmt.Fprintf(b, "### %s\n\n", heading)
	var broken []string
	for _, path := range paths {
		item, ok := byPath[path]
		if !ok {
			fmt.Fprintf(b, "- `%s` — **BROKEN OR UNVERIFIED**\n", path)
			broken = append(broken, findingID+": "+path)
			continue
		}
		linked[path] = true
		(*figure)++
		caption := valueOr(item.Sidecar.Note, filepath.Base(path))
		href := "../../" + (&url.URL{Path: filepath.ToSlash(path)}).EscapedPath()
		if isMarkdownImage(path) {
			fmt.Fprintf(b, "**Figure %d — %s**\n\n", *figure, markdownInline(caption))
			fmt.Fprintf(b, "![%s](%s)\n\n", markdownAlt(caption), href)
		} else {
			fmt.Fprintf(b, "**Evidence %d — [%s](%s)**\n\n", *figure, markdownInline(caption), href)
		}
		fmt.Fprintf(b, "- **Path:** `%s`\n", path)
		fmt.Fprintf(b, "- **Category:** %s\n", item.Sidecar.Type)
		fmt.Fprintf(b, "- **Captured:** %s\n", item.Sidecar.Timestamp)
		fmt.Fprintf(b, "- **SHA-256:** `%s`\n", item.Sidecar.SHA256)
		if item.Sidecar.Note != "" {
			fmt.Fprintf(b, "- **Operator note:** %s\n", item.Sidecar.Note)
		}
		b.WriteString("\n")
	}
	return broken
}

func writeRemediationGroup(b *strings.Builder, heading string, findings []finding) {
	fmt.Fprintf(b, "### %s\n\n", heading)
	if len(findings) == 0 {
		b.WriteString("No findings in this priority group.\n\n")
		return
	}
	for _, f := range findings {
		fmt.Fprintf(b, "- [%s — %s](#finding-%s)%s\n", f.ID, markdownInline(f.Title), f.ID, assetSuffix(f.Asset))
	}
	b.WriteString("\n")
}

func displayValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "_(not specified)_"
	}
	return value
}

func writeStringList(b *strings.Builder, label string, values []string) {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			clean = append(clean, value)
		}
	}
	if len(clean) == 0 {
		fmt.Fprintf(b, "- %s: _(not specified)_\n", label)
		return
	}
	fmt.Fprintf(b, "- %s:\n", label)
	for _, value := range clean {
		fmt.Fprintf(b, "  - %s\n", value)
	}
}

func writeNotes(b *strings.Builder, root string) {
	entries, err := os.ReadDir(filepath.Join(root, "notes"))
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(b, "\n## Notes\n\n_Unable to read notes: %s_\n", err)
		return
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	b.WriteString("\n## Notes\n\n")
	if len(names) == 0 {
		b.WriteString("_No notes recorded._\n")
		return
	}
	for _, name := range names {
		content, err := readFileIn(filepath.Join(root, "notes", name))
		if err != nil {
			continue
		}
		fmt.Fprintf(b, "### %s\n\n%s\n", strings.TrimSuffix(name, ".md"), strings.TrimSpace(string(content)))
	}
}
