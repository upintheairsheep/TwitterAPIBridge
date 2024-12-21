package twitterv1

import (
	"fmt"
	"math/big"
	"time"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/post/statuses/update
func status_update(c *fiber.Ctx) error {
	my_did, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	status := c.FormValue("status")
	trim_user := c.FormValue("trim_user")
	in_reply_to_status_id := c.FormValue("in_reply_to_status_id")

	fmt.Println("Status:", status)
	fmt.Println("TrimUser:", trim_user)
	fmt.Println("InReplyToStatusID:", in_reply_to_status_id)

	thread, err := blueskyapi.UpdateStatus(*oauthToken, *my_did, status, &in_reply_to_status_id)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	if thread.Thread.Parent == nil {
		return c.JSON(TranslatePostToTweet(thread.Thread.Post, "", "", nil, nil))
	} else {
		return c.JSON(TranslatePostToTweet(thread.Thread.Post, thread.Thread.Parent.URI, thread.Thread.Parent.Author.DID, &thread.Thread.Parent.Record.CreatedAt, nil))
	}
}

// https://web.archive.org/web/20120407091252/https://dev.twitter.com/docs/api/1/post/statuses/retweet/%3Aid
func retweet(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Get our IDs
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr

	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}

	err, originalPost, retweetPostURI := blueskyapi.ReTweet(*oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update status")
	}

	retweet := TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Post.URI, originalPost.Thread.Parent.Author.DID, &originalPost.Thread.Parent.Record.CreatedAt, nil)
	retweet.Retweeted = true
	now := time.Now() // pain, also fix this to use the proper timestamp according to the server.
	retweet.ID = bridge.BskyMsgToTwitterID(*retweetPostURI, &now, user_did)
	retweet.IDStr = retweet.ID.String()

	return c.JSON(bridge.Retweet{
		Tweet:           retweet,
		RetweetedStatus: TranslatePostToTweet(originalPost.Thread.Post, originalPost.Thread.Post.URI, originalPost.Thread.Parent.Author.DID, &originalPost.Thread.Parent.Record.CreatedAt, nil), // TODO: make this respond with proper retweet data
	})
}

// https://web.archive.org/web/20120412065707/https://dev.twitter.com/docs/api/1/post/favorites/create/%3Aid
func favourite(c *fiber.Ctx) error {
	postId := c.Params("id")
	user_did, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Fetch ID
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}

	fmt.Println("Post ID:", postId)

	err, post := blueskyapi.LikePost(*oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to like post")
	}

	newTweet := TranslatePostToTweet(post.Thread.Post, post.Thread.Post.URI, post.Thread.Parent.Author.DID, &post.Thread.Parent.Record.CreatedAt, nil)

	return c.JSON(newTweet)
}

// https://web.archive.org/web/20120412041153/https://dev.twitter.com/docs/api/1/post/favorites/destroy/%3Aid
func Unfavourite(c *fiber.Ctx) error { // yes i am canadian
	postId := c.Params("id")
	user_did, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Fetch ID
	idBigInt, ok := new(big.Int).SetString(postId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postIdPtr, _, _, err := bridge.TwitterMsgIdToBluesky(idBigInt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	postId = *postIdPtr
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}

	err, post := blueskyapi.UnlikePost(*oauthToken, postId, *user_did)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to unlike post")
	}

	newTweet := TranslatePostToTweet(post.Thread.Post, post.Thread.Post.URI, post.Thread.Parent.Author.DID, &post.Thread.Parent.Record.CreatedAt, nil)

	return c.JSON(newTweet)
}
