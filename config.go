package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func configPath() (string, error) {
	if p := os.Getenv("DOROCAP_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dorocap", "config"), nil
}

func loadConfigRoot() (string, error) {
	p, err := configPath()
	if err != nil {
		return "", err
	}
	b, err := readFileIn(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func saveConfigRoot(dir string) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := mkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	return writeFileAtomic(p, []byte(dir+"\n"), 0o600)
}

func engagementName(root string) string {
	scope, err := loadScope(root)
	if err != nil {
		return ""
	}
	return scope.Engagement
}

func cmdPath(args []string) error {
	if len(args) == 0 {
		root, err := loadConfigRoot()
		if err != nil {
			return err
		}
		if root == "" {
			printWarning("no global engagement path set")
		} else {
			fmt.Println(root)
		}
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: dorocap path [<dir>]")
	}

	dir, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	if !hasEngagementMarker(dir) {
		return fmt.Errorf("%s is not an engagement (no %s found)", dir, markerFile)
	}
	if err := saveConfigRoot(dir); err != nil {
		return err
	}
	printOK("global engagement path set to %s", dir)
	return nil
}

func cmdInfo(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: dorocap info")
	}
	cfg, err := loadConfigRoot()
	if err != nil {
		return err
	}
	if cfg == "" {
		fmt.Println("global path:       (unset)")
	} else {
		fmt.Printf("global path:       %s\n", cfg)
	}

	root, err := findRoot()
	if err != nil {
		fmt.Println("active engagement: none —", err)
		return nil
	}
	source := "global config"
	if local, err := findRootFromCwd(); err == nil && local == root {
		source = "current directory"
	}
	fmt.Printf("active engagement: %s\n", root)
	if name := engagementName(root); name != "" {
		fmt.Printf("name:              %s\n", name)
	}
	fmt.Printf("resolved via:      %s\n", source)
	return nil
}
