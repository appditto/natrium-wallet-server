import requests
import redis
import rapidjson as json
import os

rdata = redis.StrictRedis(host=os.getenv(
    'REDIS_HOST', 'localhost'), port=6379, db=int(os.getenv('REDIS_DB', '2')))

dolartoday_price = 'https://s3.amazonaws.com/dolartoday/data.json'

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


dolartoday_bolivar()
print("DolarToday USD-VES:", rdata.hget("prices",
                                        "dolartoday:usd-ves").decode('utf-8'))
