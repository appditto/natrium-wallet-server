import requests, redis, json

#rblocks = redis.StrictRedis(host='localhost', port=6379, db=0)
#rwork = redis.StrictRedis(host='localhost', port=6379, db=1)
rdata = redis.StrictRedis(host='localhost', port=6379, db=2)

dolarsi_ars_prices='https://www.dolarsi.com/api/api.php?type=valoresprincipales'

dolar_blue = "1" # based on the market, aka "dólar informal"

def dolarsi_ars():
    response = json.loads(requests.get(url=dolarsi_ars_prices).text)
    try:
        price_ars_raw = response[dolar_blue]['casa']['venta']
    except KeyError:
        print("Invalid response " + str(response))
        return
    price_ars = price_ars_raw.replace('.','').replace(',','.')
    print(rdata.hset("prices", "dolarsi:usd-ars", price_ars),"DolarSi USD-ARS", price_ars)

dolarsi_ars()
print("DolarSi USD-ARS:", rdata.hget("prices", "dolarsi:usd-ars").decode('utf-8'))
