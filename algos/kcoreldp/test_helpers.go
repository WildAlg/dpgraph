package kcoreldp

import (
	"fmt"
	"os"
)

// readFile and fmtSscanf are tiny shims used only by tests, kept in a
// non-_test.go file so the test file can stay focused on assertions.

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func fmtSscanf(s string, v *float64) (int, error) {
	return fmt.Sscanf(s, "%f", v)
}
