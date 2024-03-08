package net

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/appditto/natrium-wallet-server/gql"
	"github.com/appditto/natrium-wallet-server/models"
	"github.com/appditto/natrium-wallet-server/utils"
	"k8s.io/klog/v2"
)

type RPCClient struct {
	Url        string
	BpowClient *gql.BpowClient
}

// Base request
func (client *RPCClient) MakeRequest(request interface{}) ([]byte, error) {
	requestBody, _ := json.Marshal(request)
	// HTTP post
	httpRequest, err := http.NewRequest(http.MethodPost, client.Url, bytes.NewBuffer(requestBody))
	if err != nil {
		klog.Errorf("Error building request %s", err)
		return nil, err
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	resp, err := Client.Do(httpRequest)
	if err != nil {
		klog.Errorf("Error making RPC request %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	// Try to decode+deserialize
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Error decoding response body %s", err)
		return nil, err
	}
	return body, nil
}

func (client *RPCClient) MakeAccountInfoRequest(account string) (map[string]interface{}, error) {
	request := models.AccountInfoAction{
		Action:         "account_info",
		Account:        account,
		Pending:        true,
		Representative: true,
	}
	response, err := client.MakeRequest(request)
	if err != nil {
		klog.Errorf("Error making request %s", err)
		return nil, err
	}
	var responseMap map[string]interface{}
	err = json.Unmarshal(response, &responseMap)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		return nil, err
	}
	// Check that it's ok
	if _, ok := responseMap["frontier"]; !ok {
		if _, ok := responseMap["error"]; ok {
			if responseMap["error"].(string) == "Account not found" {
				// This response is ok, unopened account
				return responseMap, nil
			}
		}
		klog.Errorf("Error in account_info response %s", err)
		return nil, err
	}

	return responseMap, nil
}

// This returns how many pending blocks an account has, up to 51, for anti-spam measures
func (client *RPCClient) GetReceivableCount(account string, bananoMode bool) (int, error) {
	threshold := "1000000000000000000000000"
	if bananoMode {
		threshold = "1000000000000000000000000000"
	}
	request := models.ReceivableRequest{
		Action:               "receivable",
		Account:              account,
		Threshold:            threshold,
		Count:                51,
		IncludeOnlyConfirmed: true,
	}
	response, err := client.MakeRequest(request)
	if err != nil {
		klog.Errorf("Error making request %s", err)
		return 0, err
	}
	var parsed models.ReceivableResponse
	err = json.Unmarshal(response, &parsed)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		return 0, err
	}

	return len(parsed.Blocks), nil
}

func (client *RPCClient) MakeBlockRequest(hash string) (models.BlockResponse, error) {
	request := models.BlockRequest{
		Action:    "block_info",
		Hash:      hash,
		JsonBlock: true,
	}
	response, err := client.MakeRequest(request)
	if err != nil {
		klog.Errorf("Error making request %s", err)
		return models.BlockResponse{}, err
	}
	var blockResponse models.BlockResponse
	err = json.Unmarshal(response, &blockResponse)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		return models.BlockResponse{}, err
	}
	return blockResponse, nil
}

func (client *RPCClient) WorkGenerate(hash string, difficultyMultiplier int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chanSize := 0
	if client.BpowClient != nil {
		chanSize++
	}
	if utils.GetEnv("WORK_URL", "") != "" {
		chanSize++
	}

	if chanSize == 0 {
		return "", fmt.Errorf("No work providers available")
	}

	results := make(chan string, chanSize)
	errors := make(chan error, chanSize)

	if client.BpowClient != nil {
		go func() {
			res, err := client.BpowClient.WorkGenerate(hash, difficultyMultiplier)
			if err != nil {
				errors <- err
				return
			}
			results <- res
		}()
	}

	workURL := utils.GetEnv("WORK_URL", "")
	if workURL != "" {
		go client.httpWorkGenerate(ctx, hash, difficultyMultiplier, results, errors)
	}

	select {
	case res := <-results:
		client.sendWorkCancel(hash)
		return res, nil
	case err := <-errors:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (client *RPCClient) httpWorkGenerate(ctx context.Context, hash string, difficultyMultiplier int, results chan<- string, errors chan<- error) {
	difficulty := "fffffff800000000"
	if difficultyMultiplier < 64 {
		difficulty = "fffffe0000000000"
	}

	request := models.WorkGenerate{
		Action:     "work_generate",
		Hash:       hash,
		Difficulty: difficulty,
	}

	requestBody, _ := json.Marshal(request)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, utils.GetEnv("WORK_URL", ""), bytes.NewBuffer(requestBody))
	if err != nil {
		errors <- err
		return
	}
	httpRequest.Header.Add("Content-Type", "application/json")

	resp, err := Client.Do(httpRequest)
	if err != nil {
		errors <- err
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errors <- err
		return
	}

	var workResp models.WorkResponse
	if err := json.Unmarshal(body, &workResp); err != nil {
		errors <- err
		return
	}

	results <- workResp.Work
}

func (client *RPCClient) sendWorkCancel(hash string) {
	workURL := utils.GetEnv("WORK_URL", "")
	if workURL == "" {
		// If WORK_URL is not set, do not proceed with the HTTP request
		return
	}

	// Construct the request for work cancellation
	request := models.WorkGenerate{
		Action: "work_cancel",
		Hash:   hash,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		klog.Errorf("Error marshalling work cancel request: %s", err)
		return
	}

	httpRequest, err := http.NewRequest(http.MethodPost, workURL, bytes.NewBuffer(requestBody))
	if err != nil {
		klog.Errorf("Error creating work cancel request: %s", err)
		return
	}
	httpRequest.Header.Add("Content-Type", "application/json")

	resp, err := Client.Do(httpRequest)
	if err != nil {
		klog.Errorf("Error sending work cancel request: %s", err)
		return
	}
	defer resp.Body.Close()
}
