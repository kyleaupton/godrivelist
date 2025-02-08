//go:build linux

package godrivelist

import (
	"encoding/json"
	"os/exec"
	"strings"
)

func list() ([]Drive, error) {
	// Use lsblk to get drive information
	cmd := exec.Command("lsblk", "--json", "--bytes", "--output", "NAME,SIZE,MOUNTPOINT,RO,RM,MODEL")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse lsblk JSON output
	var result struct {
		Blockdevices []struct {
			Name       string `json:"name"`
			Size       int64  `json:"size"`
			Mountpoint string `json:"mountpoint"`
			Ro         bool   `json:"ro"`
			Rm         bool   `json:"rm"`
			Model      string `json:"model"`
		} `json:"blockdevices"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	var drives []Drive
	for _, device := range result.Blockdevices {
		// Skip partition entries
		if strings.Contains(device.Name, "loop") || len(device.Name) > 3 {
			continue
		}

		var mountpoints []Mountpoint
		if device.Mountpoint != "" {
			mountpoints = append(mountpoints, Mountpoint{Path: device.Mountpoint})
		}

		drive := Drive{
			Device:      "/dev/" + device.Name,
			DisplayName: "/dev/" + device.Name,
			Description: device.Model,
			Size:        device.Size,
			Mountpoints: mountpoints,
			Raw:         "/dev/" + device.Name,
			Protected:   device.Ro,
			System:      !device.Rm && device.Mountpoint == "/",
		}

		drives = append(drives, drive)
	}

	return drives, nil
}
