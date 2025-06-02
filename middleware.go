package main

import (
	"context"
	"fmt"

	"github.com/Specter242/Gator/internal/database"
)

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		if s.Config.CurrentUserName == "" {
			return fmt.Errorf("you must be logged in to use this command")
		}
		user, err := s.db.GetUser(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("could not find user: %v", err)
		}
		return handler(s, cmd, user)
	}
}
