package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/Specter242/Gator/internal/config"
	"github.com/Specter242/Gator/internal/database"
)

type state struct {
	db     *database.Queries
	Config *config.Config
}

type command struct {
	Name string
	Args []string
}

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

func (c *commands) register(name string, f func(*state, command) error) {
	if c.Commandmap == nil {
		c.Commandmap = make(map[string]func(*state, command) error)
	}
	c.Commandmap[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	if handler, ok := c.Commandmap[cmd.Name]; ok {
		return handler(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.Name)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <username>", cmd.Name)
	}
	username := cmd.Args[0]
	ctx := context.Background()
	_, err := s.db.GetUser(ctx, username)
	if err != nil {
		return fmt.Errorf("user not found: %s", username)
	}
	s.Config.CurrentUserName = cmd.Args[0]
	homeDir, err := config.GetHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}
	err = config.Write(homeDir, *s.Config)
	if err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}
	fmt.Printf("Logged in as %s\n", s.Config.CurrentUserName)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <username>", cmd.Name)
	}
	username := cmd.Args[0]
	ctx := context.Background()
	_, err := s.db.CreateUser(ctx, username)
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}
	s.Config.CurrentUserName = username
	homeDir, err := config.GetHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}
	err = config.Write(homeDir, *s.Config)
	if err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}
	fmt.Printf("User %s registered and logged in\n", username)
	return nil
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}
	ctx := context.Background()
	err := s.db.Reset(ctx)
	if err != nil {
		return fmt.Errorf("error deleting all users: %v", err)
	}
	fmt.Printf("All users deleted\n")
	return nil
}

func handlerGetUsers(s *state, cmd command) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}
	ctx := context.Background()
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
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <time_between_requests>", cmd.Name)
	}
	tick, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid time format: %v", err)
	}
	println("Collecting feeds every %s\n", tick)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for ; ; <-ticker.C {
		handlerScrapeFeeds(s, command{Name: "scrapefeeds"})
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("usage: %s <name> <feed_url>", cmd.Name)
	}
	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]
	ctx := context.Background()
	feed, err := fetchFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}
	feedRow, err := s.db.CreateFeed(ctx, database.CreateFeedParams{
		Name:   feedName,
		Url:    feedURL,
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed: %v", err)
	}
	_, err = s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feedRow.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed follow: %v", err)
	}
	fmt.Printf("Successfully fetched and processed feed: %s\n", feed.Channel.Title)
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}
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

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <url>", cmd.Name)
	}
	feedURL := cmd.Args[0]
	ctx := context.Background()
	_, err := fetchFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}
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

func handlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}
	ctx := context.Background()
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

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <url>", cmd.Name)
	}
	feedURL := cmd.Args[0]
	ctx := context.Background()
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
		return fmt.Errorf("feed not found in database")
	}
	err = s.db.RemoveFeedFollow(ctx, database.RemoveFeedFollowParams{
		UserID: user.ID,
		FeedID: dbFeed.ID,
	})
	if err != nil {
		return fmt.Errorf("error deleting feed follow: %v", err)
	}
	fmt.Printf("%s successfully unfollowed feed: %s\n", user.Name, dbFeed.Name)
	return nil
}

func handlerHelp(s *state, cmd command) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}
	fmt.Println("Available commands:")
	fmt.Println("  login <username> - Log in as the specified user")
	fmt.Println("  register <username> - Register a new user")
	fmt.Println("  reset - Reset the database")
	fmt.Println("  users - Get all users")
	fmt.Println("  agg - run aggregator service")
	fmt.Println("  addfeed <name> <url> - Add a new feed with the specified name and URL")
	fmt.Println("  feeds - List all feeds")
	fmt.Println("  follow <url> - Follow a feed by URL")
	fmt.Println("  following - List all followed feeds")
	fmt.Println("  unfollow <url> - Unfollow a feed by URL")
	fmt.Println("  help - Show this help message")
	return nil
}

func handlerScrapeFeeds(s *state, cmd command) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("usage: %s", cmd.Name)
	}
	next, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("error getting next feed to fetch: %v", err)
	}
	err = s.db.LastFetchedAt(context.Background(), next.ID)
	if err != nil {
		return fmt.Errorf("error marking feed as fetched: %v", err)
	}
	fetch, err := fetchFeed(context.Background(), next.Url)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}
	for _, item := range fetch.Channel.Item {
		var pubTime time.Time
		var parseErr error
		layouts := []string{
			time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822,
		}
		for _, layout := range layouts {
			pubTime, parseErr = time.Parse(layout, item.PubDate)
			if parseErr == nil {
				break
			}
		}
		if parseErr != nil {
			return fmt.Errorf("error parsing pubDate %q: %v", item.PubDate, parseErr)
		}
		_, err := s.db.CreatePost(context.Background(), database.CreatePostParams{
			FeedID:      next.ID,
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: pubTime,
		})
		if err != nil {
			return fmt.Errorf("error creating post: %v", err)
		}
		fmt.Printf("Post created: %s\n", item.Title)
	}
	return nil
}

func handlerBrowse(s *state, cmd command) error {
	limit := 2
	if len(cmd.Args) > 1 {
		return fmt.Errorf("usage: %s [limit]", cmd.Name)
	}
	if len(cmd.Args) == 1 {
		var err error
		_, err = fmt.Sscanf(cmd.Args[0], "%d", &limit)
		if err != nil || limit <= 0 {
			return fmt.Errorf("invalid limit: %s", cmd.Args[0])
		}
	}
	ctx := context.Background()
	// Get current user
	if s.Config.CurrentUserName == "" {
		return fmt.Errorf("not logged in")
	}
	user, err := s.db.GetUser(ctx, s.Config.CurrentUserName)
	if err != nil {
		return fmt.Errorf("could not get user: %v", err)
	}
	posts, err := s.db.GetPostsForUser(ctx, database.GetPostsForUserParams{
		ID:     user.ID,
		Limit:  int32(limit),
		Offset: 0,
	})
	if err != nil {
		return fmt.Errorf("error getting posts: %v", err)
	}
	if len(posts) == 0 {
		fmt.Println("No posts found.")
		return nil
	}
	for _, post := range posts {
		fmt.Printf("- %s (%s) %s\n", post.Title, post.Url, post.PublishedAt)
	}
	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
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
