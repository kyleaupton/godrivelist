//go:build darwin

package godrivelist

import (
	"encoding/xml"
	"os/exec"
	"strings"
)

type diskutilList struct {
	XMLName xml.Name `xml:"plist"`
	Dict    struct {
		Array struct {
			Dict []struct {
				Key   []string `xml:"key"`
				Value []string `xml:"string"`
				Int   []int64  `xml:"integer"`
			} `xml:"dict"`
		} `xml:"array"`
	} `xml:"dict"`
}

func list() ([]Drive, error) {
	// Use diskutil to get drive information
	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var plist diskutilList
	if err := xml.Unmarshal(output, &plist); err != nil {
		return nil, err
	}

	var drives []Drive
	for _, dict := range plist.Dict.Array.Dict {
		var device, description string
		var size int64
		var mountpoints []Mountpoint

		for i, key := range dict.Key {
			switch key {
			case "DeviceIdentifier":
				device = "/dev/" + dict.Value[i]
			case "DeviceNode":
				device = dict.Value[i]
			case "VolumeName":
				description = dict.Value[i]
			case "Size":
				size = dict.Int[0]
			case "MountPoint":
				if dict.Value[i] != "" {
					mountpoints = append(mountpoints, Mountpoint{Path: dict.Value[i]})
				}
			}
		}

		// Get additional disk info
		cmd = exec.Command("diskutil", "info", "-plist", device)
		output, err = cmd.Output()
		if err != nil {
			continue
		}

		var infoList struct {
			XMLName xml.Name `xml:"plist"`
			Dict    struct {
				Key   []string `xml:"key"`
				Value []string `xml:"string"`
				Bool  []bool   `xml:"false,true"`
			} `xml:"dict"`
		}

		if err := xml.Unmarshal(output, &infoList); err != nil {
			continue
		}

		var isSystem, isProtected bool
		for i, key := range infoList.Dict.Key {
			switch key {
			case "Internal":
				isSystem = infoList.Dict.Bool[i]
			case "Ejectable":
				isSystem = !infoList.Dict.Bool[i]
			case "WritableMedia":
				isProtected = !infoList.Dict.Bool[i]
			}
		}

		drive := Drive{
			Device:      device,
			DisplayName: device,
			Description: description,
			Size:        size,
			Mountpoints: mountpoints,
			Raw:         strings.Replace(device, "/dev/", "/dev/r", 1),
			Protected:   isProtected,
			System:      isSystem,
		}

		drives = append(drives, drive)
	}

	return drives, nil
}
