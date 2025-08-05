package main

import (
	"log"
	"os"

	"github.com/davidschrooten/open-atlas-search/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Printf("Error executing command: %v", err)
		os.Exit(1)
	}
}
