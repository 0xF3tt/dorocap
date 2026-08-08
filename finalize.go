package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func cmdFinalize(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: dorocap finalize")
	}
	root, err := findRoot()
	if err != nil {
		return err
	}
	if err := cmdVerify(nil); err != nil {
		return fmt.Errorf("cannot finalize an unverified engagement: %w", err)
	}
	draft := filepath.Join(root, "report", "draft", "report.md")
	b, err := readFileIn(draft)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("draft report not found; run `dorocap export` first")
		}
		return err
	}
	final := filepath.Join(root, "report", "final", "report.md")
	if err := writeFileAtomic(final, b, 0o600); err != nil {
		return err
	}
	printOK("final report saved %s", final)
	return nil
}
