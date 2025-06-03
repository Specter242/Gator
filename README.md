Gator

Gator is a command-line RSS feed aggregator and reader written in Go. It allows users to register, log in, add and follow feeds, browse posts, and manage their subscriptions using a PostgreSQL backend.

Features

-User registration and login
-Add new RSS feeds
-Follow and unfollow feeds
-Browse and list posts from followed feeds
-Scrape and aggregate new posts from feeds
-View all users and feeds
-Command-line interface

Project Structure
.
├── commands.go                # Command handlers and CLI logic
├── main.go                    # Application entry point
├── middleware.go              # Middleware for authentication
├── internal/
│   ├── config/                # Configuration management
│   │   └── config.go
│   └── database/              # Database models and queries (sqlc generated)
│       ├── db.go
│       ├── models.go
│       └── users.sql.go
├── sql/
│   ├── queries/               # SQL query definitions
│   │   └── users.sql
│   └── schema/                # Database schema migrations
│       ├── 001_users.sql
│       ├── 002_feeds.sql
│       ├── 003_follows.sql
│       ├── 004_last_fetched_at.sql
│       └── 005_posts.sql
├── go.mod
├── go.sum
└── .gitignore

Getting Started

Prerequisites

Go 1.18+
PostgreSQL

Setup

1.Clone the repository:

git clone https://github.com/Specter242/Gator.git
cd Gator

2.Set up the database:

Create a PostgreSQL database.
Apply the migrations in schema to your database.

3.Configure the application:

On first run, a .gatorconfig.json file will be created in your home directory.
Edit the db_url in .gatorconfig.json to point to your PostgreSQL database.

4.Build and run:

go build -o gator
./gator help

Usage

Run the CLI with one of the following commands:

login <username> - Log in as the specified user
register <username> - Register a new user
reset - Reset the database (delete all users)
users - List all users
addfeed <name> <url> - Add a new feed
feeds - List all feeds
follow <url> - Follow a feed by URL
following - List all followed feeds
unfollow <url> - Unfollow a feed by URL
scrapefeeds - Scrape all feeds for new posts
browse - Browse posts in the database
help - Show help message

Example:

./gator register alice
./gator login alice
./gator addfeed "Go Blog" "https://blog.golang.org/feed.atom"
./gator follow "https://blog.golang.org/feed.atom"
./gator browse

Development

SQL queries are defined in users.sql and compiled to Go code using sqlc.
Main application logic is in main.go and commands.go.
Configuration is managed in config.go.

License
MIT License

