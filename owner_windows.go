//go:build windows

package main

import "fmt"

func statPathUID(path string) (int, error) {
	return 0, fmt.Errorf("owner checks are not used on Windows (%s)", path)
}
