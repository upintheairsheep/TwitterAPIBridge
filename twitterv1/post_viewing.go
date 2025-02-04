package twitterv1

import (
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"time"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/bridge"
	"github.com/Preloading/MastodonTwitterAPI/db_controller"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508224719/https://dev.twitter.com/docs/api/1/get/statuses/home_timeline
func home_timeline(c *fiber.Ctx) error {
	// Get all of our keys, beeps, and bops
	user_did, session_uuid, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	encryptionKey, err := GetEncryptionKeyFromRequest(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	// Check for context
	max_id := c.Query("max_id")
	context := ""

	// Handle getting things in the past
	if max_id != "" {
		// Get the timeline context from the DB
		maxIDBigInt, ok := new(big.Int).SetString(max_id, 10)
		if !ok {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid max_id format")
		}
		maxID, _, _ := bridge.TwitterMsgIdToBluesky(maxIDBigInt)
		fmt.Println("Max ID: " + maxID)
		contextPtr, err := db_controller.GetTimelineContext(*user_did, *session_uuid, *maxIDBigInt, *encryptionKey)
		if err == nil {
			context = *contextPtr
		}
	}

	err, res := blueskyapi.GetTimeline(*oauthToken, context)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch timeline")
	}

	// Translate the posts to tweets
	tweets := []bridge.Tweet{}

	for _, item := range res.Feed {
		tweets = append(tweets, TranslatePostToTweet(item.Post, item.Reply.Parent.URI, item.Reply.Parent.Author.DID, &item.Reply.Parent.Record.CreatedAt, item.Reason))
	}

	// Store the oldest message id, along with our context in the DB
	oldestTweet := tweets[0]
	for _, tweet := range tweets {

		// TODO: Remove this later once retweets are working better
		if tweet.RetweetedStatus != nil {
			continue
		}

		tweetTime, err := bridge.TwitterTimeParser(tweet.CreatedAt)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to parse tweet time")
		}
		oldestTweetTime, err := bridge.TwitterTimeParser(oldestTweet.CreatedAt)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to parse oldest tweet time")
		}
		if tweetTime.Before(oldestTweetTime) {
			oldestTweet = tweet
		}
	}
	err = db_controller.SetTimelineContext(*user_did, *session_uuid, oldestTweet.ID, res.Cursor, *encryptionKey)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save timeline context")
	}

	return c.JSON(tweets)

}

// https://web.archive.org/web/20120708204036/https://dev.twitter.com/docs/api/1/get/statuses/show/%3Aid
func GetStatusFromId(c *fiber.Ctx) error {
	encodedId := c.Params("id")
	idBigInt, ok := new(big.Int).SetString(encodedId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	uri, _, _ := bridge.TwitterMsgIdToBluesky(idBigInt) // TODO: maybe look up with the retweet? idk

	_, _, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header")
	}

	err, thread := blueskyapi.GetPost(*oauthToken, uri, 0, 1)

	if err != nil {
		return err
	}

	return c.JSON(TranslatePostToTweet(thread.Thread.Post, "", "", nil, nil))
}

func TranslatePostToTweet(tweet blueskyapi.Post, replyMsgBskyURI string, replyUserBskyId string, replyTimeStamp *time.Time, postReason *blueskyapi.PostReason) bridge.Tweet {
	tweetEntities := bridge.Entities{
		Hashtags:     nil,
		Urls:         nil,
		UserMentions: []bridge.UserMention{},
		Media:        []bridge.Media{},
	}

	id := 1
	for _, image := range tweet.Record.Embed.Images {
		// Process each image
		// fmt.Println("Image:", "http://10.0.0.77:3000/cdn/img/?url="+url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:"+item.Post.Author.DID+"/"+image.Image.Ref.Link+"/@jpeg"))
		tweetEntities.Media = append(tweetEntities.Media, bridge.Media{
			ID:       *big.NewInt(int64(id)),
			IDStr:    strconv.Itoa(id),
			MediaURL: "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/"+tweet.Author.DID+"/"+image.Image.Ref.Link+"/@jpeg"),
			// MediaURLHttps: "https://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape("https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:"+image.Image.Ref.Link+"@jpeg"),
		})
		id++
	}
	for _, faucet := range tweet.Record.Facets {
		// I haven't seen this exceed 1 element yet
		// if len(faucet.Features) > 1 {
		// fmt.Println("Faucet with more than 1 feature found!")
		// faucetJSON, err := json.Marshal(faucet)
		// if err != nil {
		// 	fmt.Println("Error encoding faucet to JSON:", err)
		// } else {
		// 	fmt.Println("Faucet JSON:", string(faucetJSON))
		// }
		// // }
		// fmt.Println(faucet.Features[0].Type)
		switch faucet.Features[0].Type {
		case "app.bsky.richtext.facet#mention":
			tweetEntities.UserMentions = append(tweetEntities.UserMentions, bridge.UserMention{
				Name:       "test",
				ScreenName: "test",
				//ScreenName: item.Post.Record.Text[faucet.Index.ByteStart+1 : faucet.Index.ByteEnd],
				ID: *bridge.BlueSkyToTwitterID(faucet.Features[0].Did),
				Indices: []int{
					faucet.Index.ByteStart,
					faucet.Index.ByteEnd,
				},
			})
		}

	}

	bsky_retweet_author := tweet.Author

	isRetweet := false
	// Checking if this tweet is a retweet
	if postReason != nil {
		// This might contain other things in the future, idk
		if postReason.Type == "app.bsky.feed.defs#reasonRepost" {
			// We are a retweet.
			isRetweet = true
			tweet.Author = postReason.By
		}
	}

	convertedTweet := bridge.Tweet{
		Coordinates: nil,
		Favourited:  tweet.Viewer.Like != nil,
		CreatedAt: func() string {
			if isRetweet {
				return bridge.TwitterTimeConverter(postReason.IndexedAt)
			}
			return bridge.TwitterTimeConverter(tweet.Record.CreatedAt)
		}(),
		Truncated:    false,
		Text:         tweet.Record.Text,
		Entities:     tweetEntities,
		Annotations:  nil, // I am curious what annotations are
		Contributors: nil,
		ID: func() big.Int {
			// we have to use psudo ids because of https://github.com/bluesky-social/atproto/issues/1811
			if isRetweet {
				return bridge.BskyMsgToTwitterID(tweet.URI, postReason.IndexedAt, &postReason.By.DID)
			}
			return bridge.BskyMsgToTwitterID(tweet.URI, tweet.Record.CreatedAt, nil)
		}(),
		IDStr: func() string {
			if isRetweet {
				id := bridge.BskyMsgToTwitterID(tweet.URI, postReason.IndexedAt, &postReason.By.DID)
				return id.String()
			}
			id := bridge.BskyMsgToTwitterID(tweet.URI, tweet.Record.CreatedAt, nil)
			return id.String()
		}(),
		Retweeted:         tweet.Viewer.Repost != nil,
		RetweetCount:      tweet.RepostCount,
		Geo:               nil,
		Place:             nil,
		PossiblySensitive: false,
		InReplyToUserID: func() *big.Int {
			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			if id.Cmp(big.NewInt(0)) == 0 {
				return nil
			}
			return id
		}(),
		InReplyToUserIDStr: func() *string {
			id := bridge.BlueSkyToTwitterID(replyUserBskyId)
			if id.Cmp(big.NewInt(0)) == 0 {
				return nil
			}
			idStr := id.String()
			return &idStr
		}(),
		InReplyToScreenName: &tweet.Author.DisplayName,
		User: bridge.TwitterUser{
			Name: func() string {
				if tweet.Author.DisplayName == "" {
					return tweet.Author.Handle
				}
				return tweet.Author.DisplayName
			}(),
			ProfileSidebarBorderColor: "eeeeee",
			ProfileBackgroundTile:     false,
			ProfileSidebarFillColor:   "efefef",
			CreatedAt:                 bridge.TwitterTimeConverter(tweet.Author.Associated.CreatedAt),
			ProfileImageURL:           "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + "&width=128&height=128",
			// ProfileImageURLHttps:           "https://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(tweet.Author.Avatar) + "&width=128&height=128",
			Location:            "Twitter",
			ProfileLinkColor:    "009999",
			FollowRequestSent:   false,
			URL:                 "",
			ScreenName:          tweet.Author.Handle,
			ContributorsEnabled: false,
			UtcOffset:           nil,
			IsTranslator:        false,
			ID:                  *bridge.BlueSkyToTwitterID(tweet.URI),
			// IDStr:                          bridge.BlueSkyToTwitterID(tweet.URI).String(),
			ProfileUseBackgroundImage: false,
			ProfileTextColor:          "333333",
			Protected:                 false,
			Lang:                      "en",
			Notifications:             nil,
			TimeZone:                  nil,
			Verified:                  false,
			ProfileBackgroundColor:    "C0DEED",
			GeoEnabled:                true,
			Description:               "",
			ProfileBackgroundImageURL: "http://a0.twimg.com/images/themes/theme1/bg.png",
			// ProfileBackgroundImageURLHttps: "http://a0.twimg.com/images/themes/theme1/bg.png",
			Following: nil,

			// huh
			DefaultProfile:      false,
			DefaultProfileImage: false,
			ShowAllInlineMedia:  false,

			// User Stats
			// ListedCount:     0,
			// FavouritesCount: 0,
			// FollowersCount:  200,
			// FriendsCount:    100,
			// StatusesCount:   333,
		},
		Source: "Bluesky",
		InReplyToStatusID: func() *big.Int {
			if replyTimeStamp == nil {
				return nil
			}
			id := bridge.BskyMsgToTwitterID(replyMsgBskyURI, *replyTimeStamp, &replyUserBskyId)
			return &id
		}(),
		InReplyToStatusIDStr: func() *string {
			if replyTimeStamp == nil {
				return nil
			}
			id := bridge.BskyMsgToTwitterID(replyMsgBskyURI, *replyTimeStamp, &replyUserBskyId)
			idStr := id.String()
			return &idStr
		}(),
		RetweetedStatus: func() *bridge.Tweet {
			if isRetweet {
				retweet_bsky := tweet
				retweet_bsky.Author = bsky_retweet_author
				translatedTweet := TranslatePostToTweet(retweet_bsky, replyMsgBskyURI, replyUserBskyId, replyTimeStamp, nil)
				return &translatedTweet
			}
			return nil
		}(),
	}
	return convertedTweet
}

// This request is an "internal" request, and thus, these are very little to no docs. this is a problem.
// The most docs I could find: https://blog.fgribreau.com/2012/01/twitter-unofficial-api-getting-tweets.html
func TweetInfo(c *fiber.Ctx) error {
	_, _, oauthToken, err := GetAuthFromReq(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("OAuth token not found in Authorization header (yes i know this isn't complient with the twitter api)")
	}

	encodedId := c.Params("id")
	idBigInt, ok := new(big.Int).SetString(encodedId, 10)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID format")
	}
	id, _, _ := bridge.TwitterMsgIdToBluesky(idBigInt)

	err, thread := blueskyapi.GetPost(*oauthToken, id, 1, 0)

	if err != nil {
		return err
	}

	likes, err := blueskyapi.GetLikes(*oauthToken, id, 100)

	if err != nil {
		return err
	}

	reposters, err := blueskyapi.GetRetweetAuthors(*oauthToken, id, 100)

	if err != nil {
		return err
	}

	repliers := []big.Int{}
	favourites := []big.Int{}
	retweeters := []big.Int{}

	for _, reply := range thread.Thread.Replies {
		repliers = append(repliers, *bridge.BlueSkyToTwitterID(reply.Author.DID))
	}
	for _, like := range likes.Likes {
		favourites = append(favourites, *bridge.BlueSkyToTwitterID(like.Actor.DID))
	}
	for _, reposter := range reposters.RepostedBy {
		retweeters = append(retweeters, *bridge.BlueSkyToTwitterID(reposter.DID))
	}

	return c.JSON(bridge.TwitterActivitiySummary{
		FavouritesCount: thread.Thread.Post.LikeCount,
		RetweetsCount:   thread.Thread.Post.RepostCount,
		RepliersCount:   thread.Thread.Post.ReplyCount,
		Favourites:      favourites,
		Retweets:        retweeters,
		Repliers:        repliers,
	})
}
