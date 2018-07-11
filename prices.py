import redis, urllib3, certifi, socket, json, time, cfscrape
from exchanges.bitfinex import Bitfinex
from coinmarketcap import Market

#rblocks = redis.StrictRedis(host='localhost', port=6379, db=0)
#rwork = redis.StrictRedis(host='localhost', port=6379, db=1)
rdata = redis.StrictRedis(host='localhost', port=6379, db=2)

currency_list = [ "AUD", "BRL", "CAD", "CHF", "CLP", "CNY", "CZK", "DKK", "EUR", "GBP", "HKD", "HUF", "IDR", "ILS", "INR", "JPY", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "PKR", "PLN", "RUB", "SEK", "SGD", "THB", "TRY", "TWD", "USD", "ZAR" ]

def coinmarketcap():
	try:
		cmc = Market()
		for currency in currency_list:
			try:
				price_data = cmc.ticker('nano',limit=1,convert=currency.upper())[0]
				data_name = 'price_'+currency.lower()
				price_currency = price_data[data_name]
				print(rdata.hset("prices", "coinmarketcap:nano-"+currency.lower(), price_currency),"Coinmarketcap NANO-"+currency.upper(), price_currency)
			except:
				print("Failed to get price for NANO-"+currency.upper()+" Error")
		price_data = cmc.ticker('nano',limit=1,convert=currency.upper())[0]	
		price_btc = price_data['price_btc']	
		print(rdata.hset("prices", "coinmarketcap:nano-btc", price_btc),price_btc)
		print(rdata.hset("prices", "coinmarketcap:lastupdate",int(time.time())),int(time.time()))
	except:
		print("Failed to load from CoinMarketCap")

def bitfinex():
	try:
		bitfinex = Bitfinex().get_current_price()
		print(rdata.hset("prices","bitfinex:btc-usd",bitfinex))
		print(rdata.hset("prices","bitfinex:lastupdate",int(time.time())))
	except:
		print("Failed to load from Bitfinex")

bitfinex()
coinmarketcap()

print("Coinmarketcap NANO-USD:", rdata.hget("prices", "coinmarketcap:nano-usd").decode('utf-8'))
print("Coinmarketcap NANO-BTC:", rdata.hget("prices", "coinmarketcap:nano-btc").decode('utf-8'))
print("Last Update:          ", rdata.hget("prices", "coinmarketcap:lastupdate").decode('utf-8'))
print("Bitfinex BTC-USD:     ", rdata.hget("prices", "bitfinex:btc-usd").decode('utf-8'))
print("Last Update:          ", rdata.hget("prices", "bitfinex:lastupdate").decode('utf-8'))

