package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/appditto/natrium-wallet-server/controller"
	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/appleboy/go-fcm"
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
	bananoMode := flag.Bool("banano", false, "Run in BANANO mode (Kalium)")
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

	// Setup database conn
	config := &database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASS"),
		User:     os.Getenv("DB_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   os.Getenv("DB_NAME"),
	}
	fmt.Println("🏡 Connecting to database...")
	db, err := database.NewConnection(config)
	if err != nil {
		panic(err)
	}

	fmt.Println("🦋 Running database migrations...")
	database.Migrate(db)

	// Create app
	app := fiber.New()

	// Setup RPC Client
	nanoRpcUrl := utils.GetEnv("RPC_URL", "http://localhost:7076")
	rpcClient := net.RPCClient{
		Url: nanoRpcUrl,
	}

	// Setup FCM client
	var fcmClient *fcm.Client
	fcmToken := utils.GetEnv("FCM_TOKEN", "")
	if fcmToken != "" {
		svc, err := fcm.NewClient(fcmToken)
		if err != nil {
			klog.Errorf("Error initating FCM client: %v", err)
			os.Exit(1)
		}
		fcmClient = svc
	}

	// Setup controllers
	pricePrefix := "nano"
	if *bananoMode {
		pricePrefix = "banano"
	}
	wsClientMap := controller.NewWSSubscriptions()
	wsc := controller.WsController{RPCClient: &rpcClient, PricePrefix: pricePrefix, WSClientMap: wsClientMap, BananoMode: *bananoMode, DB: db}
	hc := controller.HttpController{RPCClient: &rpcClient, BananoMode: *bananoMode, DB: db, WSClientMap: wsClientMap, FcmClient: fcmClient}

	// HTTP Routes
	app.Post("/api", hc.HandleAction)
	app.Post("/callback", hc.HandleHTTPCallback)

	// Websocket upgrade
	// HTTP/WS Routes
	app.Use("/", func(c *fiber.Ctx) error {
		// Get IP Address
		c.Locals("ip", utils.IPAddress(c))
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
