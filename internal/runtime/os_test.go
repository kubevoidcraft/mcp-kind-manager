package runtime

import (
	"runtime"
	"testing"
)

func TestDetectOS(t *testing.T) {
	info := DetectOS()

	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	switch runtime.GOOS {
	case "darwin":
		if info.Platform != "macOS" {
			t.Errorf("Platform = %q, want %q", info.Platform, "macOS")
		}
	case "linux":
		if info.Platform != "Linux" {
			t.Errorf("Platform = %q, want %q", info.Platform, "Linux")
		}
	case "windows":
		if info.Platform != "Windows" {
			t.Errorf("Platform = %q, want %q", info.Platform, "Windows")
		}
	}

	if info.PlatformNote == "" {
		t.Error("PlatformNote should not be empty")
	}
}
