#!/usr/bin/env python
import argparse
import asyncio
import ipaddress
import json
import logging
import os
import sys
import time
import uuid
import uvloop
from logging.handlers import TimedRotatingFileHandler, WatchedFileHandler

import aiofcm
from aiohttp import ClientSession, WSMessage, WSMsgType, log, web
from aioredis import create_redis

from rpc import RPC, allowed_rpc_actions
from util import Util

asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())

# Configuration arguments

parser = argparse.ArgumentParser(description="Natrium/Kalium Wallet Server")
parser.add_argument('-b', '--banano', action='store_true', help='Run for BANANO (Kalium-mode)', default=False)
parser.add_argument('--host', type=str, help='Host to listen on (e.g. 127.0.0.1)', default='127.0.0.1')
parser.add_argument('--path', type=str, help='(Optional) Path to run application on (for unix socket, e.g. /tmp/natriumapp.sock', default=None)
parser.add_argument('-p', '--port', type=int, help='Port to listen on', default=5076)
parser.add_argument('--log-file', type=str, help='Log file location', default='natriumcast.log')
options = parser.parse_args()

try:
    listen_host = str(ipaddress.ip_address(options.host))
    listen_port = int(options.port)
    log_file = options.log_file
    app_path = options.path
    if app_path is None:
        server_desc = f'on {listen_host} port {listen_port}'
    else:
        server_desc = f'on {app_path}'
    if options.banano:
        banano_mode = True
        print(f'Starting KALIUM Server (BANANO) {server_desc}')
    else:
        banano_mode = False
        print(f'Starting NATRIUM Server (NANO) {server_desc}')
except Exception:
    parser.print_help()
    sys.exit(0)

# Environment configuration

rpc_url = os.getenv('RPC_URL', 'http://[::1]:7076')
fcm_api_key = os.getenv('FCM_API_KEY', None)
fcm_sender_id = os.getenv('FCM_SENDER_ID', None)
debug_mode = True if os.getenv('DEBUG', 1) != 0 else False

# Objects

loop = asyncio.get_event_loop()
rpc = RPC(rpc_url, banano_mode)
util = Util(banano_mode)

# all currency conversions that are available
currency_list = ["BTC", "ARS", "AUD", "BRL", "CAD", "CHF", "CLP", "CNY", "CZK", "DKK", "EUR", "GBP", "HKD", "HUF", "IDR",
                 "ILS", "INR", "JPY", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "PKR", "PLN", "RUB", "SEK", "SGD",
                 "THB", "TRY", "TWD", "USD", "VES", "ZAR"]

# Push notifications

async def delete_fcm_token_for_account(account : str, token : str, r : web.Request):
    await r.app['rfcm'].delete(token)

async def update_fcm_token_for_account(account : str, token : str, r : web.Request, v2 : bool = False):
    """Store device FCM registration tokens in redis"""
    redisInst = r.app['rfcm'] if v2 else r.app['rdata']
    await set_or_upgrade_token_account_list(account, token, r, v2=v2)
    # Keep a list of tokens associated with this account
    cur_list = await redisInst.get(account)
    if cur_list is not None:
        cur_list = json.loads(cur_list.replace('\'', '"'))
    else:
        cur_list = {}
    if 'data' not in cur_list:
        cur_list['data'] = []
    if token not in cur_list['data']:
        cur_list['data'].append(token)
    await redisInst.set(account, json.dumps(cur_list))

async def get_or_upgrade_token_account_list(account : str, token : str, r : web.Request, v2 : bool = False) -> list:
    redisInst = r.app['rfcm'] if v2 else r.app['rdata']
    curTokenList = await redisInst.get(token)
    if curTokenList is None:
        return []
    else:
        try:
            curToken = json.loads(curTokenList)
            return curToken
        except Exception:
            curToken = curTokenList
            await redisInst.set(token, json.dumps([curToken]), expire=2592000)
            if account != curToken:
                return []
    return json.loads(await redisInst.get(token))

async def set_or_upgrade_token_account_list(account : str, token : str, r : web.Request, v2 : bool = False) -> list:
    redisInst = r.app['rfcm'] if v2 else r.app['rdata']
    curTokenList = await redisInst.get(token)
    if curTokenList is None:
        await redisInst.set(token, json.dumps([account]), expire=2592000) 
    else:
        try:
            curToken = json.loads(curTokenList)
            if account not in curToken:
                curToken.append(account)
                await redisInst.set(token, json.dumps(curToken), expire=2592000)
        except Exception as e:
            curToken = curTokenList
            await redisInst.set(token, json.dumps([curToken]), expire=2592000)
    return json.loads(await redisInst.get(token))

async def get_fcm_tokens(account : str, r : web.Request, v2 : bool = False) -> list:
    """Return list of FCM tokens that belong to this account"""
    redisInst = r.app['rfcm'] if v2 else r.app['rdata']
    tokens = await redisInst.get(account)
    if tokens is None:
        return []
    tokens = json.loads(tokens.replace('\'', '"'))
    # Rebuild the list for this account removing tokens that dont belong anymore
    new_token_list = {}
    new_token_list['data'] = []
    if 'data' not in tokens:
        return []
    for t in tokens['data']:
        account_list = await get_or_upgrade_token_account_list(account, t, r, v2=v2)
        if account not in account_list:
            continue
        new_token_list['data'].append(t)
    await redisInst.set(account, new_token_list)
    return new_token_list['data']

### END Utility functions

# Primary handler for all websocket connections
async def handle_user_message(r : web.Request, msg : WSMessage, ws : web.WebSocketResponse = None):
    """Process data sent by client"""
    address = util.get_request_ip(r)
    message = msg.data
    uid = ws.id if ws is not None else 0
    now = int(round(time.time() * 1000))
    if address in r.app['last_msg']:
        if (now - r.app['last_msg'][address]) < 10:
            log.server_logger.error('client messaging too quickly: %s ms; %s; %s; User-Agent: %s', str(
                now - r.app['last_msg'][address]), address, uid, str(
                r.headers.get('User-Agent')))
            return None
    r.app['last_msg'][address] = now
    log.server_logger.info('request; %s, %s, %s', message, address, uid)
    if message not in r.app['active_messages']:
        r.app['active_messages'].add(message)
    else:
        log.server_logger.error('request already active; %s; %s; %s', message, address, uid)
        return None
    ret = None
    try:
        request_json = json.loads(message)
        if request_json['action'] in allowed_rpc_actions:
            if 'request_id' in request_json:
                requestid = request_json['request_id']
            else:
                requestid = None

            # adjust counts so nobody can block the node with a huge request
            if 'count' in request_json:
                if (request_json['count'] < 0) or (request_json['count'] > 3500):
                    request_json['count'] = 3500

            # rpc: account_subscribe (only applies to websocket connections)
            if request_json['action'] == "account_subscribe" and ws is not None:
                # If account doesnt match the uuid self-heal
                resubscribe = True
                if 'uuid' in request_json:
                    # Perform multi-account upgrade if not already done
                    account = await r.app['rdata'].hget(request_json['uuid'], "account")
                    # No account for this uuid, first subscribe
                    if account is None:
                        resubscribe = False
                    else:
                        # If account isn't stored in list-format, modify it so it is
                        # If it already is, add this account to the list
                        try:
                            account_list = json.loads(account)
                            if 'account' in request_json and request_json['account'].lower() not in account_list:
                                account_list.append(request_json['account'].lower())
                                await r.app['rdata'].hset(request_json['uuid'], "account", json.dumps(account_list))
                        except Exception:
                            if 'account' in request_json and request_json['account'].lower() != account.lower():
                                resubscribe = False
                            else:
                                # Perform upgrade to list style
                                account_list = []
                                account_list.append(account.lower())
                                await r.app['rdata'].hset(request_json['uuid'], "account", json.dumps(account_list))
                # already subscribed, reconnect (websocket connections)
                if 'uuid' in request_json and resubscribe:
                    del r.app['clients'][uid]
                    uid = request_json['uuid']
                    ws.id = uid
                    r.app['clients'][uid] = ws
                    log.server_logger.info('reconnection request;' + address + ';' + uid)
                    try:
                        if 'currency' in request_json and request_json['currency'] in currency_list:
                            currency = request_json['currency']
                            r.app['cur_prefs'][uid] = currency
                            await r.app['rdata'].hset(uid, "currency", currency)
                        else:
                            setting = await r.app['rdata'].hget(uid, "currency")
                            if setting is not None:
                                r.app['cur_prefs'][uid] = setting
                            else:
                                r.app['cur_prefs'][uid] = 'usd'
                                await r.app['rdata'].hset(uid, "currency", 'usd')

                        # Get relevant account
                        account_list = json.loads(await r.app['rdata'].hget(uid, "account"))
                        if 'account' in request_json:
                            account = request_json['account']
                        else:
                            # Legacy connections
                            account = account_list[0]
                        if 'nano_' in account:
                            account_list.remove(account)
                            account_list.append(account.replace("nano_", "xrb_"))
                            account = account.replace('nano_', 'xrb_')
                            await r.app['rdata'].hset(uid, "account", json.dumps(account_list))
                        await rpc.rpc_reconnect(ws, r, account)
                        await r.app['rdata'].rpush("conntrack",
                                    str(float(time.time())) + ":" + uid + ":connect:" + address)
                        # Store FCM token for this account, for push notifications
                        if 'fcm_token' in request_json:
                            await update_fcm_token_for_account(account, request_json['fcm_token'], r)
                        elif 'fcm_token_v2' in request_json and 'notification_enabled' in request_json:
                            if request_json['notification_enabled']:
                                await update_fcm_token_for_account(account, request_json['fcm_token_v2'], r, v2=True)
                            else:
                                await delete_fcm_token_for_account(account, request_json['fcm_token_v2'], r) 
                    except Exception as e:
                        log.server_logger.error('reconnect error; %s; %s; %s', str(e), address, uid)
                        reply = {'error': 'reconnect error', 'detail': str(e)}
                        if requestid is not None: reply['request_id'] = requestid
                        ret = json.dumps(reply)
                # new user, setup uuid(or use existing if available) and account info
                else:
                    log.server_logger.info('subscribe request; %s; %s', util.get_request_ip(r), uid)
                    try:
                        if 'currency' in request_json and request_json['currency'] in currency_list:
                            currency = request_json['currency']
                        else:
                            currency = 'usd'
                            await rpc.rpc_subscribe(ws, r, request_json['account'].replace("nano_", "xrb_"), currency)
                            await r.app['rdata'].rpush(f"conntrack {str(float(time.time()))}:{uid}:connect:{address}")
                            # Store FCM token if available, for push notifications
                            if 'fcm_token' in request_json:
                                await update_fcm_token_for_account(request_json['account'], request_json['fcm_token'], r)
                            elif 'fcm_token_v2' in request_json and 'notification_enabled' in request_json:
                                if request_json['notification_enabled']:
                                    await update_fcm_token_for_account(request_json['account'], request_json['fcm_token_v2'], r, v2=True)
                                else:
                                    await delete_fcm_token_for_account(request_json['account'], request_json['fcm_token_v2'], r)
                    except Exception as e:
                        log.server_logger.error('subscribe error;%s;%s;%s', str(e), address, uid)
                        reply = {'error': 'subscribe error', 'detail': str(e)}
                        if requestid is not None: reply['request_id'] = requestid
                        ret = json.dumps(reply)
            elif request_json['action'] == "fcm_update":
                # Updating FCM token
                if 'fcm_token_v2' in request_json and 'account' in request_json and 'enabled' in request_json:
                    if request_json['enabled']:
                        await update_fcm_token_for_account(request_json['account'], request_json['fcm_token_v2'], r, v2=True)
                    else:
                        await delete_fcm_token_for_account(request_json['account'], request_json['fcm_token_v2'], r)
            # rpc: price_data
            elif request_json['action'] == "price_data":
                log.server_logger.info('price data request;%s;%s', util.get_request_ip(r), uid)
                try:
                    if request_json['currency'].upper() in currency_list:
                        try:
                            price = await r.app['rdata'].hget("prices",
                                                "coingecko:nano-" + request_json['currency'].lower())
                            reply = json.dumps({
                                'currency': request_json['currency'].lower(),
                                'price': str(price)
                            })
                            ret = reply
                        except Exception:
                            log.server_logger.error(
                                'price data error, unable to get price;%s;%s', util.get_request_ip(r), uid)
                            ret = json.dumps({
                                'error':'price data error - unable to get price'
                            })
                    else:
                        log.server_logger.error(
                            'price data error, unknown currency;%s;%s', util.get_request_ip(r), uid)
                        ret = json.dumps({
                            'error':'unknown currency'
                        })
                except Exception as e:
                    log.server_logger.error('price data error;%s;%s;%s', str(e), util.get_request_ip(r), uid)
                    ret = json.dumps({
                        'error':'price data error',
                        'details':str(e)
                    })
            # rpc: account_check
            elif request_json['action'] == "account_check":
                log.server_logger.info('account check request;%s;%s', util.get_request_ip(r), uid)
                try:
                    response = await rpc.rpc_accountcheck(r, uid, request_json['account'])
                    ret = json.dumps(response)
                except Exception as e:
                    log.server_logger.error('account check error;%s;%s;%s', str(e), util.get_request_ip(r), uid)
                    ret = json.dumps({
                        'error': 'account check error',
                        'detail': str(e)
                    })
            # rpc: work_generate
            elif request_json['action'] == "work_generate":
                """
                If we care about blocking work to specific client versions we can so here:
                if r.headers.get('X-Client-Version') is None:
                    xcver = 0
                else:
                    xcver = int(r.headers.get('X-Client-Version'))
                """
                try:
                    reply = await rpc.work_defer(r, uid, request_json)
                    ret = json.dumps(reply)
                except Exception as e:
                    log.server_logger.error('work RPC error; %s;%s;%s;User-Agent:%s',
                        str(e), util.get_request_ip(r), uid, r.headers.get('User-Agent'))
                    ret = json.dumps({
                        'error':'work RPC error',
                        'detail':str(e)
                    })
            # rpc: process
            elif request_json['action'] == "process":
                try:
                    do_work = False
                    if 'do_work' in request_json and request_json['do_work'] == True:
                        do_work = True
                    reply = await rpc.process_defer(r, uid, json.loads(request_json['block']), do_work)
                    if reply is None:
                        raise Exception
                    ret = json.dumps(reply)
                except Exception as e:
                    log.server_logger.error('process rpc error;%s;%s;%s;User-Agent:%s',
                        str(e), util.get_request_ip(r), uid, str(r.headers.get('User-Agent')))
                    ret = json.dumps({
                        'error':'process rpc error',
                        'detail':str(e)
                    })
            # rpc: pending
            elif request_json['action'] == "pending":
                try:
                    reply = await rpc.pending_defer(r, uid, request_json)
                    if reply is None:
                        raise Exception
                    ret = json.dumps(reply)
                except Exception as e:
                    log.server_logger.error('pending rpc error;%s;%s;%s;User-Agent:%s', str(
                        e), util.get_request_ip(r), uid, str(r.headers.get('User-Agent')))
                    ret = json.dumps({
                        'error':'pending rpc error',
                        'detail':str(e)
                    })
            elif request_json['action'] == 'account_history':
                if await r.app['rdata'].hget(uid, "account") is None:
                    await r.app['rdata'].hset(uid, "account", json.dumps([request_json['account']]))
                try:
                    response = await rpc.json_post(request_json)
                    if response is None:
                        raise Exception
                    ret = json.dumps(response)
                except Exception as e:
                    log.server_logger.error('rpc error;%s;%s;%s', str(e), util.get_request_ip(r), uid)
                    ret = json.dumps({
                        'error':'account_history rpc error',
                        'detail': str(e)
                    })
            # rpc: fallthrough and error catch
            else:
                try:
                    response = await rpc.json_post(request_json)
                    if response is None:
                        raise Exception
                    ret = json.dumps(response)
                except Exception as e:
                    log.server_logger.error('rpc error;%s;%s;%s', str(e), util.get_request_ip(r), uid)
                    ret = json.dumps({
                        'error':'rpc error',
                        'detail': str(e)
                    })
    except Exception as e:
        log.server_logger.error('uncaught error;%s;%s;%s', sys.exc_info(), util.get_request_ip(r), uid)
        ret = json.dumps({
            'error':'general error',
            'detail':sys.exc_info()
        })
    finally:
        r.app['active_messages'].remove(message)
        return ret

async def websocket_handler(r : web.Request):
    """Handler for websocket connections and messages"""

    ws = web.WebSocketResponse()
    await ws.prepare(r)

    # Connection Opened
    ws.id = str(uuid.uuid4())
    r.app['clients'][ws.id] = ws
    log.server_logger.info('new connection;%s;%s;User-Agent:%s', util.get_request_ip(r), ws.id, str(
        r.headers.get('User-Agent')))

    try:
        async for msg in ws:
            if msg.type == WSMsgType.TEXT:
                if msg.data == 'close':
                    await ws.close()
                else:
                    reply = await handle_user_message(r, msg, ws=ws)
                    if reply is not None:
                        log.server_logger.debug('Sending response %s to %s', reply, util.get_request_ip(r))
                        await ws.send_str(reply)
            elif msg.type == WSMsgType.CLOSE:
                log.server_logger.info('WS Connection closed normally')
                break
            elif msg.type == WSMsgType.ERROR:
                log.server_logger.info('WS Connection closed with error %s', ws.exception())
                break

        log.server_logger.info('WS connection closed normally')
    except Exception:
        log.server_logger.exception('WS Closed with exception')
    finally:
        if ws.id in r.app['clients']:
            del r.app['clients'][ws.id]
        await ws.close()

    return ws

async def callback(r : web.Request):
    request_json = await r.json()
    hash = request_json['hash']
    log.server_logger.debug(f"callback received {hash}")
    request_json['block'] = json.loads(request_json['block'])

    if request_json['block']['type'] == 'state':
        link = request_json['block']['link_as_account']
        if r.app['subscriptions'].get(link):
            log.server_logger.info("Pushing to clients %s", str(r.app['subscriptions'][link]))
            for sub in r.app['subscriptions'][link]:
                r.app['clients'][sub].send_str(json.dumps(request_json))
        # Push FCM notification if this is a send
        if fcm_api_key is None:
            return
        fcm_tokens = set(await get_fcm_tokens(link, r))
        fcm_tokens_v2 = set(await get_fcm_tokens(link, r, v2=True))
        if (fcm_tokens is None or len(fcm_tokens) == 0) and (fcm_tokens_v2 is None or len(fcm_tokens_v2) == 0):
            return
        message = {
            "action":"block",
            "hash":request_json['block']['previous']
        }
        response = await rpc.json_post(message)
        if response is None:
            return
        # See if this block was already pocketed
        cached_hash = await r.app['rdata'].get(f"link_{hash}")
        if cached_hash is not None:
            return
        prev_data = response
        prev_data = prev_data['contents'] = json.loads(prev_data['contents'])
        prev_balance = int(prev_data['contents']['balance'])
        cur_balance = int(request_json['block']['balance'])
        send_amount = prev_balance - cur_balance
        if send_amount >= 1000000000000000000000000:
            # This is a send, push notifications
            fcm = aiofcm.FCM(fcm_sender_id, fcm_api_key)
            # Send notification with generic title, send amount as body. App should have localizations and use this information at its discretion
            for t in fcm_tokens:
                message = aiofcm.Message(
                            device_token=t,
                            data = {
                                "amount": str(send_amount)
                            },
                            priority=aiofcm.PRIORITY_HIGH
                )
                await fcm.send_message(message)
            notification_title = f"Received {util.raw_to_nano(send_amount)} {'NANO' if not banano_mode else 'BANANO'}"
            notification_body = f"Open {'Natrium' if not banano_mode else 'Kalium'} to view this transaction."
            for t2 in fcm_tokens_v2:
                message = aiofcm.Message(
                    device_token = t2,
                    notification = {
                        "title":notification_title,
                        "body":notification_body,
                        "sound":"default",
                        "tag":link
                    },
                    data = {
                        "click_action": "FLUTTER_NOTIFICATION_CLICK",
                        "account": link
                    },
                    priority=aiofcm.PRIORITY_HIGH
                )
                await fcm.send_message(message)
    elif r.app['subscriptions'].get(request_json['account']):
        log.server_logger.info("Pushing to client %s", str(r.app['subscriptions'][request_json['account']]))
        for sub in r.app['subscriptions'][request_json['account']]:
            r.app['clients'][sub].send_str(json.dumps(request_json))

async def send_prices(app):
    """Send price updates to connected clients once per minute"""
    while True:
        # global active_work
        # active_work = set()
        # empty out this set periodically, to ensure clients dont somehow get stuck when an error causes their
        # work not to return
        if len(app['clients']):
            log.server_logger.info('pushing price data to %d connections', len(app['clients']))
            btc = float(await app['rdata'].hget("prices", "coingecko:nano-btc"))
            for client in app['clients']:
                try:
                    try:
                        currency = app['cur_prefs'][client]
                    except Exception:
                        currency = 'usd'
                    price = float(await app['rdata'].hget("prices", "coingecko:nano-" + currency.lower()))

                    app['clients'].send_str(
                        '{"currency":"' + currency.lower() + '","price":' + str(price) + ',"btc":' + str(btc) + '}')
                except Exception:
                    log.server_logger.exception('error pushing prices for client %s', client)
        await asyncio.sleep(60)

async def init_app():
    """ Initialize the main application instance and return it"""
    async def close_redis(app):
        """Close redis connections"""
        log.server_logger.info('Closing redis connections')
        app['rfcm'].close()
        app['rdata'].close()

    async def open_redis(app):
        """Open redis connections"""
        log.server_logger.info("Opening redis connections")
        app['rfcm'] = await create_redis(('localhost', 6379),
                                                db=1, encoding='utf-8')
        app['rdata'] = await create_redis(('localhost', 6379),
                                                db=2, encoding='utf-8')
        # Global vars
        app['clients'] = {} # Keep track of connected clients
        app['last_msg'] = {} # Last time a client has sent a message
        app['active_messages'] = set() # Avoid duplicate messages from being processes simultaneously
        app['cur_prefs'] = {} # Client currency preferences
        app['subscriptions'] = {} # Store subscription UUIDs, this is used for targeting callback accounts
        app['active_work'] = set() # Keep track of active work requests to prevent duplicates

    # Setup logger
    if debug_mode:
        logging.basicConfig(level='DEBUG')
    else:
        root = log.server_logger.getLogger()
        logging.basicConfig(level='INFO')
        handler = WatchedFileHandler(log_file)
        formatter = logging.Formatter("%(asctime)s;%(levelname)s;%(message)s", "%Y-%m-%d %H:%M:%S %z")
        handler.setFormatter(formatter)
        root.addHandler(handler)
        root.addHandler(TimedRotatingFileHandler(log_file, when="d", interval=1, backupCount=100))        

    app = web.Application()
    app.add_routes([web.get('/', websocket_handler)]) # All WS requests
    app.add_routes([web.post('/callback', callback)]) # HTTP Callback from node
    #app.add_routes([web.post('/callback', callback)])
    app.on_startup.append(open_redis)
    app.on_shutdown.append(close_redis)

    return app

app = loop.run_until_complete(init_app())

def main():
    """Main application loop"""

    # Periodic price job
    price_task = loop.create_task(send_prices(app))

    # Start web/ws server
    async def start():
        runner = web.AppRunner(app)
        await runner.setup()
        if app_path is not None:
            site = web.UnixSite(runner, app_path)
        else:
            site = web.TCPSite(runner, listen_host, listen_port)
        await site.start()

    async def end():
        await app.shutdown()

    loop.run_until_complete(start())

    # Main program
    try:
        loop.run_forever()
    except KeyboardInterrupt:
        pass
    finally:
        price_task.cancel()
        loop.run_until_complete(end())

    loop.close()

if __name__ == "__main__":
    main()
