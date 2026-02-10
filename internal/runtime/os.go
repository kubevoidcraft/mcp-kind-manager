// Package runtime provides OS and container runtime detection.
package runtime

import (
	"fmt"
	"runtime"
)

// OSInfo holds information about the host operating system.
type OSInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Platform     string `json:"platform"`
	PlatformNote string `json:"platform_note,omitempty"`
}

// DetectOS returns information about the host operating system.
func DetectOS() OSInfo {
	info := OSInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "darwin":
		info.Platform = "macOS"
		info.PlatformNote = "Containers run in a VM. Docker Desktop, Colima, Podman Machine, or Lima required."
	case "linux":
		info.Platform = "Linux"
		info.PlatformNote = "Native container support. Docker or Podman can run directly. WSL2 environment possible."
	case "windows":
		info.Platform = "Windows"
		info.PlatformNote = "Containers typically run in WSL2 or Hyper-V via Docker Desktop or Podman Machine."
	default:
		info.Platform = runtime.GOOS
		info.PlatformNote = fmt.Sprintf("Unsupported platform: %s", runtime.GOOS)
	}

	return info
}
