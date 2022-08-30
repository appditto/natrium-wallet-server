package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type WsController struct {
	RPCClient   *net.RPCClient
	PricePrefix string
	WSClientMap *WSClientMap
	BananoMode  bool
}

func (wc *WsController) HandleWSMessage(c *websocket.Conn) {
	ipAddr := c.Locals("ip")
	// Create a UUID for this subscription
	id := uuid.New()
	wc.WSClientMap.Put(WSClient{id: id, accounts: []string{}})
	// Cleanups when connection is closed
	defer wc.WSClientMap.Delete(id)
	defer database.GetRedisDB().Hdel("connected_clients", id.String())

	var (
		mt  int
		msg []byte
		err error
	)
	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			klog.Error("read: %s", err)
			break
		}
		klog.Infof("recv: %s", msg)
		// Determine type of message and unMarshal
		var baseRequest models.BaseRequest
		if err = json.Unmarshal(msg, &baseRequest); err != nil {
			klog.Errorf("Error unmarshalling websocket base request %s", err)
			errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
			if err = c.WriteMessage(mt, errJson); err != nil {
				klog.Errorf("write: %s", err)
				break
			}
			continue
		}

		if baseRequest.Action == "account_subscribe" {
			var subscribeRequest models.AccountSubscribe
			if err = json.Unmarshal(msg, &subscribeRequest); err != nil {
				errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
				if err = c.WriteMessage(mt, errJson); err != nil {
					klog.Errorf("write: %s", err)
					break
				}
				continue
			}
			// Check if account is valid
			if !utils.ValidateAddress(subscribeRequest.Account, wc.BananoMode) {
				c.WriteMessage(mt, []byte("{\"error\":\"Invalid account\"}"))
				continue
			}
			// Handle subscribe
			// New subscription (no UUID)
			if subscribeRequest.Uuid == nil {
				klog.Infof("Received account_subscribe: %s, %s", subscribeRequest.Account, ipAddr)
				// Get curency
				var currency string
				if subscribeRequest.Currency != nil {
					currency = *subscribeRequest.Currency
				} else {
					currency = "usd"
				}
				// Ensure account has nano_ address
				if strings.HasPrefix(subscribeRequest.Account, "xrb_") {
					subscribeRequest.Account = fmt.Sprintf("nano_%s", strings.TrimPrefix(subscribeRequest.Account, "xrb_"))
				}

				// Get account info
				accountInfo, err := wc.RPCClient.MakeAccountInfoRequest(subscribeRequest.Account)
				if err != nil {
					klog.Errorf("Error getting account info %v", err)
					c.WriteMessage(mt, []byte("{\"error\":\"subscribe error\"}"))
					continue
				}

				// Add account to tracker
				wc.WSClientMap.AddAccount(id, subscribeRequest.Account)

				// Add preferences to database
				// ! We can't set expiry on these keys, but we are intended to terminate them after the client disconnected
				//! We may want to clean up this HSet during deployments or something
				database.GetRedisDB().Hset("connected_clients", id.String(), currency)

				// Get price info to include in response
				priceCur, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("%s-%s", wc.PricePrefix, strings.ToLower(currency)))
				if err != nil {
					klog.Errorf("Error getting price %v", err)
				}
				priceBtc, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("%s-btc", wc.PricePrefix))
				if err != nil {
					klog.Errorf("Error getting BTC price %v", err)
				}
				accountInfo["currency"] = currency
				accountInfo["price"] = priceCur
				accountInfo["btc"] = priceBtc
				if wc.BananoMode {
					// Also tag nano price
					// response['nano'] = float(await r.app['rdata'].hget("prices", f"{self.price_prefix}-nano"))
					priceNano, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("%s-nano", wc.PricePrefix))
					if err != nil {
						klog.Errorf("Error getting nano price %v", err)
					}
					accountInfo["nano"] = priceNano
				}

				// Tag pending count
				pendingCount, err := wc.RPCClient.GetReceivableCount(subscribeRequest.Account, wc.BananoMode)
				if err != nil {
					klog.Errorf("Error getting pending count %v", err)
				}
				accountInfo["pending_count"] = pendingCount

				// Send our finished response
				c.WriteJSON(accountInfo)

				// ! TODO deal with FCM tokens

			}
		} else {
			// Unknown request via websocket
			klog.Errorf("Unknown action sent via websocket %s", baseRequest.Action)
			errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
			if err = c.WriteMessage(mt, errJson); err != nil {
				klog.Errorf("write: %s", err)
				//break
			}
		}
	}
}
