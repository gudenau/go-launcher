//go:build !windows

package main

import (
	"os"
)

// A wrapper for os.Stat that checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// A wrapper for os.Stat that opens a file
func openFile(name string) (*os.File, error) {
	return os.Open(name)
}

// A wrapper for os.Stat that creates a file
func createFile(name string) (*os.File, error) {
	return os.Create(name)
}

// A wrapper for os.MkdirAll that creates a bunch of directories
func createParents(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}
