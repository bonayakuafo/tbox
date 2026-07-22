package client

import (
	"runtime"
	"testing"
)

// TestBuildXrayAssetName verifies that a valid xray release asset name can be
// built for the current platform.
func TestBuildXrayAssetName(t *testing.T) {
	name, err := buildXrayAssetName()
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" ||
		runtime.GOARCH == "386" || runtime.GOARCH == "arm" {
		if err != nil {
			t.Fatalf("current architecture should be supported but errored: %v", err)
		}
		if name == "" {
			t.Fatal("asset name should not be empty")
		}
		// e.g. Xray-linux-64.zip / Xray-macos-arm64-v8a.zip
		if name[:5] != "Xray-" || name[len(name)-4:] != ".zip" {
			t.Errorf("unexpected asset name format: %s", name)
		}
	}
}

// TestBuildXrayAssetName_ArchMapping checks the naming rule against a fixed mapping.
func TestBuildXrayAssetName_ArchMapping(t *testing.T) {
	// Only assert an exact value on amd64 to avoid coupling to the runtime arch.
	if runtime.GOARCH != "amd64" {
		t.Skip("only assert exact name on amd64")
	}
	name, err := buildXrayAssetName()
	if err != nil {
		t.Fatal(err)
	}
	want := "Xray-" + xrayOSName() + "-64.zip"
	if name != want {
		t.Errorf("expected %s, got %s", want, name)
	}
}

// xrayOSName mirrors the OS-naming used in buildXrayAssetName, for test assertions.
func xrayOSName() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}
