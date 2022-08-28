package main

import (
	"flag"
	"os"

	"github.com/appditto/natrium-wallet-server/controller"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/utils"
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

	// Setup RPC Client
	nanoRpcUrl := utils.GetEnv("NANO_RPC_URL", "http://localhost:7076")
	rpcClient := net.RPCClient{
		Url: nanoRpcUrl,
	}

	// Setup controllers
	wsc := controller.WsController{RPCClient: &rpcClient}

	// HTTP Routes
	app.Post("/api", controller.HandleAction)
	app.Post("/callback", controller.HandleHTTPCallback)

	// Websocket upgrade
	// HTTP/WS Routes
	app.Use("/", func(c *fiber.Ctx) error {
		// Get IP Address
		headers := c.GetReqHeaders()
		var ipAddr string
		if val, ok := headers["X-Real-Ip"]; ok {
			ipAddr = val
		} else if val, ok := headers["X-Forwarded-For"]; ok {
			ipAddr = val
		} else {
			ipAddr = c.IP()
		}

		c.Locals("ip", ipAddr)
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/", websocket.New(wsc.HandleWSMessage))

	// Cors middleware
	app.Use(cors.New())

	// 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404)
	})

	app.Listen(":3000")
}
