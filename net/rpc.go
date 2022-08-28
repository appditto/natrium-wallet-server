package net

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/appditto/natrium-wallet-server/models"
	"k8s.io/klog/v2"
)

type RPCClient struct {
	Url string
}

// Base request
func (client *RPCClient) makeRequest(request interface{}) ([]byte, error) {
	requestBody, _ := json.Marshal(request)
	// HTTP post
	resp, err := http.Post(client.Url, "application/json", bytes.NewBuffer(requestBody))
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
	response, err := client.makeRequest(request)
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

	return responseMap, nil
}
