# godrivelist

A Go library to list all connected drives in your computer, supporting all major operating systems (Windows, Linux, and macOS). This library is inspired by the Node.js [drivelist](https://github.com/balena-io-modules/drivelist) package.

## Features

- Cross-platform support (Windows, Linux, macOS)
- No admin privileges required
- Returns detailed drive information including:
  - Device path
  - Display name
  - Description
  - Size
  - Mountpoints
  - Raw device path
  - Protected status
  - System drive status

## Installation

```bash
go get github.com/kyleaupton/godrivelist
```

## Usage

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/kyleaupton/godrivelist"
)

func main() {
    drives, err := godrivelist.List()
    if err != nil {
        log.Fatal(err)
    }

    output, err := json.MarshalIndent(drives, "", "  ")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(string(output))
}
```

Example output (Linux):
```json
[
  {
    "device": "/dev/sda",
    "displayName": "/dev/sda",
    "description": "WDC WD10JPVX-75J",
    "size": 1000204886016,
    "mountpoints": [
      {
        "path": "/"
      }
    ],
    "raw": "/dev/sda",
    "protected": false,
    "system": true
  },
  {
    "device": "/dev/sdb",
    "displayName": "/dev/sdb",
    "description": "Kingston DataTraveler",
    "size": 32010928128,
    "mountpoints": [
      {
        "path": "/media/user/Kingston"
      }
    ],
    "raw": "/dev/sdb",
    "protected": false,
    "system": false
  }
]
```

## Project Structure

The project has platform-specific implementations:

- Common code: `drive.go`
- Linux implementation: `drive_linux.go`
- Windows implementation: `drive_windows.go`
- macOS implementation: 
  - Go code: `drive_darwin.go`
  - Objective-C code: `darwin/disklist.h` and `darwin/disklist.m`

## Platform-specific Notes

### Linux
- Uses `lsblk` command to get drive information
- Requires `util-linux` package (usually pre-installed)

### Windows
- Uses Windows API (kernel32.dll) to get drive information
- Lists all available drives except CD-ROM drives

### macOS
- Uses DiskArbitration framework and IOKit to get disk information
- Provides raw device paths with `/dev/rdisk` prefix

## License

Apache-2.0