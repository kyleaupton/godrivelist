// +build linux

package godrivelist

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

// lsblkOutput represents the JSON output from the lsblk command
type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

// lsblkDevice represents a device in the lsblk output
type lsblkDevice struct {
	Name       string        `json:"name"`
	Label      string        `json:"label"`
	Size       interface{}   `json:"size"`
	Type       string        `json:"type"`
	ReadOnly   bool          `json:"ro"`
	Removable  bool          `json:"rm"`
	Model      string        `json:"model"`
	Mountpoint string        `json:"mountpoint"`
	Children   []lsblkDevice `json:"children"`
	FSType     string        `json:"fstype"`
	PHSector   interface{}   `json:"phy-sec"`
	Vendor     string        `json:"vendor"`
	Serial     string        `json:"serial"`
	PTType     string        `json:"pttype"`
}

// parseInterfaceToInt64 converts various types to int64
func parseInterfaceToInt64(value interface{}, defaultVal int64) int64 {
	if value == nil {
		return defaultVal
	}
	
	switch v := value.(type) {
	case string:
		result, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return defaultVal
		}
		return result
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case json.Number:
		result, err := v.Int64()
		if err != nil {
			return defaultVal
		}
		return result
	default:
		return defaultVal
	}
}

// list returns all connected drives in the system for Linux
func list() ([]Drive, error) {
	cmd := exec.Command("lsblk", "--json", "--bytes", "--output-all")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var lsblkData lsblkOutput
	err = json.Unmarshal(output, &lsblkData)
	if err != nil {
		return nil, err
	}

	var drives []Drive
	for _, device := range lsblkData.Blockdevices {
		if device.Type != "disk" {
			continue
		}

		// Parse size and block size using our helper function
		size := parseInterfaceToInt64(device.Size, 0)
		blockSize := parseInterfaceToInt64(device.PHSector, 512) // Default block size is 512

		drive := Drive{
			Device:       "/dev/" + device.Name,
			DisplayName:  "/dev/" + device.Name,
			Description:  strings.TrimSpace(device.Vendor + " " + device.Model),
			Size:         size,
			Raw:          "/dev/" + device.Name,
			Protected:    device.ReadOnly,
			System:       false, // Will be updated below
			Removable:    device.Removable,
			ReadOnly:     device.ReadOnly,
			BlockSize:    blockSize,
			PartitionType: device.PTType,
		}

		// Set mountpoints
		drive.Mountpoints = []Mountpoint{}
		if device.Mountpoint != "" {
			drive.Mountpoints = append(drive.Mountpoints, Mountpoint{
				Path:  device.Mountpoint,
				Label: device.Label,
			})
		}

		// Add mountpoints from children (partitions)
		for _, child := range device.Children {
			if child.Mountpoint != "" {
				drive.Mountpoints = append(drive.Mountpoints, Mountpoint{
					Path:  child.Mountpoint,
					Label: child.Label,
				})
			}
			// Check if any partition is mounted at root, if so mark as system drive
			if child.Mountpoint == "/" {
				drive.System = true
			}
		}

		drives = append(drives, drive)
	}

	return drives, nil
} 