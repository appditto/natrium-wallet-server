//go:generate go run github.com/Khan/genqlient
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
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
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
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
	app := chi.NewRouter()

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
	hc := controller.HttpController{RPCClient: &rpcClient, BananoMode: *bananoMode, FcmTokenRepo: fcmRepo, FcmClient: fcmClient}

	// Cors middleware
	app.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		//AllowedOrigins:   []string{"*"},
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	// Pprof
	// app.Use(pprof.New())

	// HTTP Routes
	app.Post("/api", hc.HandleAction)
	app.Post("/callback", hc.HandleHTTPCallback)

	// Alerts
	app.Route("/alerts", func(r chi.Router) {
		r.Get("/{lang}", func(w http.ResponseWriter, r *http.Request) {
			lang := chi.URLParam(r, "lang")
			activeAlert, err := GetActiveAlert(lang)
			if err != nil {
				controller.ErrInternalServerError(w, r, "Unable to retrieve alerts")
				return
			}
			render.Status(r, http.StatusOK)
			render.JSON(w, r, activeAlert)
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			activeAlert, err := GetActiveAlert("en")
			if err != nil {
				controller.ErrInternalServerError(w, r, "Unable to retrieve alerts")
				return
			}
			render.Status(r, http.StatusOK)
			render.JSON(w, r, activeAlert)
		})
	})

	// Setup WS endpoint
	wsHub := controller.NewHub(*bananoMode, &rpcClient, fcmRepo)
	go wsHub.Run()
	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		controller.WebsocketChl(wsHub, w, r)
	})

	// Start nano WS client
	callbackChan := make(chan *net.WSCallbackMsg, 100)
	if utils.GetEnv("NODE_WS_URL", "") != "" {
		go net.StartNanoWSClient(utils.GetEnv("NODE_WS_URL", ""), &callbackChan)
	}

	// Read channel to notify clients of blocks of new blocks
	go func() {
		for msg := range callbackChan {
			if msg.Block.Subtype != "send" {
				continue
			}
			callbackMsg := map[string]interface{}{
				"account": msg.Account,
				"block":   msg.Block,
				"hash":    msg.Hash,
				"is_send": "true",
				"amount":  msg.Amount,
			}
			serialized, err := json.Marshal(callbackMsg)
			if err != nil {
				klog.Errorf("Error serializing callback message: %v", err)
				continue
			}

			// See if they are subscribed
			for client, _ := range wsHub.Clients {
				for _, account := range client.Accounts {
					if account == msg.Block.LinkAsAccount {
						client.Send <- serialized
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
		for client, _ := range wsHub.Clients {
			currency := client.Currency
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
			serialized, err := json.Marshal(priceMessage)
			if err != nil {
				klog.Errorf("Error serializing price message: %v", err)
				continue
			}
			client.Send <- serialized

		}
	})
	s.StartAsync()

	http.ListenAndServe(":3000", app)
}
