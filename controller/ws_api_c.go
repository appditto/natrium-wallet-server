package controller

import (
	"encoding/json"

	"github.com/appditto/natrium-wallet-server/models"
	"github.com/gofiber/websocket/v2"
	"github.com/golang/glog"
)

func HandleWSMessage(c *websocket.Conn) {
	var (
		mt  int
		msg []byte
		err error
	)
	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			glog.Error("read: %s", err)
			break
		}
		glog.Infof("recv: %s", msg)
		// Determine type of message and unMarshal
		var baseRequest models.BaseRequest
		if err = json.Unmarshal(msg, &baseRequest); err != nil {
			glog.Errorf("Error unmarshalling websocket base request %s", err)
			errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
			if err = c.WriteMessage(mt, errJson); err != nil {
				glog.Errorf("write: %s", err)
				break
			}
			continue
		}

		if baseRequest.Action == "account_subscribe" {
			var subscribeRequest models.AccountSubscribe
			if err = json.Unmarshal(msg, &subscribeRequest); err != nil {
				errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
				if err = c.WriteMessage(mt, errJson); err != nil {
					glog.Errorf("write: %s", err)
					break
				}
				continue
			}
			// Handle subscribe
			glog.Infof("Received account_subscribe: %s, %s, %s, %s", subscribeRequest.Account, subscribeRequest.Currency, subscribeRequest.FcmToken, subscribeRequest.NotificationEnabled)
		} else {
			// Unknown request via websocket
			glog.Errorf("Unknown action sent via websocket %s", baseRequest.Action)
			errJson, _ := json.Marshal(models.INVALID_REQUEST_ERR)
			if err = c.WriteMessage(mt, errJson); err != nil {
				glog.Errorf("write: %s", err)
				break
			}
		}
	}
}
