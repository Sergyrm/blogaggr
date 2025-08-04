# Gator - A Simple Blog Aggregator CLI

**Gator** is a lightweight CLI tool written in Go for aggregating and viewing blog posts from various RSS feeds. It allows you to subscribe to feeds, store them in a PostgreSQL database, and browse the latest content from the command line.

---

## Requirements

Before using Gator, make sure you have the following installed on your system:

- **Go** (version 1.18 or later): https://golang.org/dl/
- **PostgreSQL**: https://www.postgresql.org/download/

---

## Installation

You can install the Gator CLI using `go install`:

```bash
go install github.com/Sergyrm/blogaggr@latest

Create a file named .gatorconfig.json in your home directory or working directory and add the connection string for PostgreSQL:

```
{
  "db_url": "postgres://username:password@localhost:5432/gatordb?sslmode=disable"
}

## Usage

gator register <username>
gator login <username>
gator reset
gator users
gator agg <time_between_reqs>
gator addfeed <name> <url>
gator feeds
gator follow <feed_id>
gator following
gator unfollow <feed_id>
gator browse