package client

import (
	"tbox/client/config"
	"tbox/core"
	"tbox/core/setting/key"
	"tbox/log"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

var CoreName = ""
var CorePath = ""

func init() {
	CoreName = viper.GetString(key.ClientCore)
	path, err := resolveCorePath(CoreName)
	if err != nil {
		log.Error(err)
		log.Error("please download the latest version from ", config.Releases[CoreName])
		log.Error("and move the extracted file to ", core.GetCoreDir(CoreName))
		os.Exit(0)
	}
	CorePath = path

	if err := validateCoreResources(CoreName, CorePath); err != nil {
		log.Error(err)
		os.Exit(0)
	}
	// cache the resolved path so nodes using the default core skip re-resolution
	coreCache[CoreName] = CorePath
}

// coreCache caches resolved core binary paths to avoid re-lookup/re-download per node.
var coreCache = map[string]string{}

// ResolveCore resolves (downloading if needed) the binary path for the given core
// name and validates its resource files. Called at runtime when picking a core per
// node (SelectCore).
func ResolveCore(coreName string) (string, error) {
	if p, ok := coreCache[coreName]; ok {
		return p, nil
	}
	path, err := resolveCorePath(coreName)
	if err != nil {
		return "", err
	}
	if err := validateCoreResources(coreName, path); err != nil {
		return "", err
	}
	coreCache[coreName] = path
	return path, nil
}

func resolveCorePath(coreName string) (string, error) {
	// 1. check the CORE_HOME env var directory
	corePath := os.Getenv("CORE_HOME")
	if corePath != "" {
		if IsExistExe(corePath, coreName) {
			return filepath.Join(corePath, coreName), nil
		}
	}

	// 2. check the core-specific subdirectory
	coreDir := core.GetCoreDir(coreName)
	if IsExistExe(coreDir, coreName) {
		return filepath.Join(coreDir, coreName), nil
	}

	// 3. check the current executable's directory (recursively)
	path, _ := os.Executable()
	files, _ := FindFileByName(filepath.Dir(path), coreName, ".exe")
	if len(files) != 0 {
		return files[0], nil
	}

	// 4. check the PATH environment variable
	if temp := getExePath(coreName); temp != "" {
		return temp, nil
	}

	// 5. auto-download (sing-box / xray)
	switch coreName {
	case "sing-box":
		binaryPath, err := EnsureSingBoxAvailable()
		if err != nil {
			return "", fmt.Errorf("auto-download of sing-box failed: %w", err)
		}
		return binaryPath, nil
	case "xray":
		binaryPath, err := EnsureXrayAvailable()
		if err != nil {
			return "", fmt.Errorf("auto-download of xray failed: %w", err)
		}
		return binaryPath, nil
	case "ssr":
		binaryPath, err := EnsureSSRAvailable()
		if err != nil {
			return "", fmt.Errorf("auto-download of ssr failed: %w", err)
		}
		return binaryPath, nil
	}

	return "", fmt.Errorf("could not find the %s program under %s", coreName, filepath.Dir(path))
}

func validateCoreResources(coreName, corePath string) error {
	// sing-box and the ssr converter do not need geoip.dat or geosite.dat
	if coreName == "sing-box" || coreName == "ssr" {
		return nil
	}

	if XrayAssetDir(corePath) != "" {
		return nil
	}
	return fmt.Errorf("could not find resource files geoip.dat and geosite.dat under %s\nor set the CORE_LOCATION_ASSET environment variable to point at them", filepath.Dir(corePath))
}

// XrayAssetDir returns a directory containing both geoip.dat and geosite.dat, or
// an empty string if none is found. Used by validateCoreResources and when running
// xray to set XRAY_LOCATION_ASSET so that the xray child process can locate its
// resource files (the xray binary may live in a different directory from its
// assets, e.g. when resolved from PATH or CORE_HOME).
func XrayAssetDir(corePath string) string {
	candidates := []string{
		os.Getenv("CORE_LOCATION_ASSET"),
		os.Getenv("core.location.asset"),
		filepath.Dir(corePath),
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if IsExistFile(filepath.Join(dir, "geoip.dat")) && IsExistFile(filepath.Join(dir, "geosite.dat")) {
			return dir
		}
	}
	return ""
}

func IsExistFile(file string) bool {
	fp, err := os.Stat(file)
	return err == nil && !fp.IsDir()
}

// check whether the filename program exists under dirPath
func IsExistExe(dirPath, filename string) bool {
	if runtime.GOOS == "windows" {
		fp, err := os.Stat(filepath.Join(dirPath, filename+".exe"))
		if err == nil && !fp.IsDir() {
			return true
		}
	}
	fp, err := os.Stat(filepath.Join(dirPath, filename))
	return err == nil && !fp.IsDir()
}

// walk the directory to find a file
func FindFileByName(root, name, ext string) ([]string, error) {
	root = strings.TrimRight(root, string(os.PathSeparator))
	paths, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	objList := make([]string, 0)
	for _, p := range paths {
		absPath := root + string(os.PathSeparator) + p.Name()
		if p.IsDir() {
			o, err := FindFileByName(absPath, name, ext)
			if err != nil {
				continue
			}
			objList = append(objList, o...)
		} else {
			if p.Name() == name || p.Name() == name+ext {
				objList = append(objList, absPath)
			}
		}
	}
	return objList, nil
}

// look up an executable's path in PATH
func getExePath(name string) string {
	data := os.Getenv("PATH")
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	for _, x := range strings.Split(data, sep) {
		if strings.TrimSpace(x) != "" {
			if IsExistExe(strings.TrimSpace(x), name) {
				return filepath.Join(strings.TrimSpace(x), name)
			}
		}
	}
	return ""
}
