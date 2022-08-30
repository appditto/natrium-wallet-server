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
	"k8s.io/klog/v2"
)

func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	// Server options
	flag.Usage = usage
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	if utils.GetEnv("ENVIRONMENT", "development") == "development" {
		flag.Set("stderrthreshold", "INFO")
		flag.Set("v", "3")
	}
	bolivarPriceUpdate := flag.Bool("bolivar-price-update", false, "Update bolivar price")
	nanoPriceUpdate := flag.Bool("nano-price-update", false, "Update nano prices")
	bananoPriceUpdate := flag.Bool("banano-price-update", false, "Update banano prices")
	flag.Parse()

	// Price job
	if *bolivarPriceUpdate {
		err := net.UpdateDolarTodayPrice()
		if err != nil {
			klog.Errorf("Error updating dolar today price: %v", err)
			os.Exit(1)
		}
		err = net.UpdateDolarSiPrice()
		if err != nil {
			klog.Errorf("Error updating dolar si price: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	} else if *nanoPriceUpdate {
		err := net.UpdateNanoCoingeckoPrices()
		if err != nil {
			klog.Errorf("Error updating nano prices: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	} else if *bananoPriceUpdate {
		err := net.UpdateBananoCoingeckoPrices()
		if err != nil {
			klog.Errorf("Error updating banano prices: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

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
