package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"tbox/core"
	"tbox/log"
)

const xrayLatestReleaseURL = "https://api.github.com/repos/XTLS/Xray-core/releases/latest"
const xrayFallbackVersion = "v1.8.24"

// EnsureXrayAvailable ensures the xray binary exists, downloading it from
// GitHub if missing. The xray release zip contains the xray executable plus
// geoip.dat and geosite.dat; all are extracted together.
func EnsureXrayAvailable() (string, error) {
	coreName := "xray"
	coreDir := core.GetCoreDir(coreName)
	binaryName := coreName
	if runtime.GOOS == "windows" {
		binaryName = coreName + ".exe"
	}
	binaryPath := filepath.Join(coreDir, binaryName)

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	if err := os.MkdirAll(coreDir, 0755); err != nil {
		return "", fmt.Errorf("create xray dir: %w", err)
	}

	log.Info("downloading xray...")
	if err := downloadXrayLatest(coreDir); err != nil {
		return "", fmt.Errorf("download xray: %w", err)
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("xray binary not found after download: %w", err)
	}

	log.Info("xray download complete: ", binaryPath)
	return binaryPath, nil
}

func downloadXrayLatest(targetDir string) error {
	assetName, err := buildXrayAssetName()
	if err != nil {
		return err
	}

	var version, downloadURL string
	if release, err := fetchLatestRelease(xrayLatestReleaseURL); err == nil {
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				version = release.TagName
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}
	if downloadURL == "" {
		version = xrayFallbackVersion
		downloadURL = fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/%s", version, assetName)
	}

	downloadURL = applyGitHubMirror(downloadURL)
	log.Info("xray version to download: ", version)
	log.Info("download URL: ", downloadURL)

	tempFile, err := os.CreateTemp("", "xray-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	if err := downloadFile(downloadURL, tempFile.Name()); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// xray releases are always zip files; extract directly into the target
	// directory (includes xray + geoip.dat + geosite.dat).
	if err := extractZip(tempFile.Name(), targetDir); err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}

	binaryName := "xray"
	if runtime.GOOS == "windows" {
		binaryName = "xray.exe"
	}
	binaryPath := filepath.Join(targetDir, binaryName)
	if _, err := os.Stat(binaryPath); err != nil {
		// fallback: some packages have subdirectories; recurse to find it
		found, ferr := findExtractedBinary(targetDir, binaryName)
		if ferr != nil {
			return fmt.Errorf("find binary: %w", ferr)
		}
		if found != binaryPath {
			if err := copyFile(found, binaryPath); err != nil {
				return fmt.Errorf("move binary: %w", err)
			}
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("chmod binary: %w", err)
		}
	}

	return nil
}

// buildXrayAssetName constructs the xray release asset name based on the
// runtime. Naming follows Xray-core releases: amd64->64, 386->32,
// arm64->arm64-v8a, arm->arm32-v7a.
func buildXrayAssetName() (string, error) {
	goos := runtime.GOOS
	archMap := map[string]string{
		"amd64": "64",
		"386":   "32",
		"arm64": "arm64-v8a",
		"arm":   "arm32-v7a",
	}
	arch, ok := archMap[runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("unsupported architecture: %s/%s", goos, runtime.GOARCH)
	}

	osName := goos
	switch goos {
	case "darwin":
		osName = "macos"
	}
	return fmt.Sprintf("Xray-%s-%s.zip", osName, arch), nil
}

// fetchLatestRelease is a shared GitHub-latest-release fetcher for use across cores.
func fetchLatestRelease(apiURL string) (*releaseInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "tbox/3.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	var release releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &release, nil
}
