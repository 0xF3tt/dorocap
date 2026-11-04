package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func cmdFinalize(args []string) error {
	acceptEdits := false
	switch len(args) {
	case 0:
	case 1:
		if args[0] != "--accept-edits" {
			return fmt.Errorf("usage: dorocap finalize [--accept-edits]")
		}
		acceptEdits = true
	default:
		return fmt.Errorf("usage: dorocap finalize [--accept-edits]")
	}
	root, err := findRoot()
	if err != nil {
		return err
	}
	if err := cmdVerify(nil); err != nil {
		return fmt.Errorf("cannot finalize an unverified engagement: %w", err)
	}
	draft := draftReportPath(root)
	b, err := readFileIn(draft)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("draft report not found; run `dorocap export` first")
		}
		return err
	}
	if acceptEdits {
		if err := writeDraftChecksum(root, b); err != nil {
			return err
		}
	} else if err := draftChecksumMatches(root, b); err != nil {
		return err
	}
	final := filepath.Join(root, "report", "final", "report.md")
	if err := writeFileAtomic(final, b, 0o600); err != nil {
		return err
	}
	printOK("final report saved %s", final)
	return nil
}
