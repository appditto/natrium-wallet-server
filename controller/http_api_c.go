package controller

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/models/dbmodels"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/appleboy/go-fcm"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type HttpController struct {
	RPCClient   *net.RPCClient
	BananoMode  bool
	WSClientMap *WSClientMap
	DB          *gorm.DB
	FcmClient   *fcm.Client
}

var supportedActions = []string{
	"account_history",
}

func (hc *HttpController) HandleAction(c *fiber.Ctx) error {
	// ipAddress := utils.IPAddress(c)

	// Determine type of message and unMarshal
	var baseRequest models.BaseRequest
	if err := json.Unmarshal(c.Request().Body(), &baseRequest); err != nil {
		klog.Errorf("Error unmarshalling http base request %s", err)
		return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
	}

	if !slices.Contains(supportedActions, baseRequest.Action) {
		klog.Errorf("Action %s is not supported", baseRequest.Action)
		return c.Status(fiber.StatusBadRequest).JSON(models.UNSUPPORTED_ACTION_ERR)
	}

	// Handle actions
	if baseRequest.Action == "account_history" {
		// Retrieve account history
		var accountHistory models.AccountHistory
		if err := json.Unmarshal(c.Request().Body(), &accountHistory); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}

		// Check if account is valid
		if !utils.ValidateAddress(accountHistory.Account, hc.BananoMode) {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}
		// Limit the maximum count
		if accountHistory.Count != nil && *accountHistory.Count > 3500 {
			*accountHistory.Count = 3500
		}
		// Post request as-is to node
		response, err := hc.RPCClient.MakeRequest(accountHistory)
		if err != nil {
			klog.Errorf("Error making account history request %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error making account history request",
			})
		}
		var responseMap map[string]interface{}
		err = json.Unmarshal(response, &responseMap)
		if err != nil {
			klog.Errorf("Error unmarshalling account history response %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error making account history request",
			})
		}

		return c.Status(fiber.StatusOK).JSON(responseMap)
	}

	return c.Status(fiber.StatusBadRequest).JSON(models.UNSUPPORTED_ACTION_ERR)
}

func (hc *HttpController) HandleHTTPCallback(c *fiber.Ctx) error {
	var callback models.Callback
	if err := json.Unmarshal(c.Request().Body(), &callback); err != nil {
		klog.Errorf("Error unmarshalling callback %s", err)
		return c.Status(fiber.StatusOK).SendString("ok")
	}
	var callbackBlock models.CallbackBlock
	if err := json.Unmarshal([]byte(callback.Block), &callbackBlock); err != nil {
		klog.Errorf("Error unmarshalling callback block %s", err)
		return c.Status(fiber.StatusOK).SendString("ok")
	}

	// Supports push notificaiton
	if hc.FcmClient == nil {
		return c.Status(fiber.StatusOK).SendString("ok")
	}

	// Get previous block
	previous, err := hc.RPCClient.MakeBlockRequest(callbackBlock.Previous)
	if err != nil {
		klog.Errorf("Error making block request %s", err)
		return c.Status(fiber.StatusOK).SendString("ok")
	}

	// ! TODO 	? not sure what the point of this is
	// # See if this block was already pocketed
	// cached_hash = await r.app['rdata'].get(f"link_{hash}")
	// if cached_hash is not None:
	// 		return web.HTTPOk()

	minimumNotification := big.NewInt(0)
	minimumNotification.SetString("1000000000000000000000000", 10)

	curBalance := big.NewInt(0)
	curBalance, ok := curBalance.SetString(callbackBlock.Balance, 10)
	if !ok {
		klog.Error("Error settingcur balance")
		return c.Status(fiber.StatusOK).SendString("ok")
	}
	prevBalance := big.NewInt(0)
	prevBalance, ok = prevBalance.SetString(previous.Contents.Balance, 10)
	if !ok {
		klog.Error("Error setting prev balance")
		return c.Status(fiber.StatusOK).SendString("ok")
	}

	// Delta
	sendAmount := big.NewInt(0).Sub(prevBalance, curBalance)
	if sendAmount.Cmp(minimumNotification) > 0 {
		// Is a send we want to notify if we can
		// See if we have any tokens for this account
		var tokens []dbmodels.FcmToken
		if err := hc.DB.Where("account = ?", callbackBlock.LinkAsAccount).Find(&tokens).Error; err != nil {
			// ! Debug
			klog.Errorf("Error finding tokens %s", err)
			// No tokens
			return c.Status(fiber.StatusOK).SendString("ok")
		}
		if len(tokens) == 0 {
			// ! DEBUG
			klog.Errorf("No tokens found for account %s", callbackBlock.LinkAsAccount)
			// No tokens
			return c.Status(fiber.StatusOK).SendString("ok")
		}

		// We have tokens, make it happen
		var notificationTitle string
		var appName string
		if hc.BananoMode {
			appName = "Kalium"
			asBan, err := utils.RawToBanano(sendAmount.String(), true)
			if err != nil {
				klog.Errorf("Error converting raw to banano %s", err)
				return c.Status(fiber.StatusOK).SendString("ok")
			}
			notificationTitle = fmt.Sprintf("Received %f BANANO", asBan)
		} else {
			appName = "Natrium"
			asBan, err := utils.RawToNano(sendAmount.String(), true)
			if err != nil {
				klog.Errorf("Error converting raw to nano %s", err)
				return c.Status(fiber.StatusOK).SendString("ok")
			}
			notificationTitle = fmt.Sprintf("Received %f NANO", asBan)
		}
		notificationBody := fmt.Sprintf("Open %s to receive this transaction.", appName)

		klog.Infof("Prepared to send %s:%s", notificationTitle, notificationBody)

		for _, token := range tokens {
			// Create the message to be sent.
			msg := &fcm.Message{
				To:       token.FcmToken,
				Priority: "high",
				Data: map[string]interface{}{
					"click_action": "FLUTTER_NOTIFICATION_CLICK",
					"account":      callbackBlock.LinkAsAccount,
				},
				Notification: &fcm.Notification{
					Title: notificationTitle,
					Body:  notificationBody,
					Tag:   callbackBlock.LinkAsAccount,
					Sound: "default",
				},
			}
			klog.Infof("Sending notification to %s", token.FcmToken)
			hc.FcmClient.Send(msg)
		}
	}

	return c.Status(fiber.StatusOK).SendString("ok")
}
