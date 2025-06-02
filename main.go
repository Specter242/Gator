package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Specter242/Gator/internal/config"
	"github.com/Specter242/Gator/internal/database"
	_ "github.com/lib/pq" // Required for PostgreSQL driver
)

func main() {
	// Use empty string to get the home directory in config functions
	homeDir := ""

	// Read the existing configuration from the home directory
	cfg, err := config.Read(homeDir)
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

	// Open a database connection
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close() // Ensure the database connection is closed

	// Create a new instance of the Queries struct with the database connection
	dbQueries := database.New(db)

	// Write the updated config back to disk in the home directory
	err = config.Write(homeDir, cfg)
	if err != nil {
		log.Fatalf("Error writing config file: %v", err)
	}

	// Create a new instance of commands
	cmds := &commands{}

	// Register available commands
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerGetUsers)
	cmds.register("agg", handlerFetchFeed)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))

	// Initialize application state
	appState := &state{
		db:     dbQueries,
		Config: &cfg,
	}

	// If there are command-line args, process them as a command
	if len(os.Args) > 1 {
		cmd := command{
			Name: os.Args[1],
			Args: os.Args[2:],
		}

		if err := cmds.run(appState, cmd); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Read the config again to verify changes were persisted
		//updatedCfg, err := config.Read(homeDir)
		//if err != nil {
		//	fmt.Printf("Warning: Could not verify config update: %v\n", err)
		//} else {
		//	fmt.Printf("Configuration updated:\n")
		//fmt.Printf("  Current User: %s\n", updatedCfg.CurrentUserName)
		//	fmt.Printf("  Config Path: %s\n", config.GetConfigPath(homeDir))
		//}

	} else {
		// Display error message and exit with code 1 when no arguments provided
		fmt.Println("Error: Not enough arguments")
		fmt.Println("\nAvailable commands:")
		fmt.Println("  login <username> - Log in as the specified user")
		fmt.Println("  register <username> - Register a new user")
		fmt.Println("  reset - Reset the database")
		fmt.Println("  users - Get all users")
		fmt.Println("  agg - run aggregator service")
		fmt.Println("  addfeed <name> <url> - Add a new feed with the specified name and URL")
		fmt.Println("  feeds - List all feeds")
		fmt.Println("  follow <url> - Follow a feed by URL")
		fmt.Println("  following - List all followed feeds")
		fmt.Println("  help - Show this help message")
		os.Exit(1)
	}
}
