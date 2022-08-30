package controller

import (
	"encoding/json"

	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/net"
	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/slices"
	"k8s.io/klog/v2"
)

type HttpController struct {
	RPCClient  *net.RPCClient
	BananoMode bool
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

func HandleHTTPCallback(c *fiber.Ctx) error {
	return nil
}
