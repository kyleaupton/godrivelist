package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/kyleaupton/godrivelist"
)

func main() {
	// Get the list of drives
	drives, err := godrivelist.List()
	if err != nil {
		log.Fatal("Error getting drive list:", err)
	}

	// Pretty print the drives as JSON
	output, err := json.MarshalIndent(drives, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling to JSON:", err)
	}

	fmt.Println(string(output))
} 