package net

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/utils/mocks"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Mock redis client
	os.Setenv("MOCK_REDIS", "true")
	defer os.Unsetenv("MOCK_REDIS")
	// Mock HTTP client
	Client = &mocks.MockClient{}
}

func TestDolarTodayPrice(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 304,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.DolarTodayResponse,
		}, nil
	}

	err := UpdateDolarTodayPrice()
	assert.Equal(t, nil, err)
	dolarToday, err := database.GetRedisDB().Hget("prices", "dolartoday:usd-ves")
	assert.Equal(t, nil, err)
	assert.Equal(t, "8.15", dolarToday)
}

func TestDolarSiPrice(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.DolarSiResponse,
		}, nil
	}

	err := UpdateDolarSiPrice()
	assert.Equal(t, nil, err)
	dolarSi, err := database.GetRedisDB().Hget("prices", "dolarsi:usd-ars")
	assert.Equal(t, nil, err)
	assert.Equal(t, "290.00", dolarSi)
}

func TestUpdateNanoPrice(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.NanoCoingeckoResponse,
		}, nil
	}

	database.GetRedisDB().Hset("prices", "dolarsi:usd-ars", "290.00")
	database.GetRedisDB().Hset("prices", "dolartoday:usd-ves", "8.15")

	err := UpdateNanoCoingeckoPrices()
	assert.Equal(t, nil, err)

	for _, v := range CurrencyList {
		price, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:nano-%s", strings.ToLower(v)))
		assert.Equal(t, nil, err)
		switch v {
		case "ARS":
			assert.Equal(t, "260.84224", price)
		case "AUD":
			assert.Equal(t, "1.31", price)
		case "BRL":
			assert.Equal(t, "4.67", price)
		case "BTC":
			assert.Equal(t, "0.00004494", price)
		case "CAD":
			assert.Equal(t, "1.18", price)
		case "CHF":
			assert.Equal(t, "0.877073", price)
		case "CLP":
			assert.Equal(t, "806.45", price)
		case "CNY":
			assert.Equal(t, "6.2", price)
		case "CZK":
			assert.Equal(t, "21.92", price)
		case "DKK":
			assert.Equal(t, "6.65", price)
		case "EUR":
			assert.Equal(t, "0.894673", price)
		case "GBP":
			assert.Equal(t, "0.773667", price)
		case "HKD":
			assert.Equal(t, "7.06", price)
		case "HUF":
			assert.Equal(t, "358.53", price)
		case "IDR":
			assert.Equal(t, "13360.1", price)
		case "ILS":
			assert.Equal(t, "2.99", price)
		case "INR":
			assert.Equal(t, "71.49", price)
		case "JPY":
			assert.Equal(t, "124.76", price)
		case "KRW":
			assert.Equal(t, "1206.55", price)
		case "MXN":
			assert.Equal(t, "18.09", price)
		case "MYR":
			assert.Equal(t, "4.03", price)
		case "NOK":
			assert.Equal(t, "8.93", price)
		case "NZD":
			assert.Equal(t, "1.47", price)
		case "PHP":
			assert.Equal(t, "50.57", price)
		case "PKR":
			assert.Equal(t, "198.02", price)
		case "PLN":
			assert.Equal(t, "4.23", price)
		case "RUB":
			assert.Equal(t, "54.42", price)
		case "SEK":
			assert.Equal(t, "9.58", price)
		case "SGD":
			assert.Equal(t, "1.26", price)
		case "THB":
			assert.Equal(t, "32.89", price)
		case "TRY":
			assert.Equal(t, "16.37", price)
		case "TWD":
			assert.Equal(t, "27.35", price)
		case "USD":
			assert.Equal(t, "0.899456", price)
		case "ZAR":
			assert.Equal(t, "15.38", price)
		case "SAR":
			assert.Equal(t, "3.38", price)
		case "AED":
			assert.Equal(t, "3.3", price)
		case "KWD":
			assert.Equal(t, "0.27726", price)
		case "UAH":
			assert.Equal(t, "33.17", price)
		case "VES":
			assert.Equal(t, "7.330566", price)
		}
	}
}

func TestUpdateBananoPrice(t *testing.T) {
	// Simulate response
	mocks.GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: mocks.BananoCoingeckoResponse,
		}, nil
	}

	database.GetRedisDB().Hset("prices", "dolarsi:usd-ars", "290.00")
	database.GetRedisDB().Hset("prices", "dolartoday:usd-ves", "8.15")
	database.GetRedisDB().Hset("prices", "coingecko:nano-btc", "0.75")
	err := UpdateBananoCoingeckoPrices()
	assert.Equal(t, nil, err)

	price, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:banano-nano"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "0.0000003892413333333334", price)
	for _, v := range CurrencyList {
		price, err := database.GetRedisDB().Hget("prices", fmt.Sprintf("coingecko:banano-%s", strings.ToLower(v)))
		assert.Equal(t, nil, err)
		switch v {
		case "ARS":
			assert.Equal(t, "1.6935883999999999", price)
		case "AUD":
			assert.Equal(t, "0.00852173", price)
		case "BRL":
			assert.Equal(t, "0.03034209", price)
		case "BTC":
			assert.Equal(t, "0.000000291931", price)
		case "CAD":
			assert.Equal(t, "0.0076608", price)
		case "CHF":
			assert.Equal(t, "0.00569591", price)
		case "CLP":
			assert.Equal(t, "5.24", price)
		case "CNY":
			assert.Equal(t, "0.04023965", price)
		case "CZK":
			assert.Equal(t, "0.1424", price)
		case "DKK":
			assert.Equal(t, "0.04321482", price)
		case "EUR":
			assert.Equal(t, "0.00581072", price)
		case "GBP":
			assert.Equal(t, "0.00502146", price)
		case "HKD":
			assert.Equal(t, "0.04583988", price)
		case "HUF":
			assert.Equal(t, "2.33", price)
		case "IDR":
			assert.Equal(t, "86.75", price)
		case "ILS":
			assert.Equal(t, "0.01943407", price)
		case "INR":
			assert.Equal(t, "0.464163", price)
		case "JPY":
			assert.Equal(t, "0.81002", price)
		case "KRW":
			assert.Equal(t, "7.84", price)
		case "MXN":
			assert.Equal(t, "0.117546", price)
		case "MYR":
			assert.Equal(t, "0.02613673", price)
		case "NOK":
			assert.Equal(t, "0.058018", price)
		case "NZD":
			assert.Equal(t, "0.00952738", price)
		case "PHP":
			assert.Equal(t, "0.328369", price)
		case "PKR":
			assert.Equal(t, "1.29", price)
		case "PLN":
			assert.Equal(t, "0.02744881", price)
		case "RUB":
			assert.Equal(t, "0.353317", price)
		case "SEK":
			assert.Equal(t, "0.06224", price)
		case "SGD":
			assert.Equal(t, "0.00815694", price)
		case "THB":
			assert.Equal(t, "0.213309", price)
		case "TRY":
			assert.Equal(t, "0.106292", price)
		case "TWD":
			assert.Equal(t, "0.177564", price)
		case "USD":
			assert.Equal(t, "0.00583996", price)
		case "ZAR":
			assert.Equal(t, "0.099762", price)
		case "SAR":
			assert.Equal(t, "0.0219527", price)
		case "AED":
			assert.Equal(t, "0.02145011", price)
		case "KWD":
			assert.Equal(t, "0.00180018", price)
		case "UAH":
			assert.Equal(t, "0.215376", price)
		case "VES":
			assert.Equal(t, "0.047596", price)
		}
	}
}
