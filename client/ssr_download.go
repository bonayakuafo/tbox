package client

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"tbox/core"
	"tbox/log"
)

// ssrLatestReleaseURL points at the ssr converter repo's GitHub latest-release API.
const ssrLatestReleaseURL = "https://api.github.com/repos/bonayakuafo/ssr/releases/latest"

// EnsureSSRAvailable ensures the ssr converter binary exists, downloading it
// from GitHub if missing. Unlike sing-box/xray, the ssr release is a direct
// executable (not an archive); we just chmod it after download.
func EnsureSSRAvailable() (string, error) {
	coreName := "ssr"
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
		return "", fmt.Errorf("create ssr dir: %w", err)
	}

	log.Info("downloading ssr converter...")
	if err := downloadSSRLatest(binaryPath); err != nil {
		return "", fmt.Errorf("download ssr: %w", err)
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("ssr binary not found after download: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return "", fmt.Errorf("chmod ssr binary: %w", err)
		}
	}
	log.Info("ssr converter download complete: ", binaryPath)
	return binaryPath, nil
}

// downloadSSRLatest downloads the ssr executable matching the current platform
// to binaryPath. The ssr release publishes the executable directly (named e.g.
// ssr-linux-amd64), so there is no archive to extract.
func downloadSSRLatest(binaryPath string) error {
	assetName, err := buildSSRAssetName()
	if err != nil {
		return err
	}

	var version, downloadURL string
	if release, err := fetchLatestRelease(ssrLatestReleaseURL); err == nil {
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				version = release.TagName
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no ssr asset matching the current platform found: %s (the repo may not have released yet)", assetName)
	}

	downloadURL = applyGitHubMirror(downloadURL)
	log.Info("ssr version to download: ", version)
	log.Info("download URL: ", downloadURL)

	return downloadFile(downloadURL, binaryPath)
}

// buildSSRAssetName builds the ssr release asset name based on the runtime.
// Naming follows the ssr project release: ssr-<os>-<arch>[.exe], e.g.
// ssr-linux-amd64 / ssr-darwin-arm64 / ssr-windows-amd64.exe.
func buildSSRAssetName() (string, error) {
	goos := runtime.GOOS
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", fmt.Errorf("unsupported OS for ssr: %s", goos)
	}
	name := fmt.Sprintf("ssr-%s-%s", goos, runtime.GOARCH)
	if goos == "windows" {
		name += ".exe"
	}
	return name, nil
}
