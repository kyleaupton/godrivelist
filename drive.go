package godrivelist

// Mountpoint represents a disk mountpoint
type Mountpoint struct {
	Path  string `json:"path"`
	Label string `json:"label,omitempty"`
}

// Drive represents a physical drive in the system
type Drive struct {
	Device       string       `json:"device"`
	DisplayName  string       `json:"displayName"`
	Description  string       `json:"description"`
	Size         int64        `json:"size"`
	Mountpoints  []Mountpoint `json:"mountpoints"`
	Raw          string       `json:"raw"`
	Protected    bool         `json:"protected"`
	System       bool         `json:"system"`
	Removable    bool         `json:"removable,omitempty"`
	ReadOnly     bool         `json:"readOnly,omitempty"`
	BlockSize    int64        `json:"blockSize,omitempty"`
	BusType      string       `json:"busType,omitempty"`
	DevicePath   string       `json:"devicePath,omitempty"`
	Enumerator   string       `json:"enumerator,omitempty"`
	IsCard       bool         `json:"isCard,omitempty"`
	IsUSB        bool         `json:"isUSB,omitempty"`
	IsVirtual    bool         `json:"isVirtual,omitempty"`
	IsSCSI       bool         `json:"isSCSI,omitempty"`
	PartitionType string      `json:"partitionTableType,omitempty"`
}

// List returns all connected drives in the system
// This is the main API function mirroring the original drivelist library
func List() ([]Drive, error) {
	return list()
} 