package net

import (
	"net/http"
	"os"
	"testing"

	"github.com/appditto/natrium-wallet-server/gql"
	"github.com/appditto/natrium-wallet-server/utils/mocks"
	"github.com/stretchr/testify/assert"
)

var RpcClient *RPCClient
var RpcClientBpowEnabled *RPCClient

func init() {
	// Mock redis client
	os.Setenv("MOCK_REDIS", "true")
	defer os.Unsetenv("MOCK_REDIS")
	// Mock HTTP client
	Client = &mocks.MockClient{}
	RpcClient = &RPCClient{Url: "http://localhost:123456"}
	RpcClientBpowEnabled = &RPCClient{Url: "http://localhost:123456", BpowClient: gql.NewBpowClient("http://localhost:123456", "secret", true)}
}

func TestAccountInfoRequest(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.AccountInfoResponse,
		}, nil
	}

	resp, err := RpcClient.MakeAccountInfoRequest("nano_3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3")
	assert.Equal(t, nil, err)
	assert.Equal(t, resp["frontier"], "80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F")
	assert.Equal(t, resp["open_block"], "0E3F07F7F2B8AEDEA4A984E29BFE1E3933BA473DD3E27C662EC041F6EA3917A0")
	assert.Equal(t, resp["balance"], "11999999999999999918751838129509869131")
	assert.Equal(t, resp["representative"], "nano_1gyeqc6u5j3oaxbe5qy1hyz3q745a318kh8h9ocnpan7fuxnq85cxqboapu5")
}

func TestGetReceivableCount(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.ReceivableResponse,
		}, nil
	}

	resp, err := RpcClient.GetReceivableCount("nano_3t6k35gi95xu6tergt6p69ck76ogmitsa8mnijtpxm9fkcm736xtoncuohr3", false)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, resp)
}

func TestBlockInfo(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.BlockInfoResponse,
		}, nil
	}

	resp, err := RpcClient.MakeBlockRequest("80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F")
	assert.Equal(t, nil, err)
	assert.Equal(t, "5606157000000000000000000000000000000", resp.Balance)
	assert.Equal(t, "true", resp.Confirmed)
	assert.Equal(t, "send", resp.Subtype)
	assert.Equal(t, "nano_1ipx847tk8o46pwxt5qjdbncjqcbwcc1rrmqnkztrfjy5k7z4imsrata9est", resp.Contents.Account)
	assert.Equal(t, "CE898C131AAEE25E05362F247760F8A3ACF34A9796A5AE0D9204E86B0637965E", resp.Contents.Previous)
	assert.Equal(t, "state", resp.Contents.Type)
}

func TestWorkGenerate(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.WorkGenerateResponse,
		}, nil
	}

	resp, err := RpcClient.WorkGenerate("80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F", 64)
	assert.Equal(t, nil, err)
	assert.Equal(t, "2b3d689bbcb21dca", resp)
}

func TestWorkGenerateBPOW(t *testing.T) {
	os.Setenv("BPOW_KEY", "1234")
	defer os.Unsetenv("BPOW_KEY")
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.BpowWorkGenerateResponse,
		}, nil
	}

	resp, err := RpcClientBpowEnabled.WorkGenerate("80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F", 64)
	assert.Equal(t, nil, err)
	assert.Equal(t, "00000001cce3db6c", resp)
}
