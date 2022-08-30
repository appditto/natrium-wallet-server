package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/models/dbmodels"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type WsController struct {
	RPCClient   *net.RPCClient
	PricePrefix string
	WSClientMap *WSClientMap
	BananoMode  bool
	DB          *gorm.DB
}

func (wc *WsController) HandleWSMessage(c *websocket.Conn) {
	ipAddr := c.Locals("ip")

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
			// If UUID is present and valid, use that, otherwise generate a new one
			var id uuid.UUID
			if subscribeRequest.Uuid != nil {
				id = *subscribeRequest.Uuid
			} else {
				// Create a UUID for this subscription
				id = uuid.New()
			}
			wc.WSClientMap.Put(WSClient{id: id, accounts: []string{}, currency: "usd"})
			// Cleanups when connection is closed
			defer wc.WSClientMap.Delete(id)

			// Get curency
			var currency string
			if subscribeRequest.Currency != nil && slices.Contains(net.CurrencyList, *subscribeRequest.Currency) {
				currency = *subscribeRequest.Currency
			} else {
				currency = "usd"
			}
			//  Update currency on sub
			wc.WSClientMap.UpdateCurrency(id, currency)
			// Force nano_ address
			if !wc.BananoMode {
				// Ensure account has nano_ address
				if strings.HasPrefix(subscribeRequest.Account, "xrb_") {
					subscribeRequest.Account = fmt.Sprintf("nano_%s", strings.TrimPrefix(subscribeRequest.Account, "xrb_"))
				}
			}

			klog.Infof("Received account_subscribe: %s, %s", subscribeRequest.Account, ipAddr)

			// Get account info
			accountInfo, err := wc.RPCClient.MakeAccountInfoRequest(subscribeRequest.Account)
			if err != nil {
				klog.Errorf("Error getting account info %v", err)
				c.WriteMessage(mt, []byte("{\"error\":\"subscribe error\"}"))
				continue
			}

			// Add account to tracker
			wc.WSClientMap.AddAccount(id, subscribeRequest.Account)

			// Get price info to include in response
			priceCur, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-%s", wc.PricePrefix, strings.ToLower(currency)))
			if err != nil {
				klog.Errorf("Error getting price %v", err)
			}
			priceBtc, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:%s-btc", wc.PricePrefix))
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

			// The user may have a different UUID every time, 1 token, and multiple accounts
			// We store account/token in postgres since that's what we care about
			// Or remove the token, if notifications disabled
			if !subscribeRequest.NotificationEnabled {
				// Set token in db
				wc.DB.Delete(&dbmodels.FcmToken{}, "fcm_token = ?", subscribeRequest.FcmToken)
			} else {
				// Add token to db if not exists
				var count int64
				err = wc.DB.Model(&dbmodels.FcmToken{}).Where("fcm_token = ?", subscribeRequest.FcmToken).Where("account = ?", subscribeRequest.Account).Count(&count).Error
				if err != nil || count == 0 {
					fcmToken := &dbmodels.FcmToken{
						FcmToken: subscribeRequest.FcmToken,
						Account:  subscribeRequest.Account,
					}
					wc.DB.Create(fcmToken)
				} else if count > 0 {
					// Already exists so we will update updated_at
					if err = wc.DB.Model(&dbmodels.FcmToken{}).Where("fcm_token = ?", subscribeRequest.FcmToken).Where("account = ?", subscribeRequest.Account).Update("updated_at", time.Now()).Error; err != nil {
						klog.Errorf("Error updating fcm token updated_at %v", err)
					}
				}
			}
		} else if baseRequest.Action == "fcm_update" {
			// Update FCM/notification preferences
			var fcmUpdateRequest models.FcmUpdate
			if err = json.Unmarshal(msg, &fcmUpdateRequest); err != nil {
				errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
				if err = c.WriteMessage(mt, errJson); err != nil {
					klog.Errorf("write: %s", err)
					break
				}
				continue
			}
			// Check if account is valid
			if !utils.ValidateAddress(fcmUpdateRequest.Account, wc.BananoMode) {
				c.WriteMessage(mt, []byte("{\"error\":\"Invalid account\"}"))
				continue
			}
			// Do the updoot
			if !fcmUpdateRequest.Enabled {
				// Set token in db
				wc.DB.Delete(&dbmodels.FcmToken{}, "fcm_token = ?", fcmUpdateRequest.FcmToken)
			} else {
				// Add token to db if not exists
				var count int64
				err = wc.DB.Model(&dbmodels.FcmToken{}).Where("fcm_token = ?", fcmUpdateRequest.FcmToken).Where("account = ?", fcmUpdateRequest.Account).Count(&count).Error
				if err != nil || count == 0 {
					fcmToken := &dbmodels.FcmToken{
						FcmToken: fcmUpdateRequest.FcmToken,
						Account:  fcmUpdateRequest.Account,
					}
					wc.DB.Create(fcmToken)
				} else if count > 0 {
					// Already exists so we will update updated_at
					if err = wc.DB.Model(&dbmodels.FcmToken{}).Where("fcm_token = ?", fcmUpdateRequest.FcmToken).Where("account = ?", fcmUpdateRequest.Account).Update("updated_at", time.Now()).Error; err != nil {
						klog.Errorf("Error updating fcm token updated_at %v", err)
					}
				}
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
