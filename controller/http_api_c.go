package controller

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/repository"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/appleboy/go-fcm"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/slices"
	"k8s.io/klog/v2"
)

// ! TODO - some of this stuff should be broken into separate functions, which would make it easier to add better integration tests
// ! e.g. `process` has a lot of logic in the http handler

type HttpController struct {
	RPCClient    *net.RPCClient
	BananoMode   bool
	WSClientMap  *WSClientMap
	FcmTokenRepo *repository.FcmTokenRepo
	FcmClient    *fcm.Client
}

var supportedActions = []string{
	"account_history",
	"process",
	"pending",
	"account_balance",
	"account_block_count",
	"account_check",
	"account_info",
	"account_representative",
	"account_subscribe",
	"account_weight",
	"accounts_balances",
	"accounts_frontiers",
	"accounts_pending",
	"available_supply",
	"block",
	"block_hash",
	"blocks",
	"block_info",
	"blocks_info",
	"block_account", "block_count",
	"block_count_type",
	"chain",
	"frontiers",
	"frontier_count",
	"history",
	"key_expand",
	"representatives",
	"republish",
	"peers",
	"version",
	"pending_exists",
}

// HandleHTTPRequest handles all requests to the http server
// It's generally designed to mimic the nano node's RPC API
// Though we do additional processing in the middle for some actions
func (hc *HttpController) HandleAction(c *fiber.Ctx) error {
	// ipAddress := utils.IPAddress(c)

	// Determine type of message and unMarshal
	var baseRequest map[string]interface{}
	if err := json.Unmarshal(c.Request().Body(), &baseRequest); err != nil {
		klog.Errorf("Error unmarshalling http base request %s", err)
		return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
	}

	if _, ok := baseRequest["action"]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
	}

	action := strings.ToLower(fmt.Sprintf("%v", baseRequest["action"]))

	if !slices.Contains(supportedActions, action) {
		klog.Errorf("Action %s is not supported", action)
		return c.Status(fiber.StatusBadRequest).JSON(models.UNSUPPORTED_ACTION_ERR)
	}

	// Trim count if it exists in action, so nobody can overload the node
	if val, ok := baseRequest["count"]; ok {
		countAsInt, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 64)
		if err != nil {
			klog.Errorf("Error converting count to int %s", err)
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}
		if countAsInt > 3500 || countAsInt < 0 {
			countAsInt = 3500
		}
		baseRequest["count"] = countAsInt
	}

	// Handle actions
	if action == "account_history" {
		// Retrieve account history
		var accountHistory models.AccountHistory
		if err := json.Unmarshal(c.Request().Body(), &accountHistory); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}

		// Check if account is valid
		if !utils.ValidateAddress(accountHistory.Account, hc.BananoMode) {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
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
	} else if action == "process" {
		// Process request
		var processRequest models.ProcessRequest
		if err := json.Unmarshal(c.Request().Body(), &processRequest); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}

		if processRequest.Block == nil && processRequest.JsonBlock == nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}

		// Sometimes requests come with a string block representation
		if processRequest.JsonBlock == nil {
			if err := json.Unmarshal([]byte(*processRequest.Block), &processRequest.JsonBlock); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
			}
		}

		if processRequest.JsonBlock.Type != "state" {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}

		// Check if we wanna calculate work as part of this request
		doWork := false
		if processRequest.DoWork != nil && processRequest.JsonBlock.Work == nil {
			doWork = *processRequest.DoWork
		}

		// Determine the type of block
		if processRequest.SubType == nil {
			if strings.ReplaceAll(processRequest.JsonBlock.Link, "0", "") == "" {
				subtype := "change"
				processRequest.SubType = &subtype
			}
		} else if !slices.Contains([]string{"change", "open", "receive", "send"}, *processRequest.SubType) {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}
		// ! TODO - what is the point of this, from old server
		// 	await r.app['rdata'].set(f"link_{block['link']}", "1", expire=3600)

		// Open blocks generate work on the public key, others use previous
		var workBase string
		if processRequest.JsonBlock.Previous == "0" || processRequest.JsonBlock.Previous == "0000000000000000000000000000000000000000000000000000000000000000" {
			workbaseBytes, err := utils.AddressToPub(processRequest.JsonBlock.Account)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
			}
			workBase = hex.EncodeToString(workbaseBytes)
			subtype := "open"
			processRequest.SubType = &subtype
		} else {
			workBase = processRequest.JsonBlock.Previous
			// Since we are here, let's validate the frontier
			accountInfo, err := hc.RPCClient.MakeAccountInfoRequest(processRequest.JsonBlock.Account)
			if err != nil {
				klog.Errorf("Error making account info request %s", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error making account info request",
				})
			}
			if _, ok := accountInfo["error"]; !ok {
				// Account is opened
				if strings.ToLower(fmt.Sprintf("%s", accountInfo["frontier"])) != strings.ToLower(processRequest.JsonBlock.Previous) {
					return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
				}
			}
		}

		// We're g2g
		var difficultyMultiplier int
		if hc.BananoMode {
			difficultyMultiplier = 1
		} else if processRequest.SubType == nil {
			// ! TODO - would be good to check if this is a send or receive if subtype isn't included
			difficultyMultiplier = 64
		} else if slices.Contains([]string{"change", "send"}, *processRequest.SubType) {
			difficultyMultiplier = 64
		} else {
			difficultyMultiplier = 1
		}
		if doWork {
			work, err := hc.RPCClient.WorkGenerate(workBase, difficultyMultiplier)
			if err != nil {
				klog.Errorf("Error generating work %s", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error generating work",
				})
			}
			processRequest.JsonBlock.Work = &work
		}

		if processRequest.JsonBlock.Work == nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}

		// Now G2G to actually broadcast it
		finalProcessRequest := map[string]interface{}{
			"action":     "process",
			"json_block": true,
			"block":      processRequest.JsonBlock,
		}
		rawResp, err := hc.RPCClient.MakeRequest(finalProcessRequest)
		if err != nil {
			klog.Errorf("Error making process request %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error making process request",
			})
		}
		var responseMap map[string]interface{}
		err = json.Unmarshal(rawResp, &responseMap)
		if err != nil {
			klog.Errorf("Error unmarshalling response %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error unmarshalling response",
			})
		}
		if _, ok := responseMap["hash"]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(responseMap)
		}
		return c.Status(fiber.StatusOK).JSON(responseMap)
	} else if action == "pending" {
		var pendingRequest models.PendingRequest
		if err := json.Unmarshal(c.Request().Body(), &pendingRequest); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.INVALID_REQUEST_ERR)
		}
		ioc := true
		pendingRequest.IncludeOnlyConfirmed = &ioc
		rawResp, err := hc.RPCClient.MakeRequest(pendingRequest)
		if err != nil {
			klog.Errorf("Error making pending request %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error making pending request",
			})
		}
		var responseMap map[string]interface{}
		err = json.Unmarshal(rawResp, &responseMap)
		if err != nil {
			klog.Errorf("Error unmarshalling response %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error unmarshalling response",
			})
		}
		return c.Status(fiber.StatusOK).JSON(responseMap)
	}

	rawResp, err := hc.RPCClient.MakeRequest(baseRequest)
	if err != nil {
		klog.Errorf("Error making request %s", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error making request",
		})
	}
	var responseMap map[string]interface{}
	err = json.Unmarshal(rawResp, &responseMap)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error unmarshalling response",
		})
	}
	return c.Status(fiber.StatusOK).JSON(responseMap)
}

// HTTP Callback is only for push notifications
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
		tokens, err := hc.FcmTokenRepo.GetTokensForAccount(callbackBlock.LinkAsAccount)
		if err != nil {
			klog.Errorf("Error finding tokens for account %s %v", tokens, err)
			// No tokens
			return c.Status(fiber.StatusOK).SendString("ok")
		}
		if len(tokens) == 0 {
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
			notificationTitle = fmt.Sprintf("Received %s BANANO", strconv.FormatFloat(asBan, 'f', -1, 64))
		} else {
			appName = "Natrium"
			asBan, err := utils.RawToNano(sendAmount.String(), true)
			if err != nil {
				klog.Errorf("Error converting raw to nano %s", err)
				return c.Status(fiber.StatusOK).SendString("ok")
			}
			notificationTitle = fmt.Sprintf("Received %s Nano", strconv.FormatFloat(asBan, 'f', -1, 64))
		}
		notificationBody := fmt.Sprintf("Open %s to receive this transaction.", appName)

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
			_, err := hc.FcmClient.Send(msg)
			if err != nil {
				klog.Errorf("Error sending notification %s", err)
				return c.Status(fiber.StatusOK).SendString("ok")
			}
		}
	}

	return c.Status(fiber.StatusOK).SendString("ok")
}
