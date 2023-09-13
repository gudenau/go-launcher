//go:build !windows

package main

import (
	"os"
	"os/exec"
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

// A wrapper for os.Create that creates a file
func createFile(name string) (*os.File, error) {
	return os.Create(name)
}

// A wrapper for os.Create that creates a file with specific permissions
func createFileWithPerms(name string, perms os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perms)
}

// A wrapper for os.MkdirAll that creates a bunch of directories
func createParents(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// A wrapper for os.SymLink that creates a symbolic link
func createLink(path string, target string) error {
	return os.Symlink(target, path)
}

// A wrapper for exec.Command that sets up a new process structure
func execute(executable string, args ...string) *exec.Cmd {
	return exec.Command(executable, args...)
}
