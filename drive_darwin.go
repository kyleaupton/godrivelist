//go:build darwin

package godrivelist

import (
        "encoding/xml"
        "os/exec"
        "strconv"
        "strings"
)

type diskutilList struct {
        XMLName xml.Name `xml:"plist"`
        Dict    struct {
                Array struct {
                        Dict []struct {
                                Key    []string `xml:"key"`
                                String []string `xml:"string"`
                                Int    []int64  `xml:"integer"`
                        } `xml:"dict"`
                } `xml:"array"`
        } `xml:"dict"`
}

func findValueByKey(dict struct {
        Key    []string `xml:"key"`
        String []string `xml:"string"`
        Int    []int64  `xml:"integer"`
}, searchKey string) (string, int64, bool) {
        stringIdx := 0
        intIdx := 0
        for _, key := range dict.Key {
                if key == searchKey {
                        if searchKey == "Size" && len(dict.Int) > intIdx {
                                return "", dict.Int[intIdx], true
                        } else if len(dict.String) > stringIdx {
                                return dict.String[stringIdx], 0, true
                        }
                        return "", 0, false
                }
                // Only increment the appropriate index based on the expected type
                switch key {
                case "Size", "TotalSize":
                        intIdx++
                default:
                        stringIdx++
                }
        }
        return "", 0, false
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

                // Get device identifier
                if devID, _, ok := findValueByKey(dict, "DeviceIdentifier"); ok {
                        device = "/dev/" + devID
                } else {
                        continue
                }

                // Get device node (overrides device identifier if present)
                if devNode, _, ok := findValueByKey(dict, "DeviceNode"); ok {
                        device = devNode
                }

                // Get volume name for description
                if volName, _, ok := findValueByKey(dict, "VolumeName"); ok {
                        description = volName
                }

                // Get size
                if _, sz, ok := findValueByKey(dict, "Size"); ok {
                        size = sz
                }

                // Get mount point
                if mountPoint, _, ok := findValueByKey(dict, "MountPoint"); ok && mountPoint != "" {
                        mountpoints = append(mountpoints, Mountpoint{Path: mountPoint})
                }

                // Get additional disk info
                cmd = exec.Command("diskutil", "info", "-plist", device)
                output, err = cmd.Output()
                if err != nil {
                        continue
                }

                var infoList diskutilList
                if err := xml.Unmarshal(output, &infoList); err != nil {
                        continue
                }

                isSystem := false
                isProtected := false

                // Process disk info
                for _, infoDict := range infoList.Dict.Array.Dict {
                        if internal, _, ok := findValueByKey(infoDict, "Internal"); ok {
                                isSystem = internal == "Yes"
                        }
                        if ejectable, _, ok := findValueByKey(infoDict, "Ejectable"); ok {
                                isSystem = isSystem || ejectable == "No"
                        }
                        if writable, _, ok := findValueByKey(infoDict, "WritableMedia"); ok {
                                isProtected = writable == "No"
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