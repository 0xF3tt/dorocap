package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func dirRoot(path string) (*os.Root, string, error) {
	r, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, "", err
	}
	return r, filepath.Base(path), nil
}

func readFileIn(path string) ([]byte, error) {
	r, name, err := dirRoot(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return r.ReadFile(name)
}

func openIn(path string) (*os.File, error) {
	r, name, err := dirRoot(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return r.Open(name)
}

func statIn(path string) (os.FileInfo, error) {
	r, name, err := dirRoot(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return r.Stat(name)
}

func removeIn(path string) error {
	r, name, err := dirRoot(path)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	return r.Remove(name)
}

func chmodIn(path string, perm os.FileMode) error {
	r, name, err := dirRoot(path)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	return r.Chmod(name, perm)
}

func renameIn(oldPath, newPath string) error {
	dir := filepath.Dir(oldPath)
	if filepath.Dir(newPath) != dir {
		return fmt.Errorf("renameIn: cross-directory rename %q -> %q", oldPath, newPath)
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	return r.Rename(filepath.Base(oldPath), filepath.Base(newPath))
}

func mkdirAll(dir string, perm os.FileMode) error {
	dir = filepath.Clean(dir)
	var missing []string
	cur := dir
	for {
		r, err := os.OpenRoot(cur)
		if err == nil {
			_ = r.Close()
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append([]string{filepath.Base(cur)}, missing...)
		parent := filepath.Dir(cur)
		if parent == cur {
			return err
		}
		cur = parent
	}
	if len(missing) == 0 {
		return nil
	}
	r, err := os.OpenRoot(cur)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	return r.MkdirAll(filepath.Join(missing...), perm)
}

func copyPathAtomic(src, dst string, perm os.FileMode) error {
	in, err := openIn(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	dir := filepath.Dir(dst)
	if err := mkdirAll(dir, 0o750); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	suffix, err := uniqueSuffix()
	if err != nil {
		return err
	}
	tmp := ".dorocap-copy-" + suffix
	out, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		_ = out.Close()
		if !committed {
			_ = root.Remove(tmp)
		}
	}()
	if err := out.Chmod(perm); err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := root.Rename(tmp, filepath.Base(dst)); err != nil {
		return err
	}
	committed = true
	return nil
}

func uniqueSuffix() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func withFileLock(target string, fn func() error) error {
	dir := filepath.Dir(target)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	lock := filepath.Base(target) + ".lock"
	for attempt := 0; attempt < 200; attempt++ {
		f, err := root.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = f.Close()
			defer func() { _ = root.Remove(lock) }()
			return fn()
		}
		if os.IsPermission(err) {
			_, statErr := root.Stat(lock)
			if statErr == nil || os.IsNotExist(statErr) {
				time.Sleep(25 * time.Millisecond)
				continue
			}
		}
		if !os.IsExist(err) {
			return err
		}
		if info, statErr := root.Stat(lock); statErr == nil && time.Since(info.ModTime()) > 30*time.Second {
			_ = root.Remove(lock)
			continue
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for lock %s", target)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := mkdirAll(dir, 0o750); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	suffix, err := uniqueSuffix()
	if err != nil {
		return err
	}
	tmp := ".dorocap-tmp-" + suffix
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		_ = f.Close()
		if !committed {
			_ = root.Remove(tmp)
		}
	}()
	if err := f.Chmod(perm); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := root.Rename(tmp, filepath.Base(path)); err != nil {
		return err
	}
	committed = true
	return nil
}

func ensureWithin(base, candidate string) error {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return err
	}
	if rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes %s", base)
	}
	return nil
}
