package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/repository"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/slices"
	"k8s.io/klog/v2"
)

const (
	// Time allowed to write a message to the peer.
	WriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	PongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	PingPeriod = (PongWait * 9) / 10

	// Maximum message size allowed from peer.
	MaxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte

	// IP Address
	IPAddress string
	ID        uuid.UUID
	Accounts  []string // Subscribed accounts
	Currency  string

	mutex sync.Mutex
}

var Upgrader = websocket.Upgrader{}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	Clients map[*Client]bool

	// Outbound messages to the client
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	BananoMode  bool
	PricePrefix string

	RPCClient    *net.RPCClient
	FcmTokenRepo *repository.FcmTokenRepo
}

func NewHub(bananomode bool, rpcClient *net.RPCClient, fcmTokenRepo *repository.FcmTokenRepo) *Hub {
	var pricePrefix string
	if bananomode {
		pricePrefix = "banano"
	} else {
		pricePrefix = "nano"
	}
	return &Hub{
		Broadcast:    make(chan []byte),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Clients:      make(map[*Client]bool),
		BananoMode:   bananomode,
		PricePrefix:  pricePrefix,
		RPCClient:    rpcClient,
		FcmTokenRepo: fcmTokenRepo,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

func (h *Hub) BroadcastToClient(client *Client, message []byte) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.Send <- message
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(PongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(PongWait)); return nil })
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				klog.Errorf("error: %v", err)
			}
			break
		}
		msg = bytes.TrimSpace(bytes.Replace(msg, newline, space, -1))

		// Process message
		// Determine type of message and unMarshal
		var baseRequest map[string]interface{}
		if err = json.Unmarshal(msg, &baseRequest); err != nil {
			klog.Errorf("Error unmarshalling websocket base request %s", err)
			errJson, _ := json.Marshal(InvalidRequestError)
			c.Hub.BroadcastToClient(c, errJson)
			continue
		}

		if _, ok := baseRequest["action"]; !ok {
			errJson, _ := json.Marshal(InvalidRequestError)
			c.Hub.BroadcastToClient(c, errJson)
			continue
		}

		if baseRequest["action"] == "account_subscribe" {
			var subscribeRequest models.AccountSubscribe
			if err = mapstructure.Decode(baseRequest, &subscribeRequest); err != nil {
				klog.Errorf("Error unmarshalling websocket subscribe request %s", err)
				errJson, _ := json.Marshal(InvalidRequestError)
				c.Hub.BroadcastToClient(c, errJson)
				continue
			}
			// Check if account is valid
			if !utils.ValidateAddress(subscribeRequest.Account, c.Hub.BananoMode) {
				klog.Errorf("Invalid account %s , %v", subscribeRequest.Account, c.Hub.BananoMode)
				c.Hub.BroadcastToClient(c, []byte("{\"error\":\"Invalid account\"}"))
				continue
			}

			// Handle subscribe
			// If UUID is present and valid, use that, otherwise generate a new one
			if subscribeRequest.Uuid != nil {
				id, err := uuid.Parse(*subscribeRequest.Uuid)
				if err != nil {
					c.ID = uuid.New()
				} else {
					c.ID = id
				}
			} else {
				// Create a UUID for this subscription
				c.ID = uuid.New()
			}
			// Get curency
			if subscribeRequest.Currency != nil && slices.Contains(net.CurrencyList, strings.ToUpper(*subscribeRequest.Currency)) {
				c.Currency = strings.ToUpper(*subscribeRequest.Currency)
			} else {
				c.Currency = "USD"
			}
			// Force nano_ address
			if !c.Hub.BananoMode {
				// Ensure account has nano_ address
				if strings.HasPrefix(subscribeRequest.Account, "xrb_") {
					subscribeRequest.Account = fmt.Sprintf("nano_%s", strings.TrimPrefix(subscribeRequest.Account, "xrb_"))
				}
			}

			klog.Infof("Received account_subscribe: %s, %s", subscribeRequest.Account, c.IPAddress)

			// Get account info
			accountInfo, err := c.Hub.RPCClient.MakeAccountInfoRequest(subscribeRequest.Account)
			if err != nil || accountInfo == nil {
				klog.Errorf("Error getting account info %v", err)
				c.Hub.BroadcastToClient(c, []byte("{\"error\":\"subscribe error\"}"))
				continue
			}

			// Add account to tracker
			if !slices.Contains(c.Accounts, subscribeRequest.Account) {
				c.Accounts = append(c.Accounts, subscribeRequest.Account)
			}

			// Get price info to include in response
			priceCur, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-%s", c.Hub.PricePrefix, strings.ToLower(c.Currency)))
			if err != nil {
				klog.Errorf("Error getting price %s %v", fmt.Sprintf("coingecko:%s-%s", c.Hub.PricePrefix, strings.ToLower(c.Currency)), err)
			}
			priceBtc, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-btc", c.Hub.PricePrefix))
			if err != nil {
				klog.Errorf("Error getting BTC price %v", err)
			}
			accountInfo["uuid"] = c.ID
			accountInfo["currency"] = c.Currency
			accountInfo["price"] = priceCur
			accountInfo["btc"] = priceBtc
			if c.Hub.BananoMode {
				// Also tag nano price
				// response['nano'] = float(await r.app['rdata'].hget("prices", f"{self.price_prefix}-nano"))
				priceNano, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-nano", c.Hub.PricePrefix))
				if err != nil {
					klog.Errorf("Error getting nano price %v", err)
				}
				accountInfo["nano"] = priceNano
			}

			// Tag pending count
			pendingCount, err := c.Hub.RPCClient.GetReceivableCount(subscribeRequest.Account, c.Hub.BananoMode)
			if err != nil {
				klog.Errorf("Error getting pending count %v", err)
			}
			accountInfo["pending_count"] = pendingCount

			// Send our finished response
			response, err := json.Marshal(accountInfo)
			if err != nil {
				klog.Errorf("Error marshalling account info %v", err)
				c.Hub.BroadcastToClient(c, []byte("{\"error\":\"subscribe error\"}"))
				continue
			}
			c.Hub.BroadcastToClient(c, response)

			// The user may have a different UUID every time, 1 token, and multiple accounts
			// We store account/token in postgres since that's what we care about
			// Or remove the token, if notifications disabled
			if !subscribeRequest.NotificationEnabled {
				// Set token in db
				c.Hub.FcmTokenRepo.DeleteFcmToken(subscribeRequest.FcmToken)
			} else {
				// Add/update token if not exists
				c.Hub.FcmTokenRepo.AddOrUpdateToken(subscribeRequest.FcmToken, subscribeRequest.Account)
			}
		} else if baseRequest["action"] == "fcm_update" {
			// Update FCM/notification preferences
			var fcmUpdateRequest models.FcmUpdate
			if err = mapstructure.Decode(baseRequest, &fcmUpdateRequest); err != nil {
				klog.Errorf("Error unmarshalling websocket fcm_update request %s", err)
				errJson, _ := json.Marshal(InvalidRequestError)
				c.Hub.BroadcastToClient(c, errJson)
				continue
			}
			// Check if account is valid
			if !utils.ValidateAddress(fcmUpdateRequest.Account, c.Hub.BananoMode) {
				c.Hub.BroadcastToClient(c, []byte("{\"error\":\"Invalid account\"}"))
				continue
			}
			// Do the updoot
			if !fcmUpdateRequest.Enabled {
				// Set token in db
				c.Hub.FcmTokenRepo.DeleteFcmToken(fcmUpdateRequest.FcmToken)
			} else {
				// Add token to db if not exists
				c.Hub.FcmTokenRepo.AddOrUpdateToken(fcmUpdateRequest.FcmToken, fcmUpdateRequest.Account)
			}
		} else {
			klog.Errorf("Unknown websocket request %s", msg)
			errJson, _ := json.Marshal(InvalidRequestError)
			c.Hub.BroadcastToClient(c, errJson)
			continue
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Handles a ws connection request from user
func WebsocketChl(hub *Hub, w http.ResponseWriter, r *http.Request) {
	clientIP := utils.IPAddress(r)

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		klog.Error(err)
		return
	}
	client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256), IPAddress: clientIP, Accounts: []string{}}
	client.Hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
