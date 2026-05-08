package v1

import "os"

// mkdirAllImpl wraps os.MkdirAll so admin_test.go can avoid importing
// os directly (the file lives next to others that have specific
// import budgets).
func mkdirAllImpl(p string, mode int) error {
	return os.MkdirAll(p, os.FileMode(mode))
}
