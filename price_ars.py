import requests, redis, json

#rblocks = redis.StrictRedis(host='localhost', port=6379, db=0)
#rwork = redis.StrictRedis(host='localhost', port=6379, db=1)
rdata = redis.StrictRedis(host='localhost', port=6379, db=2)

coinmonitor_ars_prices='https://coinmonitor.info/data_ar.json'

#dolar = "DOLAR_d_CCL" # based on financial bonds, aka "contado con liquidación"
dolar = "DOLAR_d_blue" # based on street market, aka "dólar informal"
#dolar = "DOLAR_d_oficial" # defined arbitrarily by Argentine government
#dolar = "DOLAR_d_bitcoin" # based on BTC-ARS and BTC-USD prices

def coinmonitor_ars():
    response = json.loads(requests.get(url=coinmonitor_ars_prices).text)
    if dolar not in response:
        print("Invalid response " + str(response))
        return
    price_ars = response[dolar]
    print(rdata.hset("prices", "coinmonitor:usd-ars", price_ars),"CoinMonitor USD-ARS", price_ars)

coinmonitor_ars()
print("CoinMonitor USD-ARS:", rdata.hget("prices", "coinmonitor:usd-ars").decode('utf-8'))
