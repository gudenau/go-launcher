package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
)

type AdoptiumPackage struct {
	Checksum      string `json:"checksum"`
	ChecksumLink  string `json:"checksum_link"`
	DownloadCount uint64 `json:"download_count"`
	Link          string `json:"link"`
	MetadataLink  string `json:"metadata_link"`
	Name          string `json:"name"`
	SignatureLink string `json:"signature_link"`
	Size          uint64 `json:"size"`
}

func (this *AdoptiumPackage) url() string {
	return this.Link
}

func (this *AdoptiumPackage) hash() *string {
	return &this.Checksum
}

type AdoptiumBinary struct {
	Architecture  string          `json:"architecture"`
	DownloadCount uint64          `json:"download_count"`
	HeapSize      string          `json:"heap_size"`
	ImageType     string          `json:"image_type"`
	JvmImpl       string          `json:"jvm_impl"`
	Os            string          `json:"os"`
	Package       AdoptiumPackage `json:"package"`
	Project       string          `json:"project"`
	ScmRef        string          `json:"scm_ref"`
	UpdatedAt     string          `json:"updated_at"`
}

type AdoptiumFile struct {
	Link string `json:"link"`
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

type AdoptiumVersion struct {
	Build          uint32 `json:"build"`
	Major          uint32 `json:"major"`
	Minor          uint32 `json:"minor"`
	OpenjdkVersion string `json:"openjdk_version"`
	Patch          uint32 `json:"patch"`
	Security       uint32 `json:"security"`
	Semver         string `json:"semver"`
}

type AdoptiumRelease struct {
	Binaries      []AdoptiumBinary `json:"binaries"`
	DownloadCount uint64           `json:"download_count"`
	Id            string           `json:"id"`
	ReleaseLink   string           `json:"release_link"`
	ReleaseName   string           `json:"release_name"`
	ReleaseNotes  AdoptiumFile     `json:"release_notes"`
	ReleaseType   string           `json:"release_type"`
	Source        AdoptiumFile     `json:"source"`
	Timestamp     string           `json:"timestamp"`
	UpdatedAt     string           `json:"updated_at"`
	Vendor        string           `json:"vendor"`
	VersionData   AdoptiumVersion  `json:"version_data"`
}

func extractTar(destination string, source string) error {
	file, err := openFile(source)
	if err != nil {
		return errors.Join(errors.New("failed to open "+source), err)
	}
	defer func() {
		_ = file.Close()
	}()

	stream, err := gzip.NewReader(file)
	if err != nil {
		return errors.Join(errors.New("failed to decompress "+source), err)
	}

	reader := tar.NewReader(stream)
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return errors.Join(errors.New("failed to extract "+source), err)
			}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			{
				err = createParents(destination + header.Name)
				if err != nil {
					return errors.Join(errors.New("failed to extract"+source), err)
				}
			}

		case tar.TypeReg:
			{
				err = func() error {
					file, err := createFileWithPerms(destination+header.Name, os.FileMode(header.Mode))
					if err != nil {
						return err
					}
					defer func() {
						_ = file.Close()
					}()
					_, err = io.Copy(file, reader)
					return err
				}()
				if err != nil {
					return errors.Join(errors.New("failed to extract "+source), err)
				}
			}

		case tar.TypeSymlink:
			{
				err = createLink(destination+header.Name, header.Linkname)
				if err != nil {
					return errors.Join(errors.New("failed to extract "+source), err)
				}
			}

		default:
			{
				return errors.New("don't know how to handle " + header.Name + " in " + source)
			}
		}
	}

	return nil
}

func extractZip(destination string, source string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return errors.Join(errors.New("failed to open "+source), err)
	}
	defer func() {
		_ = reader.Close()
	}()

	for i := range reader.File {
		file := reader.File[i]

		if file.FileInfo().IsDir() {
			err = createParents(destination + file.Name)
			if err != nil {
				return errors.Join(errors.New("failed to extract"+source), err)
			}
		} else {
			err = func() error {
				out, err := createFileWithPerms(destination+file.Name, file.Mode())
				if err != nil {
					return err
				}
				defer func() {
					_ = out.Close()
				}()

				in, err := file.Open()
				if err != nil {
					return err
				}
				defer func() {
					_ = in.Close()
				}()

				_, err = io.Copy(out, in)
				return err
			}()
			if err != nil {
				return errors.Join(errors.New("failed to extract "+source), err)
			}
		}
	}

	return nil
}

func findJdk(path string) (string, error) {
	dirs, err := os.ReadDir(path)
	if err == nil {
		for i := range dirs {
			dir := dirs[i]
			if dir.IsDir() {
				return path + dir.Name(), nil
			}
		}
	}
	return "", errors.Join(errors.New("failed to find JVM dir"), err)
}

func downloadJdk(base string, version uint32) (string, error) {
	// https://api.adoptium.net/v3/assets/feature_releases/17/ga?architecture=x64&heap_size=normal&image_type=jre&jvm_impl=hotspot&os=linux&page=0&page_size=10&project=jdk&sort_method=DEFAULT&sort_order=DESC&vendor=eclipse
	var releases []AdoptiumRelease
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		{
			arch = "x64"
		}
	case "386":
		{
			arch = "x32"
		}
	default:
		{
			arch = runtime.GOARCH
		}
	}

	err := downloadJsonRaw(fmt.Sprintf(
		"https://api.adoptium.net/v3/assets/feature_releases/%d/ga?architecture=%s&heap_size=normal&image_type=jre&jvm_impl=hotspot&os=%s&page=0&page_size=10&project=jdk&sort_method=DEFAULT&sort_order=DESC&vendor=eclipse",
		version,
		arch,
		runtime.GOOS,
	), nil, &releases)
	if err != nil {
		return "", err
	}

	sort.Slice(releases, func(indexA int, indexB int) bool {
		a := releases[indexA].VersionData
		b := releases[indexB].VersionData

		if a.Major != b.Major {
			return a.Major < b.Major
		}

		if a.Minor != b.Minor {
			return a.Minor < b.Minor
		}

		if a.Security != b.Security {
			return a.Security < b.Security
		}

		return a.Build < b.Build
	})

	latest := releases[len(releases)-1]
	if len(latest.Binaries) != 1 {
		return "", errors.New("an incorrect amount of binaries was returned")
	}

	binary := latest.Binaries[0].Package

	// This should be
	// extension := runtime.GOOS == "windows" ? "zip" : "tar.gz"
	var extension string
	if runtime.GOOS == "windows" {
		extension = "zip"
	} else {
		extension = "tar.gz"
	}

	path := base + "/library/net/java/jdk/" + latest.VersionData.Semver + "/"
	archive := path + "jdk-" + latest.VersionData.Semver + "." + extension
	valid, err := validateHash(archive, binary.Checksum)
	if err != nil {
		return "", errors.Join(errors.New("failed to hash JVM package"), err)
	}
	if valid {
		path, err = findJdk(path)
		return path, err
	}

	err = downloadFile(archive, &binary)
	if err != nil {
		return "", errors.Join(errors.New("could not download JVM"), err)
	}

	if runtime.GOOS == "windows" {
		err = extractZip(path, archive)
	} else {
		err = extractTar(path, archive)
	}
	if err != nil {
		return "", errors.Join(errors.New("failed to extract jvm"), err)
	}

	path, err = findJdk(path)
	return path, err
}
