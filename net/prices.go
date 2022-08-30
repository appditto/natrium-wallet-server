package net

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/appditto/natrium-wallet-server/config"
	"github.com/appditto/natrium-wallet-server/database"
	"github.com/appditto/natrium-wallet-server/models"
	"k8s.io/klog/v2"
)

var CurrencyList = []string{
	"ARS", "AUD", "BRL", "BTC", "CAD", "CHF", "CLP", "CNY", "CZK", "DKK", "EUR", "GBP", "HKD", "HUF", "IDR", "ILS", "INR", "JPY", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "PKR", "PLN", "RUB", "SEK", "SGD", "THB", "TRY", "TWD", "USD", "ZAR", "SAR", "AED", "KWD", "UAH",
}

// Base request
func MakeGetRequest(url string) ([]byte, error) {
	// HTTP get
	resp, err := http.Get(url)
	if err != nil {
		klog.Errorf("Error making coingecko request %s", err)
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

func UpdateDolarTodayPrice() error {
	rawResp, err := MakeGetRequest(config.DOLARTODAY_URL)
	if err != nil {
		klog.Errorf("Error making dolar today request %s", err)
		return err
	}
	var dolarTodayResp models.DolarTodayResponse
	err = json.Unmarshal(rawResp, &dolarTodayResp)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		return err
	}

	if dolarTodayResp.Usd.LocalbitcoinRef > 0 {
		fmt.Printf("%s %f\n", "DolarToday USD-VES", dolarTodayResp.Usd.LocalbitcoinRef)
		database.GetRedisDB().Hset("prices", "dolartoday:usd-ves", dolarTodayResp.Usd.LocalbitcoinRef)
	} else {
		klog.Errorf("Error getting dolar today price")
		return errors.New("Dolartoday localbitcoin ref was 0")
	}
	return nil
}

func UpdateDolarSiPrice() error {
	rawResp, err := MakeGetRequest(config.DOLARSI_URL)
	if err != nil {
		klog.Errorf("Error making dolar today request %s", err)
		return err
	}
	var dolarsiResponse models.DolarsiResponse
	err = json.Unmarshal(rawResp, &dolarsiResponse)
	if err != nil {
		klog.Errorf("Error unmarshalling response %s", err)
		return err
	}

	if len(dolarsiResponse) < 2 {
		klog.Errorf("Error getting dolar si price")
		return errors.New("DolarSi response unexpected length")
	} else if dolarsiResponse[1].Casa.Venta == "" {
		klog.Errorf("Error getting dolar si price")
		return errors.New("DolarSi response price was empty")
	}
	price_ars := strings.ReplaceAll(dolarsiResponse[1].Casa.Venta, ".", "")
	price_ars = strings.ReplaceAll(price_ars, ",", ".")
	fmt.Printf("%s %s\n", "DolarSi USD-ARS", price_ars)
	database.GetRedisDB().Hset("prices", "dolarsi:usd-ars", price_ars)

	return nil
}

func UpdateNanoCoingeckoPrices() error {
	klog.Info("Updating nano prices\n")
	rawResp, err := MakeGetRequest(config.NANO_CG_URL)
	if err != nil {
		return err
	}
	var cgResp models.CoingeckoResponse
	if err := json.Unmarshal(rawResp, &cgResp); err != nil {
		klog.Errorf("Error unmarshalling coingecko response %v", err)
		return err
	}

	for _, currency := range CurrencyList {
		data_name := strings.ToLower(currency)
		if val, ok := cgResp.MarketData.CurrentPrice[data_name]; ok {
			fmt.Printf("%s %f\n", "Coingecko NANO-"+currency, val)
			database.GetRedisDB().Hset("prices", "coingecko:nano-"+data_name, val)
		} else {
			klog.Errorf("Error getting coingecko price for %s", data_name)
		}
	}

	usdPrice, err := database.GetRedisDB().Hget("prices", "coingecko:nano-usd")
	if err != nil {
		klog.Errorf("Error getting coingecko price for nano-usd %s", err)
		return err
	}
	usdPriceFloat, err := strconv.ParseFloat(usdPrice, 64)
	if err != nil {
		klog.Errorf("Error parsing coingecko price for nano-usd %s", err)
		return err
	}
	bolivarPrice, err := database.GetRedisDB().Hget("prices", "dolartoday:usd-ves")
	if err != nil {
		klog.Errorf("Error getting coingecko price for nano-usd %s", err)
		return err
	}
	bolivarPriceFloat, err := strconv.ParseFloat(bolivarPrice, 64)
	if err != nil {
		klog.Errorf("Error parsing coingecko price for nano-ves %s", err)
		return err
	}
	convertedves := usdPriceFloat * bolivarPriceFloat
	if err := database.GetRedisDB().Hset("prices", "coingecko:nano-ves", convertedves); err != nil {
		klog.Errorf("Error setting coingecko price for nano-ves %s", err)
		return err
	}
	fmt.Printf("%s %f\n", "Coingecko NANO-VES", convertedves)

	// # Convert to ARS
	arsPrice, err := database.GetRedisDB().Hget("prices", "dolarsi:usd-ars")
	if err != nil {
		klog.Errorf("Error getting coingecko price for nano-usd %s", err)
		return err
	}
	arsPriceFloat, err := strconv.ParseFloat(arsPrice, 64)
	if err != nil {
		klog.Errorf("Error parsing coingecko price for nano-ves %s", err)
		return err
	}
	convertedars := usdPriceFloat * arsPriceFloat
	if err := database.GetRedisDB().Hset("prices", "coingecko:nano-ars", convertedars); err != nil {
		klog.Errorf("Error setting coingecko price for nano-ves %s", err)
		return err
	}
	fmt.Printf("%s %f\n", "Coingecko NANO-ARS", convertedars)

	return nil
}

func UpdateBananoCoingeckoPrices() error {
	klog.Info("Updating BANANO prices\n")
	err := UpdateNanoCoingeckoPrices()
	if err != nil {
		return err
	}
	rawResp, err := MakeGetRequest(config.BANANO_CG_URL)
	if err != nil {
		return err
	}
	var cgResp models.CoingeckoResponse
	if err := json.Unmarshal(rawResp, &cgResp); err != nil {
		klog.Errorf("Error unmarshalling coingecko response %v", err)
		return err
	}

	for _, currency := range CurrencyList {
		data_name := strings.ToLower(currency)
		if val, ok := cgResp.MarketData.CurrentPrice[data_name]; ok {
			fmt.Printf("%s %f\n", "Coingecko BANANO-"+currency, val)
			database.GetRedisDB().Hset("prices", "coingecko:banano-"+data_name, val)
		} else {
			klog.Errorf("Error getting coingecko price for %s", data_name)
		}
	}

	usdPrice, err := database.GetRedisDB().Hget("prices", "coingecko:banano-usd")
	if err != nil {
		klog.Errorf("Error getting coingecko price for banano-usd %s", err)
		return err
	}
	usdPriceFloat, err := strconv.ParseFloat(usdPrice, 64)
	if err != nil {
		klog.Errorf("Error parsing coingecko price for banano-usd %s", err)
		return err
	}
	bolivarPrice, err := database.GetRedisDB().Hget("prices", "dolartoday:usd-ves")
	if err != nil {
		klog.Errorf("Error getting coingecko price for banano-usd %s", err)
		return err
	}
	bolivarPriceFloat, err := strconv.ParseFloat(bolivarPrice, 64)
	if err != nil {
		klog.Errorf("Error parsing coingecko price for banano-ves %s", err)
		return err
	}
	convertedves := usdPriceFloat * bolivarPriceFloat
	if err := database.GetRedisDB().Hset("prices", "coingecko:banano-ves", convertedves); err != nil {
		klog.Errorf("Error setting coingecko price for banano-ves %s", err)
		return err
	}
	fmt.Printf("%s %f\n", "Coingecko BANANO-VES", convertedves)

	// # Convert to ARS
	arsPrice, err := database.GetRedisDB().Hget("prices", "dolarsi:usd-ars")
	if err != nil {
		klog.Errorf("Error getting coingecko price for banano-usd %s", err)
		return err
	}
	arsPriceFloat, err := strconv.ParseFloat(arsPrice, 64)
	if err != nil {
		klog.Errorf("Error parsing coingecko price for banano-ves %s", err)
		return err
	}
	convertedars := usdPriceFloat * arsPriceFloat
	if err := database.GetRedisDB().Hset("prices", "coingecko:banano-ars", convertedars); err != nil {
		klog.Errorf("Error setting coingecko price for banano-ves %s", err)
		return err
	}
	fmt.Printf("%s %f\n", "Coingecko BANANO-ARS", convertedars)

	// Nano price
	// nanoprice = float(rdata.hget("prices", "coingecko:banano-btc")) / float(rdata.hget("prices", "coingecko:nano-btc"))
	// rdata.hset("prices", "coingecko:banano-nano", f"{nanoprice:.16f}")
	nanoprice, err := database.GetRedisDB().Hget("prices", "coingecko:nano-btc")
	if err != nil {
		klog.Errorf("Error getting price for nano-btc from redis %s", err)
		return err
	}
	nanopriceFloat, err := strconv.ParseFloat(nanoprice, 64)
	if err != nil {
		klog.Errorf("Error parsing price for nano-btc from redis %s", err)
		return err
	}
	nanoBanPrice := cgResp.MarketData.CurrentPrice["btc"] / nanopriceFloat
	if err := database.GetRedisDB().Hset("prices", "coingecko:banano-nano", nanoBanPrice); err != nil {
		klog.Errorf("Error setting price for banano-nano %s", err)
		return err
	}

	return nil
}
