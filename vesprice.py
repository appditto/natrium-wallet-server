import requests, redis, json

#rblocks = redis.StrictRedis(host='localhost', port=6379, db=0)
#rwork = redis.StrictRedis(host='localhost', port=6379, db=1)
rdata = redis.StrictRedis(host='localhost', port=6379, db=2)

dolartoday_price='https://s3.amazonaws.com/dolartoday/data.json'

def dolartoday_bolivar():
    response = json.loads(requests.get(url=dolartoday_price).text)
    if "USD" not in response:
        print("Invalid response " + str(response))
        return
    bolivarprice = response['USD']['localbitcoin_ref']
    if bolivarprice is None:
        print("Couldn't find localbitcoin_ref price")
        return
    print(rdata.hset("prices", "dolartoday:usd-ves", bolivarprice),"DolarToday USD-VES", bolivarprice)

dolartoday_bolivar()
print("DolarToday USD-VES:", rdata.hget("prices", "dolartoday:usd-ves").decode('utf-8'))
