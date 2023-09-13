//go:build windows

package main

import (
	"os"
	"strings"
)

// Switches paths from the sane Unix format to the Insane DOS/NT format. (Replaces all forward slashes with backslashes)
// Returns the modified string.
func insanifyPath(path string) string {
	return strings.ReplaceAll(path, "/", "\\")
}

// A wrapper for os.Stat that checks if a file exists, automatically converts paths from Unix to DOS/NT
func fileExists(path string) bool {
	_, err := os.Stat(insanifyPath(path))
	return err != nil
}

// A wrapper for os.Stat that opens a file, automatically converts paths from Unix to DOS/NT
func openFile(name string) (*os.File, error) {
	return os.Open(insanifyPath(name))
}

// A wrapper for os.Stat that creates a file, automatically converts paths from Unix to DOS/NT
func createFile(name string) (*os.File, error) {
	return os.Create(insanifyPath(name))
}

// A wrapper for os.MkdirAll that creates a bunch of directories, automatically converts paths from Unix to DOS/NT
func createParents(path string) error {
	return os.MkdirAll(insanifyPath(path), os.ModePerm)
}
