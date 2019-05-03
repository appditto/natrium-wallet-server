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
from logging.handlers import WatchedFileHandler, TimedRotatingFileHandler

from aioredis import create_redis
from aiohttp import ClientSession, log, web, WSMsgType

# Configuration arguments

parser = argparse.ArgumentParser(description="Natrium/Kalium Wallet Server")
parser.add_argument('-b', '--banano', action='store_true', help='Run for BANANO (Kalium-mode)', default=False)
parser.add_argument('-l', '--host', type=str, help='Host to listen on (e.g. 127.0.0.1)', default='127.0.0.1')
parser.add_argument('-p', '--port', type=int, help='Port to listen on', default=5076)
options = parser.parse_args()

try:
    listen_host = str(ipaddress.ip_address(options.host))
    listen_port = int(options.port)
    if options.banano:
        banano_mode = True
        print(f'Starting KALIUM Server (BANANO) on {listen_host} port {listen_port}')
    else:
        banano_mode = False
        print(f'Starting NATRIUM Server (NANO) on {listen_host} port {listen_port}')
except Exception:
    parser.print_help()
    sys.exit(0)

# Environment configuration

rpc_url = os.getenv('RPC_URL', 'http://[::1]:7076')
fcm_api_key = os.getenv('FCM_API_KEY')
fcm_sender_id = os.getenv('FCM_SENDER_ID')
debug_mode = True if os.getenv('DEBUG', 1) != 0 else False

loop = asyncio.get_event_loop()

# whitelisted commands, disallow anything used for local node-based wallet as we may be using multiple back ends
allowed_rpc_actions = ["account_balance", "account_block_count", "account_check", "account_info", "account_history",
                       "account_representative", "account_subscribe", "account_weight", "accounts_balances",
                       "accounts_frontiers", "accounts_pending", "available_supply", "block", "block_hash",
                       "block_create", "blocks", "block_info", "blocks_info", "block_account", "block_count", "block_count_type",
                       "chain", "delegators", "delegators_count", "frontiers", "frontier_count", "history",
                       "key_expand", "process", "representatives", "republish", "peers", "version", "pending",
                       "pending_exists", "price_data", "work_generate", "fcm_update"]

# all currency conversions that are available
currency_list = ["BTC", "ARS", "AUD", "BRL", "CAD", "CHF", "CLP", "CNY", "CZK", "DKK", "EUR", "GBP", "HKD", "HUF", "IDR",
                 "ILS", "INR", "JPY", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "PKR", "PLN", "RUB", "SEK", "SGD",
                 "THB", "TRY", "TWD", "USD", "VES", "ZAR"]


# Utility functions
def get_request_ip(r):
    host = r.headers.get('X-FORWARDED-FOR',None)
    if host is None:
        peername = r.transport.get_extra_info('peername')
        if peername is not None:
            host, port = peername
    return host

async def json_post(request_json, timeout=30):
    try:
        async with ClientSession() as session:
            async with session.post(rpc_url, json=request_json, timeout=timeout) as resp:
                return await resp.json(content_type=None)
    except Exception:
        log.server_logger.exception()
        return None

async def get_pending_count(r, account, uid = 0):
    """This returns how many pending blocks an account has, up to 51, for anti-spam measures"""
    message = {
        "action":"pending",
        "account":account,
        "threshold":str(10**24),
        "count":51
    }
    log.server_logger.info('sending get_pending_count; %s; %s', get_request_ip(r), uid)
    response = await json_post(message)
    if response is None or 'blocks' not in response:
        return 0
    log.server_logger.debug('received response for pending %s', json.dumps(response))        
    return len(response['blocks'])

# Push notifications

async def delete_fcm_token_for_account(account, token, r):
    await r.app['rfcm'].delete(token)

async def update_fcm_token_for_account(account, token, r, v2=False):
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

async def get_or_upgrade_token_account_list(account, token, r, v2=False):
    redisInst = r.app['rfcm'] if v2 else r.app['rdata']
    curTokenList = await redisInst.get(token)
    if curTokenList is None:
        return []
    else:
        try:
            curToken = json.loads(curTokenList)
            return curToken
        except Exception as e:
            curToken = curTokenList
            await redisInst.set(token, json.dumps([curToken]), expire=2592000)
            if account != curToken:
                return []
    return json.loads(await redisInst.get(token))

async def set_or_upgrade_token_account_list(account, token, r, v2=False):
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

async def get_fcm_tokens(account, r, v2=False):
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
        account_list = await get_or_upgrade_token_account_list(account, t, v2=v2)
        if account not in account_list:
            continue
        new_token_list['data'].append(t)
    await redisInst.set(account, new_token_list)
    return new_token_list['data']

### END Utility functions

async def rpc_reconnect(ws, r, account):
    """When a websocket connection sends a subscribe request, do this reconnection step"""
    log.server_logger.info('reconnecting;' + get_request_ip(r) + ';' + ws.id)

    rpc = {
        "action":"account_info",
        "account":"account",
        "pending":True,
        "representative": True
    }
    log.server_logger.info('sending account_info %s', account)
    response = await json_post(rpc)

    if response is None or 'frontier' not in response:
        log.server_logger.error('reconnect error; %s ; %s', get_request_ip(r), ws.id)
        ws.send_str('{"error":"reconnect error"}')
    else:
        log.server_logger.debug('received response for account_info %s', json.dumps(response))     
        if account in r.app['subscriptions']:
            r.app['subscriptions'][account].add(ws.id)
        else:
            r.app['subscriptions'][account] = set()
            r.app['subscriptions'][account].add(ws.id)
        r.app['cur_prefs'][handler.id] = await r.app['rdata'].hget(ws.id, "currency")
        await r.app['rdata'].hset(ws.id, "last-connect", float(time.time()))
        price_cur = await r.app['rdata'].hget("prices", "coingecko:nano-" + r.app['cur_prefs'][ws.id].lower())
        price_btc = await r.app['rdata'].hget("prices", "coingecko:nano-btc")
        response['currency'] = r.app['cur_prefs'][ws.id].lower()
        response['price'] = float(price_cur)
        response['btc'] = float(price_btc)
        response['pending_count'] = await get_pending_count(r, account, uid = ws.id)
        response = json.dumps(response)

        log.server_logger.info(
            'reconnect response sent ; %s bytes; %s; %s', str(len(response)), get_request_ip(r), ws.id)

        await ws.send_str(response)

async def rpc_subscribe(ws, r, account, currency):
    """Clients subscribing for the first time"""
    logging.info('subscribing;%s;%s', get_request_ip(r), ws.id)

    rpc = {
        'action':'account_info',
        'account':account,
        'pending':True,
        'representative':True
    }
    log.server_logger.info('sending account_info;%s;%s', get_request_ip, ws.id)
    response = await json_post(rpc)

    if response is None or 'frontier' not in response:
        log.server_logger.error('reconnect error; %s ; %s', get_request_ip(r), ws.id)
        ws.send_str('{"error":"subscribe error"}')
    else:
        log.server_logger.debug('received response for account_info %s', json.dumps(response))     
        if account in r.app['subscriptions']:
            r.app['subscriptions'][account].add(ws.id)
        else:
            r.app['subscriptions'][account] = set()
            r.app['subscriptions'][account].add(ws.id)
        await r.app['rdata'].hset(ws.id, "account", json.dumps([account]))
        r.app['cur_prefs'][ws.id] = currency
        await r.app['rdata'].hset(ws.id, "currency", currency)
        await r.app['rdata'].hset(ws.id, "last-connect", float(time.time()))
        response['uuid'] = ws.id
        price_cur = await r.app['rdata'].hget("prices", "coingecko:nano-" + r.app['cur_prefs'][ws.id].lower())
        price_btc = await r.app['rdata'].hget("prices", "coingecko:nano-btc")
        response['currency'] = r.app['cur_prefs'][ws.id].lower()
        response['price'] = float(price_cur)
        response['btc'] = float(price_btc)
        response['pending_count'] = await get_pending_count(r, account)
        response = json.dumps(response)

        log.server_logger.info(
            'subscribe response sent ; %s bytes; %s; %s', str(len(response)), get_request_ip(r), ws.id)

        ws.send_str(response)

# Primary handler for all websocket connections
async def handle_user_message(r, msg, ws=None):
    """Process data sent by client"""
    address = get_request_ip(r)
    message = msg.data
    uid = ws.id if ws is not None else 0
    now = int(round(time.time() * 1000))
    if address in r.app['last_msg']:
        if (now - r.app['last_msg'][address]) < 25:
            log.server_logger.error('client messaging too quickly: %s ms; %s; %s; User-Agent: %s', str(
                now - r.app['last_msg'][address]), address, uid, str(
                r.headers.get('User-Agent')))
            return
    r.app['last_msg'][address] = now
    log.server_logger.info('request; %s, %s, %s', message, address, uid)
    if message not in r.app['active_messages']:
        r.app['active_messages'].add(message)
    else:
        log.server_logger.error('request already active; %s; %s; %s', message, address, uid)
        return
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
                            account_list = json.loads(account.decode('utf-8'))
                            if 'account' in request_json and request_json['account'].lower() not in account_list:
                                account_list.append(request_json['account'].lower())
                                await r.app['rdata'].hset(request_json['uuid'], "account", json.dumps(account_list))
                        except Exception as e:
                            if 'account' in request_json and request_json['account'].lower() != account.decode('utf-8').lower():
                                resubscribe = False
                            else:
                                # Perform upgrade to list style
                                account_list = []
                                account_list.append(account.decode('utf-8').lower())
                                await r.app['rdata'].hset(request_json['uuid'], "account", json.dumps(account_list))
                # already subscribed, reconnect (websocket connections)
                if 'uuid' in request_json and resubscribe:
                    del r.app['clients'][uid]
                    uid = request_json['uuid']
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
                        account_list = json.loads(await r.app['rdata'].hget(uid, "account").decode('utf-8'))
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
                        await rpc_reconnect(ws, r, account)
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
                        return json.dumps(reply)
                # new user, setup uuid(or use existing if available) and account info
                else:
                    log.server_logger.info('subscribe request; %s; %s', get_request_ip(r), uid)
                    try:
                        if 'currency' in request_json and request_json['currency'] in currency_list:
                            currency = request_json['currency']
                        else:
                            currency = 'usd'
                            await rpc_subscribe(ws, r, request_json['account'].replace("nano_", "xrb_"), currency)
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
                        return json.dumps(reply)
            elif request_json['action'] == "fcm_update":
                # Updating FCM token
                if 'fcm_token_v2' in request_json and 'account' in request_json and 'enabled' in request_json:
                    if request_json['enabled']:
                        await update_fcm_token_for_account(request_json['account'], request_json['fcm_token_v2'], r, v2=True)
                    else:
                        await delete_fcm_token_for_account(request_json['account'], request_json['fcm_token_v2'], r)
    except Exception:
        pass

async def websocket_handler(r):
    """Handler for websocket connections and messages"""

    ws = web.WebSocketResponse()
    await ws.prepare(r)

    # Connection Opened
    ws.id = str(uuid.uuid4())
    r.app['clients'][ws.id] = ws
    log.server_logger.info('new connection;%s;%s;User-Agent:%s', get_request_ip(r), ws.id, str(
        r.headers.get('User-Agent')))

    try:
        async for msg in ws:
            if msg.type == WSMsgType.TEXT:
                if msg.data == 'close':
                    await ws.close()
                else:
                    reply = await handle_user_message(r, msg, ws=ws)
                    if reply is not None:
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

async def send_prices():
    """Send price updates to connected clients once per minute"""
    while True:
        pass
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

    # Setup logger
    if debug_mode:
        logging.basicConfig(level='DEBUG')
    else:
        root = log.server_logger.getLogger()
        logging.basicConfig(level='INFO')
        handler = WatchedFileHandler(os.environ.get("NANO_LOG_FILE", "natriumcast.log"))
        formatter = logging.Formatter("%(asctime)s;%(levelname)s;%(message)s", "%Y-%m-%d %H:%M:%S %z")
        handler.setFormatter(formatter)
        root.addHandler(handler)
        root.addHandler(TimedRotatingFileHandler(os.environ.get("NANO_LOG_FILE", "natriumcast.log"), when="d", interval=1, backupCount=100))        

    app = web.Application()
    app.add_routes([web.get('/', websocket_handler)]) # All WS requests
    #app.add_routes([web.post('/callback', callback)])
    app.on_startup.append(open_redis)
    app.on_shutdown.append(close_redis)

    return app

app = loop.run_until_complete(init_app())

def main():
    """Main application loop"""

    # Periodic price job
    price_task = loop.create_task(send_prices())

    # Start web/ws server
    async def start():
        runner = web.AppRunner(app)
        await runner.setup()
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


