package main

import (
	"fmt"

	"github.com/Specter242/Gator/internal/config"
)

// State holds a pointer to a config.Config struct
type state struct {
	Config *config.Config
}

// command represents a command with its name and arguments
// It is used to parse and execute commands in the application
type command struct {
	// Name is the name of the command
	Name string
	// Args are the arguments for the command
	Args []string
}

// commands is a map of command names to their respective handler functions
type commands struct {
	Commandmap map[string]func(*state, command) error
}

// register adds a new command handler to the command map
// It takes a command name and a handler function as arguments
// If the command map is nil, it initializes it first
func (c *commands) register(name string, f func(*state, command) error) {
	if c.Commandmap == nil {
		c.Commandmap = make(map[string]func(*state, command) error)
	}
	c.Commandmap[name] = f
}

// run executes the command handler for the given command
// It checks if the command exists in the command map
// and calls the corresponding handler function
// If the command is not found, it returns an error
func (c *commands) run(s *state, cmd command) error {
	if handler, ok := c.Commandmap[cmd.Name]; ok {
		return handler(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.Name)
}

// handlerLogin is a command handler for the "login" command
// It updates the current user in the configuration file
// and writes the updated configuration back to disk
// It takes a pointer to the state and a command as arguments
// and returns an error if any issues occur during the process
func handlerLogin(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <username>", cmd.Name)
	}

	// Set the current user in the config
	s.Config.CurrentUserName = cmd.Args[0]

	// Get home directory
	homeDir, err := config.GetHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	// Write the updated config with the new username to the home directory
	err = config.Write(homeDir, *s.Config)
	if err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	fmt.Printf("Logged in as %s\n", s.Config.CurrentUserName)
	return nil
}
