import redis
import json
import os
import time
import sys
import requests

rdata = redis.StrictRedis(host=os.getenv('REDIS_HOST', 'localhost'), port=6379, db=int(os.getenv('REDIS_DB', '2')))

currency_list = ["ARS", "AUD", "BRL", "BTC", "CAD", "CHF", "CLP", "CNY", "CZK", "DKK", "EUR", "GBP", "HKD", "HUF", "IDR", "ILS", "INR",
                 "JPY", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "PKR", "PLN", "RUB", "SEK", "SGD", "THB", "TRY", "TWD", "USD", "ZAR", "SAR", "AED", "KWD"]

coingecko_url = 'https://api.coingecko.com/api/v3/coins/nano?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false&sparkline=false'


def coingecko():
    response = requests.get(url=coingecko_url).json()
    if 'market_data' not in response:
        return
    for currency in currency_list:
        try:
            data_name = currency.lower()
            price_currency = response['market_data']['current_price'][data_name]
            print(rdata.hset("prices", "coingecko:nano-"+data_name,
                             price_currency), "Coingecko NANO-"+currency, price_currency)
        except Exception:
            exc_type, exc_obj, exc_tb = sys.exc_info()
            print('exception', exc_type, exc_obj, exc_tb.tb_lineno)
            print("Failed to get price for NANO-"+currency.upper()+" Error")
    # Convert to VES
    usdprice = float(rdata.hget(
        "prices", "coingecko:nano-usd").decode('utf-8'))
    bolivarprice = float(rdata.hget(
        "prices", "dolartoday:usd-ves").decode('utf-8'))
    convertedves = usdprice * bolivarprice
    rdata.hset("prices", "coingecko:nano-ves", convertedves)
    print("Coingecko NANO-VES", rdata.hget("prices",
                                           "coingecko:nano-ves").decode('utf-8'))
    print(rdata.hset("prices", "coingecko:lastupdate",
                     int(time.time())), int(time.time()))


coingecko()

print("Coingecko NANO-USD:", rdata.hget("prices",
                                        "coingecko:nano-usd").decode('utf-8'))
print("Coingecko NANO-BTC:", rdata.hget("prices",
                                        "coingecko:nano-btc").decode('utf-8'))
print("Last Update:          ", rdata.hget(
    "prices", "coingecko:lastupdate").decode('utf-8'))
