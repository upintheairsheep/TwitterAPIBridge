package twitterv1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func InitServer() {
	app := fiber.New()

	// Initialize default config
	app.Use(logger.New())

	// Custom middleware to log request details
	app.Use(func(c *fiber.Ctx) error {
		// fmt.Println("Request Method:", c.Method())
		// fmt.Println("Request URL:", c.OriginalURL())
		// fmt.Println("Post Body:", string(c.Body()))
		// fmt.Println("Headers:", string(c.Request().Header.Header()))
		// fmt.Println()
		return c.Next()
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Auth
	app.Post("/oauth/access_token", access_token)

	// Tweeting
	app.Post("/1/statuses/update.json", status_update)

	// Interactions
	app.Post("/1/statuses/retweet/:id.json", retweet)
	app.Post("/1/favorites/create/:id.json", favourite)
	app.Post("/1/favorites/destroy/:id.json", Unfavourite)
	app.Post("/1/statuses/destroy/:id.json", DeleteTweet)

	// Posts
	app.Get("/1/statuses/home_timeline.json", home_timeline)
	app.Get("/1/statuses/user_timeline.json", user_timeline)
	app.Get("/1/statuses/show/:id.json", GetStatusFromId)
	app.Get("/i/statuses/:id/activity/summary.json", TweetInfo)
	app.Get("/1/related_results/show/:id.json", RelatedResults)

	// Users
	app.Get("/1/users/show.xml", user_info)
	app.Get("/1/users/lookup.json", UsersLookup)
	app.Post("/1/users/lookup.json", UsersLookup)
	app.Get("/1/friendships/lookup.xml", UserRelationships)
	app.Get("/1/friendships/show.xml", GetUsersRelationship)

	// Connect
	app.Get("/1/users/search.json", UserSearch)

	// Trends
	app.Get("/1/trends/:woeid.json", trends_woeid)

	// Setings
	app.Get("/1/account/settings.xml", GetSettings)
	app.Get("/1/account/push_destinations/device.xml", PushDestinations)

	// Legal cuz why not?
	app.Get("/1/legal/tos.json", TOS)
	app.Get("/1/legal/privacy.json", PrivacyPolicy)

	// CDN Downscaler
	app.Get("/cdn/img", CDNDownscaler)

	app.Listen(":3000")
}
