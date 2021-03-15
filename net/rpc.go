package net

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

type RPCClient struct {
	Url string
}

// Base request
func (client RPCClient) makeRequest(request interface{}) ([]byte, error) {
	requestBody, _ := json.Marshal(request)
	// HTTP post
	resp, err := http.Post(client.Url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		glog.Errorf("Error making RPC request %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	// Try to decode+deserialize
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Error decoding response body %s", err)
		return nil, err
	}
	return body, nil
}
