//go:generate go run github.com/Khan/genqlient
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appditto/natrium-wallet-server/controller"
	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/gql"
	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/repository"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/appleboy/go-fcm"
	"github.com/go-co-op/gocron"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/websocket/v2"
	"k8s.io/klog/v2"
)

var Version = "dev"

func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	// Server options
	flag.Usage = usage
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "INFO")
	flag.Set("v", "3")
	// if utils.GetEnv("ENVIRONMENT", "development") == "development" {
	// 	flag.Set("stderrthreshold", "INFO")
	// 	flag.Set("v", "3")
	// }
	bolivarPriceUpdate := flag.Bool("bolivar-price-update", false, "Update bolivar price")
	nanoPriceUpdate := flag.Bool("nano-price-update", false, "Update nano prices")
	bananoPriceUpdate := flag.Bool("banano-price-update", false, "Update banano prices")
	bananoMode := flag.Bool("banano", false, "Run in BANANO mode (Kalium)")
	version := flag.Bool("version", false, "Display the version")
	flag.Parse()

	if *version {
		fmt.Printf("Natrium server version: %s\n", Version)
		os.Exit(0)
	}

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
		// Alse VES and ARS first
		err := net.UpdateDolarTodayPrice()
		if err != nil {
			klog.Errorf("Error updating dolar today price: %v", err)
			// Not worth breaking the whole flow for VES
		}
		err = net.UpdateDolarSiPrice()
		if err != nil {
			klog.Errorf("Error updating dolar today price: %v", err)
			// Not worth breaking the whole flow for VES
		}
		err = net.UpdateNanoCoingeckoPrices()
		if err != nil {
			klog.Errorf("Error updating nano prices: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	} else if *bananoPriceUpdate {
		err := net.UpdateDolarTodayPrice()
		if err != nil {
			klog.Errorf("Error updating dolar today price: %v", err)
			// Not worth breaking the whole flow for VES
		}
		err = net.UpdateDolarSiPrice()
		if err != nil {
			klog.Errorf("Error updating dolar today price: %v", err)
			// Not worth breaking the whole flow for VES
		}
		err = net.UpdateNanoCoingeckoPrices()
		if err != nil {
			klog.Errorf("Error updating nano prices: %v", err)
			os.Exit(1)
		}
		err = net.UpdateBananoCoingeckoPrices()
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

	if utils.GetEnv("WORK_URL", "") == "" && utils.GetEnv("BPOW_KEY", "") == "" {
		panic("Either WORK_URL or BPOW_KEY must be set for work generation")
	}

	// Create app
	app := fiber.New()

	// BPoW if applicable
	var bpowClient *gql.BpowClient
	if utils.GetEnv("BPOW_KEY", "") != "" {
		bpowUrl := "https://boompow.banano.cc/graphql"
		if utils.GetEnv("BPOW_URL", "") != "" {
			bpowUrl = utils.GetEnv("BPOW_URL", "")
		}
		bpowClient = gql.NewBpowClient(bpowUrl, utils.GetEnv("BPOW_KEY", ""), false)
	}

	// Setup RPC Client
	nanoRpcUrl := utils.GetEnv("RPC_URL", "http://localhost:7076")
	rpcClient := net.RPCClient{
		Url:        nanoRpcUrl,
		BpowClient: bpowClient,
	}

	// Setup FCM client
	var fcmClient *fcm.Client
	fcmToken := utils.GetEnv("FCM_API_KEY", "")
	if fcmToken != "" {
		svc, err := fcm.NewClient(fcmToken)
		if err != nil {
			klog.Errorf("Error initating FCM client: %v", err)
			os.Exit(1)
		}
		fcmClient = svc
	}

	// Create repository
	fcmRepo := &repository.FcmTokenRepo{
		DB: db,
	}

	// Setup controllers
	pricePrefix := "nano"
	if *bananoMode {
		pricePrefix = "banano"
	}
	wsClientMap := controller.NewWSSubscriptions()
	wsc := controller.WsController{RPCClient: &rpcClient, PricePrefix: pricePrefix, WSClientMap: wsClientMap, BananoMode: *bananoMode, FcmTokenRepo: fcmRepo}
	hc := controller.HttpController{RPCClient: &rpcClient, BananoMode: *bananoMode, FcmTokenRepo: fcmRepo, WSClientMap: wsClientMap, FcmClient: fcmClient}

	// Cors middleware
	app.Use(cors.New())
	// Pprof
	app.Use(pprof.New())

	// HTTP Routes
	app.Post("/api", hc.HandleAction)
	app.Post("/callback", hc.HandleHTTPCallback)

	// Alerts
	app.Get("/alerts/:lang?", func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		activeAlert, err := GetActiveAlert(lang)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Unable to retrieve alerts")
		}
		return c.Status(fiber.StatusOK).JSON(activeAlert)
	})

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

	// 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404)
	})

	// Start nano WS client
	callbackChan := make(chan *net.WSCallbackMsg, 100)
	if utils.GetEnv("NODE_WS_URL", "") != "" {
		go net.StartNanoWSClient(utils.GetEnv("NODE_WS_URL", ""), &callbackChan)
	}

	// Read channel to notify clients of blocks of new blocks
	go func() {
		for msg := range callbackChan {
			// See if they are subscribed
			conns := wsClientMap.GetConnsForAccount(msg.Block.LinkAsAccount)
			if len(conns) > 0 {
				if msg.Block.Subtype == "send" {
					msg := map[string]interface{}{
						"account": msg.Account,
						"block":   msg.Block,
						"hash":    msg.Hash,
						"is_send": "true",
						"amount":  msg.Amount,
					}
					for _, conn := range conns {
						wsClientMap.WriteJsonSafe(conn, msg)
					}
				}
			}
		}
	}()

	// Automatically update connected clients on prices
	s := gocron.NewScheduler(time.UTC)

	s.Every(60).Seconds().Do(func() {
		// BTC and Nano price
		btcPrice, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-btc", pricePrefix))
		if err != nil {
			klog.Errorf("Error getting btc price in cron: %v", err)
			return
		}
		btcPriceFloat, err := strconv.ParseFloat(btcPrice, 64)
		if err != nil {
			klog.Errorf("Error parsing btc price in cron: %v", err)
			return
		}
		var nanoPriceFloat float64
		if *bananoMode {
			nanoPriceStr, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-nano", pricePrefix))
			if err != nil {
				klog.Errorf("Error getting nano price in cron: %v", err)
				return
			}
			nanoPriceFloat, err = strconv.ParseFloat(nanoPriceStr, 64)
		}
		conns := wsClientMap.GetAllConns()
		klog.V(3).Infof("Updating %d clients with prices", len(conns))
		for _, conn := range conns {
			currency := conn.Currency
			curStr, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-%s", pricePrefix, strings.ToLower(currency)))
			if err != nil {
				klog.Errorf("Error getting %s price in cron: %v", currency, err)
				continue
			}
			curFloat, err := strconv.ParseFloat(curStr, 64)
			if err != nil {
				klog.Errorf("Error parsing %s price in cron: %v", currency, err)
				continue
			}

			priceMessage := models.PriceMessage{
				Currency: currency,
				Price:    curFloat,
				BtcPrice: btcPriceFloat,
			}
			if *bananoMode {
				priceMessage.NanoPrice = &nanoPriceFloat
			}
			if conn.Conn != nil {
				wsClientMap.WriteJsonSafe(conn, priceMessage)
			}
		}
	})
	s.StartAsync()

	app.Listen(":3000")
}
