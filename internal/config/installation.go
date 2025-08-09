package config

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"github.com/JSH-Team/JSHunter/internal/utils/logger"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// GitHub release URLs for precompiled binaries
const (
	AnalyzerRepoURL   = "https://github.com/rollinx1/jshunter-analyzer/releases/latest/download"
	PrettifierRepoURL = "https://github.com/rollinx1/jshunter-prettifier/releases/latest/download"
	DechunkerRepoURL  = "https://github.com/rollinx1/jshunter-dechunker/releases/latest/download"
)

func RunInstallationSteps() error {
	logger.Info("Checking for dependencies...")

	if ForceInstallation {
		logger.Info("--force flag detected, removing existing dependencies")
		if err := os.RemoveAll(GetLibsDirectory()); err != nil {
			return fmt.Errorf("failed to remove libs directory: %w", err)
		}
	}

	if err := os.MkdirAll(GetLibsDirectory(), 0755); err != nil {
		return fmt.Errorf("failed to create libs directory: %w", err)
	}

	binariesToUpdate := []string{}
	for _, binaryName := range []string{"analyzer", "prettifier", "dechunker"} {
		updateNeeded, err := needsUpdate(binaryName)
		if err != nil {
			return err
		}
		if updateNeeded {
			binariesToUpdate = append(binariesToUpdate, binaryName)
		}
	}

	if len(binariesToUpdate) == 0 {
		logger.Info("All dependencies are up to date.")
		return nil
	}

	for _, binaryName := range binariesToUpdate {
		if err := downloadAndVerify(binaryName); err != nil {
			return err
		}
	}

	return nil
}

func needsUpdate(binaryName string) (bool, error) {
	repoURL := binaries[binaryName]
	checksums, err := downloadChecksums(repoURL)
	if err != nil {
		return false, fmt.Errorf("failed to download checksums for %s: %w", binaryName, err)
	}

	platformName := getPlatformSpecificName(binaryName)
	expectedChecksum, ok := checksums[platformName]
	if !ok {
		return false, fmt.Errorf("checksum not found for %s", platformName)
	}

	binaryFileName := getBinaryFileName(binaryName)
	localPath := filepath.Join(GetLibsDirectory(), binaryFileName)

	if _, err := os.Stat(localPath); err != nil {
		if os.IsNotExist(err) {
			logger.Info("Dependency %s not found.", binaryName)
			return true, nil
		}
		return false, fmt.Errorf("failed to stat local binary %s: %w", binaryName, err)
	}

	currentChecksum, err := calculateFileSHA256(localPath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum for local binary %s: %w", binaryName, err)
	}

	if currentChecksum != expectedChecksum {
		logger.Info("New version of %s available.", binaryName)
		return true, nil
	}

	return false, nil
}

func downloadAndVerify(binaryName string) error {
	repoURL := binaries[binaryName]
	checksums, err := downloadChecksums(repoURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums for %s: %w", binaryName, err)
	}

	platformName := getPlatformSpecificName(binaryName)
	expectedChecksum, ok := checksums[platformName]
	if !ok {
		return fmt.Errorf("checksum not found for %s", platformName)
	}

	if err := downloadBinary(binaryName); err != nil {
		return err
	}

	binaryFileName := getBinaryFileName(binaryName)
	localPath := filepath.Join(GetLibsDirectory(), binaryFileName)
	newChecksum, err := calculateFileSHA256(localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for downloaded binary %s: %w", binaryName, err)
	}

	if newChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch for downloaded binary %s. expected %s, got %s", binaryName, expectedChecksum, newChecksum)
	}

	return nil
}

func calculateFileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func downloadChecksums(repoURL string) (map[string]string, error) {
	checksumURL := fmt.Sprintf("%s/checksums.txt", repoURL)
	resp, err := http.Get(checksumURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksums.txt: HTTP status %d", resp.StatusCode)
	}

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 {
			fileName := filepath.Base(parts[1])
			checksums[fileName] = parts[0]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading checksums.txt: %w", err)
	}

	return checksums, nil
}

func getPlatformSpecificName(binaryName string) string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	ext := ""
	if os == "windows" {
		ext = ".exe"
	}

	return fmt.Sprintf("%s-%s-%s%s", binaryName, os, arch, ext)
}

func getBinaryFileName(binaryName string) string {
	if runtime.GOOS == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}

// CheckInstallation checks if all required binaries are installed.
func CheckInstallation() bool {
	if ForceInstallation {
		return false
	}
	libsDir := GetLibsDirectory()

	analyzerName := getBinaryFileName("analyzer")
	prettifierName := getBinaryFileName("prettifier")
	dechunkerName := getBinaryFileName("dechunker")

	analyzerPath := filepath.Join(libsDir, analyzerName)
	prettifierPath := filepath.Join(libsDir, prettifierName)
	dechunkerPath := filepath.Join(libsDir, dechunkerName)

	_, err1 := os.Stat(analyzerPath)
	_, err2 := os.Stat(prettifierPath)
	_, err3 := os.Stat(dechunkerPath)

	return err1 == nil && err2 == nil && err3 == nil
}

var binaries = map[string]string{
	"analyzer":   AnalyzerRepoURL,
	"prettifier": PrettifierRepoURL,
	"dechunker":  DechunkerRepoURL,
}

func downloadBinary(binaryName string) error {
	binaryURL, ok := binaries[binaryName]
	if !ok {
		return fmt.Errorf("binary %s not found", binaryName)
	}

	fileName := getBinaryFileName(binaryName)
	dstPath := filepath.Join(GetLibsDirectory(), fileName)
	platformName := getPlatformSpecificName(binaryName)
	downloadURL := fmt.Sprintf("%s/%s", binaryURL, platformName)

	req, _ := http.NewRequest("GET", downloadURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", binaryName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download %s from %s: HTTP status %d, body: %s", binaryName, downloadURL, resp.StatusCode, string(body))
	}

	f, _ := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	bar := progressbar.NewOptions(int(resp.ContentLength),
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", binaryName)),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)
	bar.RenderBlank()

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write %s binary: %w", binaryName, err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(dstPath, 0755); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", binaryName, err)
		}
	}

	return nil
}
