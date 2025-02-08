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
                                Key    []string `xml:"key"`
                                String []string `xml:"string"`
                                Int    []int64  `xml:"integer"`
                                Bool   []bool   `xml:"false,true"`
                        } `xml:"dict"`
                } `xml:"array"`
        } `xml:"dict"`
}

type dictValue struct {
        stringVal string
        intVal    int64
        boolVal   bool
        valueType string // "string", "int", or "bool"
}

func findValueByKey(dict struct {
        Key    []string `xml:"key"`
        String []string `xml:"string"`
        Int    []int64  `xml:"integer"`
        Bool   []bool   `xml:"false,true"`
}, searchKey string) dictValue {
        valueMap := make(map[string]int)
        for i, key := range dict.Key {
                valueMap[key] = i
        }

        if idx, exists := valueMap[searchKey]; exists {
                switch searchKey {
                case "Size", "TotalSize":
                        if idx < len(dict.Int) {
                                return dictValue{intVal: dict.Int[idx], valueType: "int"}
                        }
                case "Internal", "Ejectable", "WritableMedia":
                        if idx < len(dict.Bool) {
                                return dictValue{boolVal: dict.Bool[idx], valueType: "bool"}
                        }
                default:
                        if idx < len(dict.String) {
                                return dictValue{stringVal: dict.String[idx], valueType: "string"}
                        }
                }
        }
        return dictValue{} // Return empty value if not found or index out of bounds
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
                devIDValue := findValueByKey(dict, "DeviceIdentifier")
                if devIDValue.valueType == "string" {
                        device = "/dev/" + devIDValue.stringVal
                } else {
                        continue
                }

                // Get device node (overrides device identifier if present)
                if devNodeValue := findValueByKey(dict, "DeviceNode"); devNodeValue.valueType == "string" {
                        device = devNodeValue.stringVal
                }

                // Get volume name for description
                if volNameValue := findValueByKey(dict, "VolumeName"); volNameValue.valueType == "string" {
                        description = volNameValue.stringVal
                }

                // Get size
                if sizeValue := findValueByKey(dict, "Size"); sizeValue.valueType == "int" {
                        size = sizeValue.intVal
                }

                // Get mount point
                if mountPointValue := findValueByKey(dict, "MountPoint"); mountPointValue.valueType == "string" && mountPointValue.stringVal != "" {
                        mountpoints = append(mountpoints, Mountpoint{Path: mountPointValue.stringVal})
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
                        if internalValue := findValueByKey(infoDict, "Internal"); internalValue.valueType == "bool" {
                                isSystem = internalValue.boolVal
                        }
                        if ejectableValue := findValueByKey(infoDict, "Ejectable"); ejectableValue.valueType == "bool" {
                                isSystem = isSystem || !ejectableValue.boolVal
                        }
                        if writableValue := findValueByKey(infoDict, "WritableMedia"); writableValue.valueType == "bool" {
                                isProtected = !writableValue.boolVal
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
