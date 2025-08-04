package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"strings"

	"github.com/Sergyrm/blogaggr/internal/database"
	"github.com/google/uuid"
)

func handlerAggregator(s *state, cmd command) error {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return fmt.Errorf("usage: %v <time_between_reqs>", cmd.Name)
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Printf("Collecting feeds every %s...", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)

	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("usage: %v <name> <url>", cmd.Name)
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	feedParams := database.AddFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      sql.NullString{String: name, Valid: true},
		Url:       url,
		UserID:    user.ID,
	}
	feed, err := s.db.AddFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("couldn't add feed: %w", err)
	}

	addFollowFeedParams := database.AddFollowFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}
	_, err = s.db.AddFollowFeed(context.Background(), addFollowFeedParams)
	if err != nil {
		return fmt.Errorf("couldn't add follow feed: %w", err)
	}

	printFeed(feed)

	return nil
}

func handlerGetFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
	}

	printAllFeeds(feeds)

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <feed_id>", cmd.Name)
	}

	feedUrl := cmd.Args[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), feedUrl)
	if err != nil {
		return fmt.Errorf("couldn't find feed: %w", err)
	}

	addFollowFeedParams := database.AddFollowFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed,
	}
	followFeed, err := s.db.AddFollowFeed(context.Background(), addFollowFeedParams)
	if err != nil {
		return fmt.Errorf("couldn't add follow feed: %w", err)
	}

	printNewFollowFeed(followFeed)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	following, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("couldn't get following feeds: %w", err)
	}

	printFollowing(following)

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <feed_id>", cmd.Name)
	}

	feedUrl := cmd.Args[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), feedUrl)
	if err != nil {
		return fmt.Errorf("couldn't find feed: %w", err)
	}

	deleteFollowFeedParams := database.DeleteFollowFeedParams{
		UserID: user.ID,
		FeedID: feed,
	}
	err = s.db.DeleteFollowFeed(context.Background(), deleteFollowFeedParams)
	if err != nil {
		return fmt.Errorf("couldn't unfollow feed: %w", err)
	}

	fmt.Printf("Successfully unfollowed feed with URL: %s\n", feedUrl)

	return nil
}

func scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get next feed to scrape: %w", err)
	}

	if nextFeed.Url == "" {
		fmt.Println("No feeds to scrape.")
		return nil
	}

	markFeedFetchedParams := database.MarkFeedFetchedParams{
		ID:        nextFeed.ID,
		UpdatedAt: time.Now(),
	}

	err = s.db.MarkFeedFetched(context.Background(), markFeedFetchedParams)
	if err != nil {
		return fmt.Errorf("couldn't update feed last fetched time: %w", err)
	}

	rssFeed, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return fmt.Errorf("couldn't fetch feed: %w", err)
	}

	for _, item := range rssFeed.Channel.Item {
		publishedAt := sql.NullTime{}
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			publishedAt = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			FeedID:    nextFeed.ID,
			Title:     item.Title,
			Description: sql.NullString{
				String: item.Description,
				Valid:  true,
			},
			Url:         item.Link,
			PublishedAt: publishedAt,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			fmt.Printf("Couldn't create post: %v", err)
			continue
		}
	}
	fmt.Printf("Feed %v collected, %v posts found", nextFeed.Name, len(rssFeed.Channel.Item))

	return nil
}

func printFeed(feed database.Feed) {
	fmt.Printf("Feed ID: %s\n", feed.ID)
	fmt.Printf("Name: %s\n", feed.Name.String)
	fmt.Printf("URL: %s\n", feed.Url)
	fmt.Printf("User ID: %s\n", feed.UserID)
	fmt.Printf("Created At: %s\n", feed.CreatedAt)
	fmt.Printf("Updated At: %s\n", feed.UpdatedAt)
}

func printAllFeeds(feeds []database.GetFeedsRow) {
	for _, feed := range feeds {
		fmt.Printf("Feed Name: %v\n", feed.Name.String)
		fmt.Printf("Feed URL: %v\n", feed.Url)
		fmt.Printf("Feed User Name: %v\n", feed.UserName)
		fmt.Println("-------------------------")
	}
}

func printNewFollowFeed(followFeed database.AddFollowFeedRow) {
	fmt.Printf("Feed Name: %s\n", followFeed.FeedName.String)
	fmt.Printf("User Name: %s\n", followFeed.UserName)
}

func printFollowing(following []sql.NullString) {
	if len(following) == 0 {
		fmt.Println("You are not following any feeds.")
		return
	}

	for _, follow := range following {
		fmt.Printf("Feed Name: %s\n", follow.String)
		fmt.Println("-------------------------")
	}
}