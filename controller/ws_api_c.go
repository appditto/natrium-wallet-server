package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/gofiber/websocket/v2"
	"k8s.io/klog/v2"
)

type WsController struct {
	RPCClient *net.RPCClient
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
				accountInfo["currency"] = currency

				// Update fcm token or delete based on notifications_enable
			}
		} else {
			// Unknown request via websocket
			klog.Errorf("Unknown action sent via websocket %s", baseRequest.Action)
			errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
			if err = c.WriteMessage(mt, errJson); err != nil {
				klog.Errorf("write: %s", err)
				break
			}
		}
	}
}
