package godrivelist

import (
	"fmt"
	"testing"
)

func TestList(t *testing.T) {
	drives, err := List()
	if err != nil {
		t.Fatalf("List() failed with error: %v", err)
	}
	
	// Print the drives for debugging
	fmt.Printf("Found %d drives\n", len(drives))
	for i, drive := range drives {
		fmt.Printf("Drive %d: %s (%s) - Size: %d bytes\n", 
			i, drive.Device, drive.Description, drive.Size)
		fmt.Printf("  Mountpoints: %d, System: %v, Removable: %v\n", 
			len(drive.Mountpoints), drive.System, drive.Removable)
	}
	
	// Basic assertion - we should have at least one drive
	if len(drives) == 0 {
		t.Error("No drives found, expected at least one drive")
	}
}