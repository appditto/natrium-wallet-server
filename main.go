package main

import (
	"flag"
	"os"

	"github.com/appditto/natrium-wallet-server/controller"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

func init() {
	flag.Usage = usage
	flag.Set("logtostderr", "true")
	// TODO - this forces specific log levels, might not wanna do that
	flag.Set("stderrthreshold", "INFO")
	flag.Set("v", "2")
	// This is wa
	flag.Parse()
}

func main() {
	// Create app
	app := fiber.New()

	// HTTP Routes
	app.Post("/api", controller.HandleAction)
	app.Post("/callback", controller.HandleHTTPCallback)

	// Websocket upgrade
	// HTTP/WS Routes
	app.Use("/", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/", websocket.New(controller.HandleWSMessage))

	// Cors middleware
	app.Use(cors.New())

	// 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404)
	})

	app.Listen(":3000")
}
