package net

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

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
	klog.Errorf("Making request %s", string(requestBody))
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
	if client.BpowClient != nil {
		res, err := client.BpowClient.WorkGenerate(hash, difficultyMultiplier)
		if err != nil || res == "" {
			klog.Infof("Error generating work with BPOW %s", err)
			if utils.GetEnv("WORK_URL", "") == "" {
				return "", err
			}
		}
		return res, nil
	}

	// Base send difficulty
	// Nano has 2 difficulties, higher for send, lower for receive
	// Don't bother deriving it since it can only be one of two values
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
	// HTTP post
	httpRequest, err := http.NewRequest(http.MethodPost, utils.GetEnv("WORK_URL", ""), bytes.NewBuffer(requestBody))
	if err != nil {
		klog.Errorf("Error making work gen request %s", err)
		return "", err
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	resp, err := Client.Do(httpRequest)
	if err != nil {
		klog.Errorf("Error processing work gen request %s", err)
		return "", err
	}
	defer resp.Body.Close()
	// Try to decode+deserialize
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Error decoding work response body %s", err)
		return "", err
	}

	var workResp models.WorkResponse
	err = json.Unmarshal(body, &workResp)
	if err != nil {
		klog.Errorf("Error unmarshalling work_gen response %s", err)
		return "", err
	}

	return workResp.Work, nil
}
