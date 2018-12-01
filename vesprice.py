openexchangerates_appid='' # Set your app id here

# Separate file because we want to run this conversion much less frequently due to openexchangerates free tier limitations

import redis, urllib3, certifi, socket, json, time, os, sys, requests

#rblocks = redis.StrictRedis(host='localhost', port=6379, db=0)
#rwork = redis.StrictRedis(host='localhost', port=6379, db=1)
rdata = redis.StrictRedis(host='localhost', port=6379, db=2)

openexchangerates_url=f'https://openexchangerates.org/api/latest.json?app_id={openexchangerates_appid}&show_alternative=true'

def openexchangerates():
    response = requests.get(url=openexchangerates_url).json()
    if 'rates' not in response:
        print("INvalid response " + str(response))
        return
    usdprice = rdata.hget("prices", "coingecko:nano-usd").decode('utf-8')
    if usdprice is None:
        print("Couldn't retrieve coingecko:nano-usd")
        return
    usdprice = float(usdprice)
    bolivarprice = response['rates']['VEF_BLKMKT']
    if bolivarprice is None:
        print("Couldn't find VEF_BLKMKT price")
        return
    bolivarprice = float(f'{bolivarprice:.2f}')
    conversion = str(usdprice * bolivarprice)
    print(rdata.hset("prices", "coingecko:nano-ves", conversion),"Coingecko NANO-VES", conversion)

openexchangerates()
print("Coingecko NANO-VES:", rdata.hget("prices", "coingecko:nano-ves").decode('utf-8'))
