package core

import (
	"os"
	"path/filepath"
)

var (
	DataFile    = filepath.Join(GetConfigDir(), "tbox.data.json")
	SettingFile = filepath.Join(GetConfigDir(), "tbox.setting.toml")
	RoutingFile = filepath.Join(GetConfigDir(), "tbox.routing.json")
	LogFile     = filepath.Join(GetConfigDir(), "tbox.core.log")
)

// init migrates legacy config filenames to the unified tbox.* naming.
// Renames only when the new file is absent and the old file exists (one-shot,
// lossless upgrade). The sing-box split rules file migration is handled in
// the core/singbox_split package.
func init() {
	dir := GetConfigDir()
	migrations := map[string]string{
		"data.json":       DataFile,
		"setting.toml":    SettingFile,
		"routing.json":    RoutingFile,
		"core_access.log": LogFile,
	}
	for oldName, newPath := range migrations {
		oldPath := filepath.Join(dir, oldName)
		if _, err := os.Stat(newPath); err == nil {
			continue // new file already exists, do not overwrite
		}
		if _, err := os.Stat(oldPath); err != nil {
			continue // old file does not exist, no migration needed
		}
		_ = os.Rename(oldPath, newPath)
	}
}

// GetConfigDir returns the directory containing config files.
func GetConfigDir() string {
	dir := os.Getenv("TBOX_HOME")
	if dir != "" && IsDir(dir) {
		return dir
	}
	return GetRunPath()
}

// GetRunPath returns the path of the current process executable's directory.
func GetRunPath() string {
	path, _ := os.Executable()
	return filepath.Dir(path)
}

// GetCoreDir returns the runtime directory for the given core.
func GetCoreDir(coreName string) string {
	return filepath.Join(GetConfigDir(), coreName)
}

// IsDir reports whether the path exists and is a directory.
func IsDir(path string) bool {
	i, err := os.Stat(path)
	if err == nil {
		return i.IsDir()
	}
	return false
}
