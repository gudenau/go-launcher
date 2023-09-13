package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// Uses SHA-1 to validate the integrity of a file. The hash needs to be provided in lower-case hexadecimal. Only returns
// true when the file was successfully hashed and the hashes match.
func hashFile(path string, hash string) (bool, error) {
	file, err := openFile(path)
	if err != nil {
		return false, errors.Join(errors.New("failed to hash file "+path), err)
	}
	digest := sha1.New()
	_, err = io.Copy(digest, file)
	if err != nil {
		return false, errors.Join(errors.New("failed to hash file "+path), err)
	}
	calculated := hex.EncodeToString(digest.Sum(nil))
	return calculated == hash, nil
}

// Hashes a file (if it exists) using hashFile and attempts to delete it if the hashes do not match. The hash needs to
// be provided in lower-case hexadecimal. Only returns true when the file was successfully hashed and the hashes match.
func validateHash(path string, hash string) (bool, error) {
	if fileExists(path) {
		result, err := hashFile(path, hash)
		if err != nil {
			return false, errors.Join(errors.New(fmt.Sprintf("could not validate hash of %s", path)), err)
		}
		if !result {
			err = os.Remove(path)
			if err != nil {
				return false, errors.Join(errors.New(fmt.Sprintf("could not delete corrupted file %s", path)), err)
			}
		}
		return result, nil
	}
	return false, nil
}
