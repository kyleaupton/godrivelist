package godrivelist

type Mountpoint struct {
	Path string `json:"path"`
}

type Drive struct {
	Device      string       `json:"device"`
	DisplayName string       `json:"displayName"`
	Description string       `json:"description"`
	Size        int64        `json:"size"`
	Mountpoints []Mountpoint `json:"mountpoints"`
	Raw         string       `json:"raw"`
	Protected   bool         `json:"protected"`
	System      bool         `json:"system"`
}

// List returns all connected drives in the system
func List() ([]Drive, error) {
	return list()
}
