package twitterv1

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/Preloading/MastodonTwitterAPI/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	configData *config.Config
)

func InitServer(config *config.Config) {
	configData = config
	blueskyapi.InitConfig(configData)
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	// Initialize default config
	app.Use(logger.New())

	// Custom middleware to log request details
	if config.DeveloperMode {
		app.Use(func(c *fiber.Ctx) error {
			// fmt.Println("Request Method:", c.Method())
			fmt.Println("Request URL:", c.OriginalURL())
			// fmt.Println("Post Body:", string(c.Body()))
			// fmt.Println("Headers:", string(c.Request().Header.Header()))
			// fmt.Println()
			return c.Next()
		})
	}

	// app.Get("/", func(c *fiber.Ctx) error {
	// 	return c.SendString("Hello, World!")
	// Serve static files from the "static" folder
	app.Static("/", "./static")
	app.Static("/favicon.ico", "./static/favicon.ico")
	app.Static("/robots.txt", "./static/robots.txt")
	app.Static("/static", "./static")

	// Auth
	app.Post("/oauth/access_token", access_token)
	app.Get("/1/account/verify_credentials.:filetype", VerifyCredentials)

	// Tweeting
	app.Post("/1/statuses/update.:filetype", status_update)

	// Interactions
	app.Post("/1/statuses/retweet/:id.:filetype", retweet)
	app.Post("/1/favorites/create/:id.:filetype", favourite)
	app.Post("/1/favorites/destroy/:id.:filetype", Unfavourite)
	app.Post("/1/statuses/destroy/:id.:filetype", DeleteTweet)

	// Posts
	app.Get("/1/statuses/home_timeline.:filetype", home_timeline)
	app.Get("/1/statuses/user_timeline.:filetype", user_timeline)
	app.Get("/1/statuses/show/:id.:filetype", GetStatusFromId)
	app.Get("/i/statuses/:id/activity/summary.:filetype", TweetInfo)
	app.Get("/1/related_results/show/:id.:filetype", RelatedResults)

	// Users
	app.Get("/1/users/show.:filetype", user_info)
	app.Get("/1/users/lookup.:filetype", UsersLookup)
	app.Post("/1/users/lookup.:filetype", UsersLookup)
	app.Get("/1/friendships/lookup.:filetype", UserRelationships)
	app.Get("/1/friendships/show.:filetype", GetUsersRelationship)
	app.Get("/1/favorites/:id.:filetype", likes_timeline)
	app.Post("/1/friendships/create.:filetype", FollowUser)
	app.Post("/1/friendships/destroy.:filetype", UnfollowUserForm)
	app.Post("/1/friendships/destroy/:id.:filetype", UnfollowUserParams)

	app.Get("/1/statuses/followers.:filetype", GetFollowers)
	app.Get("/1/statuses/friends.:filetype", GetFollows)

	app.Get("/1/users/recommendations.:filetype", GetSuggestedUsers)
	app.Get("/1/users/profile_image", UserProfileImage)

	// Connect
	app.Get("/1/users/search.:filetype", UserSearch)
	app.Get("/i/activity/about_me.:filetype", GetMyActivity)

	// Discover
	app.Get("/1/trends/:woeid.:filetype", trends_woeid)
	app.Get("/i/search.:filetype", InternalSearch)

	// Account / Settings
	app.Post("/1/account/update_profile.:filetype", UpdateProfile)
	app.Post("/1/account/update_profile_image.:filetype", UpdateProfilePicture)
	app.Get("/1/account/settings.:filetype", GetSettings)
	app.Get("/1/account/push_destinations/device.:filetype", PushDestinations)

	// Legal cuz why not?
	app.Get("/1/legal/tos.:filetype", TOS)
	app.Get("/1/legal/privacy.:filetype", PrivacyPolicy)

	// CDN Downscaler
	app.Get("/cdn/img", CDNDownscaler)

	// misc
	app.Get("/mobile_client_api/decider/:path", MobileClientApiDecider)

	app.Listen(":3000")
}

// misc
func MobileClientApiDecider(c *fiber.Ctx) error {
	return c.SendString("")
}

func EncodeAndSend(c *fiber.Ctx, data interface{}) error {
	encodeType := c.Params("filetype")
	switch encodeType {
	case "xml":
		// Encode the data to XML
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		enc.Indent("", "  ")
		if err := enc.Encode(data); err != nil {
			fmt.Println("Error encoding XML:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode into XML!")
		}

		// Add custom XML header
		xmlContent := buf.Bytes()
		customHeader := []byte(`<?xml version="1.0" encoding="UTF-8"?>`)
		xmlContent = append(customHeader, xmlContent...)

		return c.SendString(string(xmlContent))
	case "json":
		encoded, err := json.Marshal(data)
		if err != nil {
			fmt.Println("Error:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode into json!")
		}
		return c.SendString(string(encoded))
	default:
		return c.Status(fiber.StatusInternalServerError).SendString("Invalid file type!")
	}

}
