package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
)

// Uses SHA to validate the integrity of a file. The hash needs to be provided in lower-case hexadecimal. Only returns
// true when the file was successfully hashed and the hashes match.
func hashFile(path string, sha string) (bool, error) {
	file, err := openFile(path)
	if err != nil {
		return false, errors.Join(errors.New("failed to hash file "+path), err)
	}
	defer func() {
		_ = file.Close()
	}()

	var digest hash.Hash
	hashSize := len(sha)
	switch hashSize {
	case 40:
		{
			digest = sha1.New()
		}
	case 64:
		{
			digest = sha256.New()
		}
	default:
		{
			return false, errors.New(fmt.Sprintf("Unknown hash size %d", hashSize))
		}
	}
	_, err = io.Copy(digest, file)
	if err != nil {
		return false, errors.Join(errors.New("failed to hash file "+path), err)
	}
	calculated := hex.EncodeToString(digest.Sum(nil))
	return calculated == sha, nil
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

func readJson(path string, structure any) error {
	file, err := openFile(path)
	if err != nil {
		return errors.Join(errors.New("failed to open "+path), err)
	}
	defer func() {
		_ = file.Close()
	}()

	buffer, err := io.ReadAll(file)
	if err != nil {
		return errors.Join(errors.New("failed to read "+path), err)
	}

	err = json.Unmarshal(buffer, structure)
	if err != nil {
		return errors.Join(errors.New("failed to parse "+path), err)
	}

	return nil
}

func writeJson(path string, structure any) error {
	data, err := json.Marshal(structure)
	if err != nil {
		return errors.Join(errors.New("failed to serialize JSON for "+path), err)
	}

	file, err := createFile(path)
	if err != nil {
		return errors.Join(errors.New("failed to open file "+path), err)
	}
	defer func() {
		_ = file.Close()
	}()

	offset := 0
	remaining := len(data)

	for remaining > 0 {
		transferred, err := file.Write(data[offset:remaining])
		if err != nil {
			return errors.Join(errors.New("failed to write file "+path), err)
		}
		offset += transferred
		remaining -= transferred
	}

	return nil
}
