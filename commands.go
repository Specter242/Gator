package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/Specter242/Gator/internal/config"
	"github.com/Specter242/Gator/internal/database"
)

// State holds a pointer to a config.Config struct
type state struct {
	db     *database.Queries
	Config *config.Config
}
type command struct {
	Name string
	Args []string
}

// commands is a map of command names to their respective handler functions
type commands struct {
	Commandmap map[string]func(*state, command) error
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
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

	// Get the username from the command arguments
	username := cmd.Args[0]

	// Check if the user exists in the database
	ctx := context.Background()
	_, err := s.db.GetUser(ctx, username)
	if err != nil {
		return fmt.Errorf("user not found: %s", username)
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

// handlerRegister is a command handler for the "register" command
// It creates a new user and stores it in the database
// It takes a pointer to the state and a command as arguments
// and returns an error if any issues occur during the process
func handlerRegister(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <username>", cmd.Name)
	}

	// Get the username from the command arguments
	username := cmd.Args[0]

	// Use the database to create a new user
	// We need to add context for the database operation
	ctx := context.Background()

	// Create the user in the database
	_, err := s.db.CreateUser(ctx, username)
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}

	// Set the current user in the config
	s.Config.CurrentUserName = username

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

	fmt.Printf("User %s registered and logged in\n", username)
	return nil
}

// handlerReset is a command handler for the "reset" command
// it deletes all users from the database
func handlerReset(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}

	// Use the database to delete all users
	ctx := context.Background()

	// Delete all users from the database
	err := s.db.Reset(ctx)
	if err != nil {
		return fmt.Errorf("error deleting all users: %v", err)
	}

	fmt.Printf("All users deleted\n")
	return nil
}

// handlerGetUsers is a command handler for the "getusers" command
// It retrieves all users from the database
func handlerGetUsers(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}

	// Use the database to get all users
	ctx := context.Background()

	// Get all users from the database
	users, err := s.db.GetUsers(ctx, database.GetUsersParams{
		Limit:  100,
		Offset: 0,
	})

	if err != nil {
		return fmt.Errorf("error getting all users: %v", err)
	}

	fmt.Printf("All users:")
	for _, user := range users {
		fmt.Printf("\n- %s", user.Name)
		if user.Name == s.Config.CurrentUserName {
			fmt.Printf(" (current)")
		}
	}
	fmt.Printf("\n")
	return nil
}

func handlerFetchFeed(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s <feed_url>", cmd.Name)
	}

	// Get the feed URL from the command arguments
	feedURL := "https://www.wagslane.dev/index.xml"

	// Fetch the RSS feed
	ctx := context.Background()
	feed, err := fetchFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}

	// Print the feed title and description
	fmt.Printf("Feed Title: %s\n", feed.Channel.Title)
	fmt.Printf("Feed Description: %s\n", feed.Channel.Description)

	// Print each item in the feed
	for _, item := range feed.Channel.Item {
		fmt.Printf("\nItem Title: %s\n", item.Title)
		fmt.Printf("Item Link: %s\n", item.Link)
		fmt.Printf("Item Description: %s\n", item.Description)
		fmt.Printf("Item PubDate: %s\n", item.PubDate)
	}

	return nil
}

func handlerAddFeed(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 2 {
		return fmt.Errorf("usage: %s <name> <feed_url>", cmd.Name)
	}

	// Get the name and feed URL from the command arguments
	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	// Get current user from the config and attach it to the feed
	currentUser := s.Config.CurrentUserName
	if currentUser == "" {
		return fmt.Errorf("no user logged in, please log in first")
	}

	// Fetch the user ID from the database
	ctx := context.Background()
	user, err := s.db.GetUser(ctx, currentUser)
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}

	// Fetch the RSS feed
	feed, err := fetchFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}

	// Create the feed in the database
	feedRow, err := s.db.CreateFeed(ctx, database.CreateFeedParams{
		Name:   feedName,
		Url:    feedURL,
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed: %v", err)
	}

	// add to feed_follows
	_, err = s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feedRow.ID, // <-- Use the feed's ID here!
	})
	if err != nil {
		return fmt.Errorf("error creating feed follow: %v", err)
	}

	fmt.Printf("Successfully fetched and processed feed: %s\n", feed.Channel.Title)

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}

	// Get all feeds
	feeds, err := s.db.GetFeeds(context.Background(), database.GetFeedsParams{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		return fmt.Errorf("error getting feeds: %v", err)
	}

	for _, feed := range feeds {
		ctx := context.Background()
		user, err := s.db.GetUserById(ctx, feed.UserID)
		if err != nil {
			return fmt.Errorf("error getting user for feed %s: %v", feed.Name, err)
		}
		name := user.Name
		fmt.Printf("- %s (%s) %s\n", feed.Name, feed.Url, name)
	}
	return nil
}

func handlerFollow(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <url>", cmd.Name)
	}

	feedURL := cmd.Args[0]
	currentUser := s.Config.CurrentUserName
	if currentUser == "" {
		return fmt.Errorf("no user logged in, please log in first")
	}

	ctx := context.Background()
	user, err := s.db.GetUser(ctx, currentUser)
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}

	// Fetch the RSS feed (for validation, optional)
	_, err = fetchFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}

	// Find the feed in the database (or create it if it doesn't exist)
	feeds, err := s.db.GetFeeds(ctx, database.GetFeedsParams{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		return fmt.Errorf("error getting feeds: %v", err)
	}

	var dbFeed *database.Feed
	for _, f := range feeds {
		if f.Url == feedURL {
			dbFeed = &f
			break
		}
	}
	if dbFeed == nil {
		return fmt.Errorf("feed not found in database, please add it first")
	}

	// Create the feed follow entry
	_, err = s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: dbFeed.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed follow: %v", err)
	}
	fmt.Printf("%s successfully followed feed: %s\n", user.Name, dbFeed.Name)
	return nil
}

func handlerFollowing(s *state, cmd command) error {
	// Check if the command has the correct number of arguments
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}

	currentUser := s.Config.CurrentUserName
	if currentUser == "" {
		return fmt.Errorf("no user logged in, please log in first")
	}

	ctx := context.Background()
	user, err := s.db.GetUser(ctx, currentUser)
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}
	userid := database.GetFeedFollowsForUserParams{
		UserID: user.ID,
		Limit:  100,
		Offset: 0,
	}

	followedFeeds, err := s.db.GetFeedFollowsForUser(ctx, userid)
	if err != nil {
		return fmt.Errorf("error getting followed feeds: %v", err)
	}

	fmt.Printf("%s is following:\n", user.Name)
	for _, feed := range followedFeeds {
		fmt.Printf("- %s\n", feed.FeedName)
	}
	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Prevent redirects
		},
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", "Gator/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching feed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("error unmarshalling XML: %v", err)
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	return &feed, nil
}
