package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Specter242/Gator/internal/config"
)

func main() {
	// Get the current working directory instead of the home directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	// Read the existing configuration from the working directory
	cfg, err := config.Read(workDir)
	if err != nil {
		// If config doesn't exist yet, create a new one
		if os.IsNotExist(err) {
			cfg = config.Config{
				DBURL:           "postgres://example",
				CurrentUserName: "",
			}
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}

	// Set the current user to "Specter"
	cfg.CurrentUserName = "Specter"

	// Write the updated config back to disk
	err = config.Write(workDir, cfg)
	if err != nil {
		log.Fatalf("Error writing config file: %v", err)
	}

	// Print updated confirmation with the full config contents
	fmt.Println("Config updated successfully!")
	fmt.Println("Config file contents:")
	fmt.Printf("  Database URL: %s\n", cfg.DBURL)
	fmt.Printf("  Current User: %s\n", cfg.CurrentUserName)
	fmt.Printf("  Config Path: %s\n", config.GetConfigPath(workDir))
}
