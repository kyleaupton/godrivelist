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