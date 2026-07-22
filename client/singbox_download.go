package client

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tbox/core"
	"tbox/log"
)

const singBoxLatestReleaseURL = "https://api.github.com/repos/SagerNet/sing-box/releases/latest"
const singBoxFallbackVersion = "v1.13.7"

// gitHubMirrorPrefix is the proxy prefix used to accelerate GitHub downloads on mainland-China networks.
const gitHubMirrorPrefix = "https://ghfast.top/"

// mirrorEnvKey allows manual override of mirror behavior:
//   - unset       auto-detect; enable mirror only on mainland-China networks
//   - "off"/"0"   force-disable mirror
//   - other       used as custom mirror prefix (must end with /)
const mirrorEnvKey = "TBOX_GH_MIRROR"

// EnsureSingBoxAvailable ensures sing-box binary exists, downloading if necessary
func EnsureSingBoxAvailable() (string, error) {
	coreName := "sing-box"
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
		return "", fmt.Errorf("create sing-box dir: %w", err)
	}

	log.Info("downloading sing-box...")
	if err := downloadSingBoxLatest(coreDir); err != nil {
		return "", fmt.Errorf("download sing-box: %w", err)
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("sing-box binary not found after download: %w", err)
	}

	log.Info("sing-box download complete: ", binaryPath)
	return binaryPath, nil
}

func downloadSingBoxLatest(targetDir string) error {
	var version, downloadURL string

	release, err := fetchLatestRelease(singBoxLatestReleaseURL)
	if err == nil {
		assetName, err := buildSingBoxAssetName(release.TagName)
		if err == nil {
			for _, asset := range release.Assets {
				if asset.Name == assetName {
					version = release.TagName
					downloadURL = asset.BrowserDownloadURL
					break
				}
			}
		}
	}

	if downloadURL == "" {
		version = singBoxFallbackVersion
		assetName, err := buildSingBoxAssetName(version)
		if err != nil {
			return err
		}
		downloadURL = fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/%s/%s", version, assetName)
	}

	downloadURL = applyGitHubMirror(downloadURL)

	log.Info("sing-box version to download: ", version)
	log.Info("download URL: ", downloadURL)

	tempFile, err := os.CreateTemp("", "sing-box-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if err := downloadFile(downloadURL, tempFile.Name()); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "sing-box-extract-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	assetName, _ := buildSingBoxAssetName(version)
	if strings.HasSuffix(assetName, ".zip") {
		if err := extractZip(tempFile.Name(), tempDir); err != nil {
			return fmt.Errorf("extract zip: %w", err)
		}
	} else {
		if err := extractTarGz(tempFile.Name(), tempDir); err != nil {
			return fmt.Errorf("extract tar.gz: %w", err)
		}
	}

	binaryName := "sing-box"
	if runtime.GOOS == "windows" {
		binaryName = "sing-box.exe"
	}
	binaryPath, err := findExtractedBinary(tempDir, binaryName)
	if err != nil {
		return fmt.Errorf("find binary: %w", err)
	}

	targetBinary := filepath.Join(targetDir, binaryName)
	if err := os.Rename(binaryPath, targetBinary); err != nil {
		if err := copyFile(binaryPath, targetBinary); err != nil {
			return fmt.Errorf("move binary: %w", err)
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetBinary, 0755); err != nil {
			return fmt.Errorf("chmod binary: %w", err)
		}
	}

	return nil
}

// applyGitHubMirror prepends an accelerator mirror prefix to a GitHub download
// URL when the current network appears to be in mainland China. The
// TBOX_GH_MIRROR environment variable can force-disable it or supply a custom
// mirror. Only github.com URLs are rewritten; other hosts (e.g. api.github.com)
// are left untouched.
func applyGitHubMirror(url string) string {
	if !strings.HasPrefix(url, "https://github.com/") {
		return url
	}

	switch env := strings.TrimSpace(os.Getenv(mirrorEnvKey)); strings.ToLower(env) {
	case "":
		// unset — fall through to auto-detect
	case "off", "0", "false", "no":
		return url
	default:
		// custom mirror prefix
		return env + url
	}

	if !isChinaMainlandIP() {
		return url
	}
	log.Info("detected mainland-China network, using mirror to accelerate download")
	return gitHubMirrorPrefix + url
}

// isChinaMainlandIP probes public geo endpoints to determine whether the
// current outbound IP is in mainland China. If the endpoints are unreachable
// or the response can't be parsed we return false, keeping the no-mirror
// behavior. geoProvider describes one country-code lookup endpoint: url is the
// request URL, parse extracts an upper-case two-letter country code (e.g. "CN")
// from the response body; empty string means unparseable.
type geoProvider struct {
	name  string
	url   string
	parse func(body string) string
}

// geoProviders lists country-code lookup endpoints in priority order.
// Prefer HTTPS / Cloudflare-fronted endpoints that are reachable from mainland
// China; ip-api.com is a plaintext fallback. The first endpoint that returns a
// concrete country code is trusted and short-circuits the rest.
var geoProviders = []geoProvider{
	{
		// Cloudflare's own trace endpoint, HTTPS. Returns key=value lines,
		// including loc=XX.
		name: "cloudflare",
		url:  "https://www.cloudflare.com/cdn-cgi/trace",
		parse: func(body string) string {
			for _, line := range strings.Split(body, "\n") {
				if strings.HasPrefix(line, "loc=") {
					return strings.ToUpper(strings.TrimSpace(line[4:]))
				}
			}
			return ""
		},
	},
	{
		// api.ip.sb, Cloudflare-fronted HTTPS, JSON containing country_code.
		name:  "ip.sb",
		url:   "https://api.ip.sb/geoip",
		parse: parseCountryCodeJSON,
	},
	{
		// ip-api.com, plaintext HTTP fallback.
		name: "ip-api",
		url:  "http://ip-api.com/line/?fields=countryCode",
		parse: func(body string) string {
			return strings.ToUpper(strings.TrimSpace(body))
		},
	},
}

// parseCountryCodeJSON pulls the country_code field out of a JSON response.
func parseCountryCodeJSON(body string) string {
	var data struct {
		CountryCode string `json:"country_code"`
	}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(data.CountryCode))
}

// isChinaMainlandIP tries geo endpoints in order to detect whether the
// outbound IP is in mainland China. The first endpoint returning a concrete
// country code is trusted (whether or not it is "CN") and no further endpoints
// are tried; unreachable/unparseable responses fall through to the next; if
// all fail, returns false (no mirror).
func isChinaMainlandIP() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, p := range geoProviders {
		req, err := http.NewRequest("GET", p.url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "tbox/3.0")
		resp, err := client.Do(req)
		if err != nil {
			log.Warnf("geo-location endpoint %s unreachable, trying next: %v", p.name, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Warnf("geo-location endpoint %s returned status %d, trying next", p.name, resp.StatusCode)
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		if err != nil {
			continue
		}
		code := p.parse(string(body))
		if code == "" {
			log.Warnf("geo-location endpoint %s returned unparseable country code, trying next", p.name)
			continue
		}
		// Trust the first concrete country code, skip any remaining endpoints.
		return code == "CN"
	}
	log.Warn("all geo-location endpoints unavailable, skipping mirror")
	return false
}

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func buildSingBoxAssetName(version string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
		"arm":   "arm",
	}

	arch, ok := archMap[goarch]
	if !ok {
		return "", fmt.Errorf("unsupported architecture: %s/%s", goos, goarch)
	}

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	version = strings.TrimPrefix(version, "v")
	return fmt.Sprintf("sing-box-%s-%s-%s.%s", version, goos, arch, ext), nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(dest, header.Name)
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			if err := os.Chmod(path, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func findExtractedBinary(root, name string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == name {
			found = path
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("binary %s not found in extracted archive", name)
	}
	if err != nil && err != io.EOF {
		return "", err
	}
	return found, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
