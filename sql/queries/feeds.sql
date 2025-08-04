-- name: AddFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeeds :many
SELECT f.name
    , f.url
    , u.name AS user_name
FROM feeds f
    JOIN users u
        ON f.user_id = u.id
;

-- name: AddFollowFeed :one
WITH inserted_feed_follow AS (
    INSERT INTO follow_feeds (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)

SELECT inserted_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
    JOIN feeds
        ON inserted_feed_follow.feed_id = feeds.id
    JOIN users
        ON inserted_feed_follow.user_id = users.id
;

-- name: GetFeedByUrl :one
SELECT f.id
FROM feeds f
WHERE f.url = $1
;

-- name: GetFeedFollowsForUser :many
SELECT f.name
FROM follow_feeds ff
    JOIN feeds f
        ON ff.feed_id = f.id
WHERE ff.user_id = $1
;

-- name: DeleteFollowFeed :exec
DELETE FROM follow_feeds
WHERE user_id = $1
    AND feed_id = $2
;

-- name: MarkFeedFetched :exec
UPDATE feeds
SET updated_at = $1
    , last_fetched_at = $1
WHERE id = $2
;

-- name: GetNextFeedToFetch :one
SELECT id
    , name
    , url
FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1
;