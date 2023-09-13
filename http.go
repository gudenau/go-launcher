package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Downloadable interface {
	url() string
	hash() *string
}

// Downloads a file and optionally validates its hash. If the parent of the path does not exist it will be created. If
// the hash does not match the file will be deleted.
func downloadFile(path string, downloadable Downloadable) error {
	return downloadFileRaw(path, downloadable.url(), downloadable.hash())
}

// Downloads a file and optionally validates its hash. If the parent of the path does not exist it will be created. If
// the hash does not match the file will be deleted.
func downloadFileRaw(path string, url string, hash *string) error {
	var err error
	if hash != nil {
		valid, err := validateHash(path, *hash)
		if err != nil {
			return errors.Join(errors.New("failed to validate "+path), err)
		}
		if valid {
			return nil
		}
	}

	err = createParents(filepath.Dir(path))
	if err != nil {
		return errors.Join(errors.New("failed to create parents of "+path), err)
	}

	file, err := createFile(path)
	if err != nil {
		return errors.Join(errors.New("failed to create file "+path), err)
	}

	response, err := http.Get(url)
	if err != nil {
		return errors.Join(errors.New("failed to download "+url), err)
	}
	if response.StatusCode/100 != 2 {
		return errors.New("failed to download " + url + ": " + response.Status)
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		_ = os.Remove(path) // Don't care
		return errors.Join(errors.New("failed to download "+url), err)
	}

	_ = file.Close()

	if hash != nil {
		valid, err := validateHash(path, *hash)
		if err != nil {
			return errors.Join(errors.New("could not validate hash of "+path), err)
		}
		if !valid {
			return errors.New("download " + path + " failed to download")
		}
	}
	return nil
}

// Downloads a JSON file, optionally validates its hash and then deserializes it. If the hashes don't match the
// structure is not touched.
func downloadJson(downloadable Downloadable, structure any) error {
	return downloadJsonRaw(downloadable.url(), downloadable.hash(), structure)
}

// Downloads a JSON file, optionally validates its hash and then deserializes it. If the hashes don't match the
// structure is not touched.
func downloadJsonRaw(url string, hash *string, structure any) error {
	response, err := http.Get(url)
	if err != nil {
		return errors.Join(errors.New("failed to download "+url), err)
	}
	if response.StatusCode/100 != 2 {
		return errors.New("failed to download " + url + ": " + response.Status)
	}

	buffer, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.Join(errors.New("failed to copy "+url+" into a buffer"), err)
	}

	if hash != nil {
		digest := sha1.New()
		digest.Write(buffer)
		calculated := hex.EncodeToString(digest.Sum(nil))
		if calculated != *hash {
			return errors.New("failed to verify hash of " + url + ", got " + calculated + " and expected " + *hash)
		}
	}

	err = json.Unmarshal(buffer, structure)
	if err != nil {
		return errors.Join(errors.New("Failed to parse JSON of "+url), err)
	}

	return nil
}
