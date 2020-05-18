import requests
import redis
import rapidjson as json
import os

rdata = redis.StrictRedis(host=os.getenv(
    'REDIS_HOST', 'localhost'), port=6379, db=int(os.getenv('REDIS_DB', '2')))

dolartoday_price = 'https://s3.amazonaws.com/dolartoday/data.json'
dolarsi_ars_prices='https://www.dolarsi.com/api/api.php?type=valoresprincipales'

def dolartoday_bolivar():
    response = json.loads(requests.get(url=dolartoday_price).text)
    if "USD" not in response:
        print("Invalid response " + str(response))
        return
    bolivarprice = response['USD']['localbitcoin_ref']
    if bolivarprice is None:
        print("Couldn't find localbitcoin_ref price")
        return
    print(rdata.hset("prices", "dolartoday:usd-ves", bolivarprice),
          "DolarToday USD-VES", bolivarprice)

def dolarsi_ars():
    response = json.loads(requests.get(url=dolarsi_ars_prices).text)
    print(response)
    try:
        price_ars_raw = response[1]['casa']['venta']
    except KeyError:
        print("Invalid response " + str(response))
        return
    price_ars = price_ars_raw.replace('.','').replace(',','.')
    print(rdata.hset("prices", "dolarsi:usd-ars", price_ars),"DolarSi USD-ARS", price_ars)

dolartoday_bolivar()
print("DolarToday USD-VES:", rdata.hget("prices",
                                        "dolartoday:usd-ves").decode('utf-8'))
dolarsi_ars()
print("DolarSi USD-ARS:", rdata.hget("prices", "dolarsi:usd-ars").decode('utf-8'))
