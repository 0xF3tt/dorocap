//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func statPathUID(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("cannot determine owner of %s", path)
	}
	return int(stat.Uid), nil
}
