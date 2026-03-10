package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "modbender"
	repoName  = "hprof-analyzer"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Upgrade checks for a newer release and replaces the current binary.
// Returns (newVersion, error). If already up to date, newVersion is empty.
func Upgrade(currentVersion string) (string, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return "", fmt.Errorf("checking for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if latest == current || current == "dev" && latest == "" {
		return "", nil
	}

	archiveAsset, checksumAsset, err := findAssets(release)
	if err != nil {
		return "", err
	}

	archiveData, err := download(archiveAsset.BrowserDownloadURL)
	if err != nil {
		return "", fmt.Errorf("downloading release: %w", err)
	}

	if checksumAsset != nil {
		if err := verifyChecksum(archiveData, archiveAsset.Name, checksumAsset.BrowserDownloadURL); err != nil {
			return "", err
		}
	}

	binaryData, err := extractBinary(archiveData, archiveAsset.Name)
	if err != nil {
		return "", fmt.Errorf("extracting binary: %w", err)
	}

	if err := replaceBinary(binaryData); err != nil {
		return "", fmt.Errorf("replacing binary: %w", err)
	}

	return release.TagName, nil
}

func fetchLatestRelease() (*ghRelease, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s/%s", repoOwner, repoName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func findAssets(release *ghRelease) (archive ghAsset, checksum *ghAsset, err error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// goreleaser naming convention
	var ext string
	if goos == "windows" {
		ext = ".zip"
	} else {
		ext = ".tar.gz"
	}

	archiveName := fmt.Sprintf("hprof-analyzer_%s_%s%s", goos, goarch, ext)

	for _, a := range release.Assets {
		if a.Name == archiveName {
			archive = a
		}
		if a.Name == "checksums.txt" {
			cs := a
			checksum = &cs
		}
	}

	if archive.Name == "" {
		return archive, nil, fmt.Errorf("no release asset found for %s/%s (looking for %s)", goos, goarch, archiveName)
	}
	return archive, checksum, nil
}

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func verifyChecksum(data []byte, filename, checksumURL string) error {
	checksumData, err := download(checksumURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	expectedHash := ""
	for _, line := range strings.Split(string(checksumData), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum for %s not found in checksums.txt", filename)
	}

	actualHash := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualHash[:])

	if actualHex != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHex)
	}
	return nil
}

func extractBinary(archiveData []byte, archiveName string) ([]byte, error) {
	binaryName := "hprof-analyzer"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	if strings.HasSuffix(archiveName, ".tar.gz") {
		return extractFromTarGz(archiveData, binaryName)
	}
	return extractFromZip(archiveData, binaryName)
}

func extractFromTarGz(data []byte, binaryName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == binaryName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractFromZip(data []byte, binaryName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary %s not found in archive", binaryName)
}

func replaceBinary(newBinary []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	// Write to temp file next to existing binary, then atomic rename
	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, "hprof-analyzer-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(newBinary); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Preserve original permissions
	info, err := os.Stat(execPath)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
