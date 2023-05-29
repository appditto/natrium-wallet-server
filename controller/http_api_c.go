package controller

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/repository"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/appleboy/go-fcm"
	"github.com/go-chi/render"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/slices"
	"k8s.io/klog/v2"
)

// ! TODO - some of this stuff should be broken into separate functions, which would make it easier to add better integration tests
// ! e.g. `process` has a lot of logic in the http handler

type HttpController struct {
	RPCClient    *net.RPCClient
	BananoMode   bool
	FcmTokenRepo *repository.FcmTokenRepo
	FcmClient    *fcm.Client
}

var supportedActions = []string{
	"account_history",
	"process",
	"pending",
	"receivable",
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
func (hc *HttpController) HandleAction(w http.ResponseWriter, r *http.Request) {
	// Determine type of message and unMarshal
	var baseRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&baseRequest); err != nil {
		klog.Errorf("Error unmarshalling http base request %s", err)
		ErrInvalidRequest(w, r)
		return
	}

	if _, ok := baseRequest["action"]; !ok {
		ErrInvalidRequest(w, r)
		return
	}

	action := strings.ToLower(fmt.Sprintf("%v", baseRequest["action"]))

	if !slices.Contains(supportedActions, action) {
		klog.Errorf("Action %s is not supported", action)
		ErrUnsupportedAction(w, r)
		return
	}

	klog.Infof("Received HTTP action %s", action)

	// Trim count if it exists in action, so nobody can overload the node
	if val, ok := baseRequest["count"]; ok {
		countAsInt, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 64)
		if err != nil {
			klog.Errorf("Error converting count to int %s", err)
			ErrInvalidRequest(w, r)
			return
		}
		if countAsInt > 1000 || countAsInt < 0 {
			countAsInt = 1000
		}
		baseRequest["count"] = countAsInt
	}

	// Handle actions
	if action == "account_history" {
		// Retrieve account history
		var accountHistory models.AccountHistory
		if err := mapstructure.Decode(baseRequest, &accountHistory); err != nil {
			ErrInvalidRequest(w, r)
			return
		}

		// Check if account is valid
		if !utils.ValidateAddress(accountHistory.Account, hc.BananoMode) {
			ErrInvalidRequest(w, r)
			return
		}
		// Post request as-is to node
		response, err := hc.RPCClient.MakeRequest(accountHistory)
		if err != nil {
			klog.Errorf("Error making account history request %s", err)
			ErrInternalServerError(w, r, "Error making account history request")
			return
		}
		var responseMap map[string]interface{}
		err = json.Unmarshal(response, &responseMap)
		if err != nil {
			klog.Errorf("Error unmarshalling account history response %s", err)
			ErrInternalServerError(w, r, "Error making account history request")
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, responseMap)
		return
	} else if action == "process" {
		var jsonBlock bool
		if _, ok := baseRequest["json_block"]; ok {
			// The node is really dumb with their APIs, it's ambiguous whether it can be a bool or a string
			jsonBlock, ok = baseRequest["json_block"].(bool)
			if !ok {
				jsonBlockStr, ok := baseRequest["json_block"].(string)
				if !ok || (jsonBlockStr != "true" && jsonBlockStr != "false") {
					klog.Error("json_block must be true or false")
					ErrBadrequest(w, r, "json_block must be true or false")
					return
				}
				if jsonBlockStr == "true" {
					jsonBlock = true
				} else {
					jsonBlock = false
				}
			}
		}

		var processRequestStringBlock models.ProcessRequestStringBlock
		var processRequestBlock models.ProcessJsonBlock
		var processRequestJsonBlock models.ProcessRequestJsonBlock
		if jsonBlock {
			if err := mapstructure.Decode(baseRequest, &processRequestJsonBlock); err != nil {
				klog.Errorf("Error decoding process request %s", err)
				ErrBadrequest(w, r, err.Error())
				return
			}
		} else {
			if err := mapstructure.Decode(baseRequest, &processRequestStringBlock); err != nil {
				klog.Errorf("Error decoding process request string block %s", err)
				ErrBadrequest(w, r, err.Error())
				return
			}
			if err := json.Unmarshal([]byte(*processRequestStringBlock.Block), &processRequestBlock); err != nil {
				klog.Errorf("Error unmarshal process request %s", err)
				ErrBadrequest(w, r, err.Error())
				return
			}
			processRequestJsonBlock = models.ProcessRequestJsonBlock{
				Block:   &processRequestBlock,
				Action:  processRequestStringBlock.Action,
				SubType: processRequestStringBlock.SubType,
				DoWork:  processRequestStringBlock.DoWork,
			}
		}

		if processRequestJsonBlock.Block.Type != "state" {
			klog.Errorf("Only state blocks are supported")
			ErrBadrequest(w, r, "Only state blocks are supported")
			return
		}

		// Check if we wanna calculate work as part of this request
		doWork := false
		if processRequestJsonBlock.DoWork != nil && processRequestJsonBlock.Block.Work == nil {
			doWork = *processRequestJsonBlock.DoWork
		}

		// Determine the type of block
		if processRequestJsonBlock.SubType == nil {
			if strings.ReplaceAll(processRequestJsonBlock.Block.Link, "0", "") == "" {
				subtype := "change"
				processRequestJsonBlock.SubType = &subtype
			}
		} else if !slices.Contains([]string{"change", "open", "receive", "send"}, *processRequestJsonBlock.SubType) {
			klog.Errorf("Invalid subtype %s", *processRequestJsonBlock.SubType)
			ErrBadrequest(w, r, fmt.Sprintf("Invalid subtype %s", *processRequestJsonBlock.SubType))
			return
		}
		// ! TODO - what is the point of this, from old server
		// 	await r.app['rdata'].set(f"link_{block['link']}", "1", expire=3600)

		// Open blocks generate work on the public key, others use previous
		if doWork {
			var workBase string
			if processRequestJsonBlock.Block.Previous == "0" || processRequestJsonBlock.Block.Previous == "0000000000000000000000000000000000000000000000000000000000000000" {
				workbaseBytes, err := utils.AddressToPub(processRequestJsonBlock.Block.Account)
				if err != nil {
					klog.Errorf("Error converting address to public key %s", err)
					ErrBadrequest(w, r, err.Error())
					return
				}
				workBase = hex.EncodeToString(workbaseBytes)
				subtype := "open"
				processRequestJsonBlock.SubType = &subtype
			} else {
				workBase = processRequestJsonBlock.Block.Previous
				// Since we are here, let's validate the frontier
				accountInfo, err := hc.RPCClient.MakeAccountInfoRequest(processRequestJsonBlock.Block.Account)
				if err != nil {
					klog.Errorf("Error making account info request %s", err)
					ErrInternalServerError(w, r, "Error making account info request")
					return
				}
				if _, ok := accountInfo["error"]; !ok {
					// Account is opened
					if strings.ToLower(fmt.Sprintf("%s", accountInfo["frontier"])) != strings.ToLower(processRequestJsonBlock.Block.Previous) {
						klog.Errorf("Invalid frontier %s", processRequestJsonBlock.Block.Previous)
						ErrBadrequest(w, r, err.Error())
						return
					}
				}
			}

			// We're g2g
			var difficultyMultiplier int
			if hc.BananoMode {
				difficultyMultiplier = 1
			} else if processRequestJsonBlock.SubType == nil {
				// ! TODO - would be good to check if this is a send or receive if subtype isn't included
				difficultyMultiplier = 64
			} else if slices.Contains([]string{"change", "send"}, *processRequestJsonBlock.SubType) {
				difficultyMultiplier = 64
			} else {
				difficultyMultiplier = 1
			}
			if doWork {
				work, err := hc.RPCClient.WorkGenerate(workBase, difficultyMultiplier)
				if err != nil {
					klog.Errorf("Error generating work %s", err)
					ErrInternalServerError(w, r, "Error generating work")
					return
				}
				processRequestJsonBlock.Block.Work = &work
			}
		}

		if processRequestJsonBlock.Block.Work == nil {
			klog.Errorf("Work is required")
			ErrInvalidRequest(w, r)
			return
		}

		// Now G2G to actually broadcast it
		finalProcessRequest := map[string]interface{}{
			"action":     "process",
			"json_block": true,
			"block":      processRequestJsonBlock.Block,
		}
		if processRequestJsonBlock.SubType != nil {
			finalProcessRequest["subtype"] = processRequestJsonBlock.SubType
		}
		rawResp, err := hc.RPCClient.MakeRequest(finalProcessRequest)
		if err != nil {
			klog.Errorf("Error making process request %s", err)
			ErrInternalServerError(w, r, "Error making process request")
			return
		}
		var responseMap map[string]interface{}
		err = json.Unmarshal(rawResp, &responseMap)
		if err != nil {
			klog.Errorf("Error unmarshalling response %s", err)
			ErrInternalServerError(w, r, "Error unmarshalling response")
			return
		}
		klog.Infof("Successfully processed block %s", responseMap["hash"])
		render.Status(r, http.StatusOK)
		render.JSON(w, r, responseMap)
		return
	} else if action == "pending" {
		var pendingRequest models.PendingRequest
		if err := mapstructure.Decode(baseRequest, &pendingRequest); err != nil {
			ErrInvalidRequest(w, r)
			return
		}
		// We force include_only_confirmed since natrium/kalium don't include it
		// "receivable" can be used to bypass this behavior
		ioc := true
		pendingRequest.IncludeOnlyConfirmed = &ioc
		if pendingRequest.Count < 1 {
			pendingRequest.Count = 1000
		}
		rawResp, err := hc.RPCClient.MakeRequest(pendingRequest)
		if err != nil {
			klog.Errorf("Error making pending request %s", err)
			ErrInternalServerError(w, r, "Error making pending request")
			return
		}
		var responseMap map[string]interface{}
		err = json.Unmarshal(rawResp, &responseMap)
		if err != nil {
			klog.Errorf("Error unmarshalling response %s", err)
			ErrInternalServerError(w, r, "Error unmarshalling response")
			return
		}
		render.Status(r, http.StatusOK)
		render.JSON(w, r, responseMap)
		return
	}

	rawResp, err := hc.RPCClient.MakeRequest(baseRequest)
	if err != nil {
		klog.Errorf("Error making request %s", err)
		ErrInternalServerError(w, r, "Error making request")
		return
	}
	var responseMap map[string]interface{}
	err = json.Unmarshal(rawResp, &responseMap)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		ErrInternalServerError(w, r, "Error unmarshalling response")
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, responseMap)
}

// HTTP Callback is only for push notifications
func (hc *HttpController) HandleHTTPCallback(w http.ResponseWriter, r *http.Request) {
	var callback models.Callback
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		klog.Errorf("Error unmarshalling callback %s", err)
		render.Status(r, http.StatusOK)
		return
	}
	var callbackBlock models.CallbackBlock
	if err := json.Unmarshal([]byte(callback.Block), &callbackBlock); err != nil {
		klog.Errorf("Error unmarshalling callback block %s", err)
		render.Status(r, http.StatusOK)
		return
	}

	// Supports push notificaiton
	if hc.FcmClient == nil {
		render.Status(r, http.StatusOK)
		return
	}

	// Get previous block
	previous, err := hc.RPCClient.MakeBlockRequest(callbackBlock.Previous)
	if err != nil {
		klog.Errorf("Error making block request %s", err)
		render.Status(r, http.StatusOK)
		return
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
		klog.Error("Error setting cur balance")
		render.Status(r, http.StatusOK)
		return
	}
	prevBalance := big.NewInt(0)
	prevBalance, ok = prevBalance.SetString(previous.Contents.Balance, 10)
	if !ok {
		render.Status(r, http.StatusOK)
		return
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
			render.Status(r, http.StatusOK)
			return
		}
		if len(tokens) == 0 {
			// No tokens
			render.Status(r, http.StatusOK)
			return
		}

		// We have tokens, make it happen
		var notificationTitle string
		var appName string
		if hc.BananoMode {
			appName = "Kalium"
			asBan, err := utils.RawToBanano(sendAmount.String(), true)
			if err != nil {
				klog.Errorf("Error converting raw to banano %s", err)
				render.Status(r, http.StatusOK)
				return
			}
			notificationTitle = fmt.Sprintf("Received %s BANANO", strconv.FormatFloat(asBan, 'f', -1, 64))
		} else {
			appName = "Natrium"
			asBan, err := utils.RawToNano(sendAmount.String(), true)
			if err != nil {
				klog.Errorf("Error converting raw to nano %s", err)
				render.Status(r, http.StatusOK)
				return
			}
			notificationTitle = fmt.Sprintf("Received Ӿ%s", strconv.FormatFloat(asBan, 'f', -1, 64))
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
				render.Status(r, http.StatusOK)
				return
			}
		}
	}

	render.Status(r, http.StatusOK)
}
