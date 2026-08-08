package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func cmdNote(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: dorocap note <type> <text...>")
	}
	typ := args[0]
	if err := validateCategory("note type", typ, true); err != nil {
		return err
	}
	text := strings.Join(args[1:], " ")
	if strings.ContainsAny(text, "\r\n\x00") {
		return fmt.Errorf("note must be a single line")
	}

	root, err := findRoot()
	if err != nil {
		return err
	}

	dir := filepath.Join(root, "notes")
	if err := mkdirAll(dir, 0o750); err != nil {
		return err
	}

	path := filepath.Join(dir, typ+".md")
	line := fmt.Sprintf("- %s %s\n", time.Now().UTC().Format(time.RFC3339), text)

	err = withFileLock(path, func() error {
		old, readErr := readFileIn(path)
		if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}
		return writeFileAtomic(path, append(old, []byte(line)...), 0o600)
	})
	if err != nil {
		return err
	}

	printOK("saved %s", path)
	return nil
}
