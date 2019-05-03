import uuid
import asyncio
import os
import json
import logging
import time
from logging.handlers import WatchedFileHandler, TimedRotatingFileHandler
import aioredis
from aiohttp import ClientSession, log, web, WSMsgType

# get environment
rpc_url = os.getenv('NANO_RPC_URL', 'http://127.0.0.1:7076') 
app_port = os.getenv('NANO_SERVER_PORT', 5076)
fcm_api_key = os.getenv('FCM_API_KEY')
fcm_sender_id = os.getenv('FCM_SENDER_ID')
debug_mode = os.getenv('DEBUG', 1)

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

# Primary handler for all websocket connections
async def handle_user_message(ws, r, msg):
    """Process data sent by client"""

    address = get_request_ip(r)
    message = msg.data
    now = int(round(time.time() * 1000))
    if address in r.app['last_msg']:
        if (now - r.app['last_msg'][address]) < 25:
            logging.error('client messaging too quickly: ' + str(
                now - r.app['last_msg'][address]) + 'ms;' + address + ';' + ws.id + ';User-Agent:' + str(
                r.headers.get('User-Agent')))
    r.app['last_msg'][address] = now
    logging.info('request;' + message + ';' + address + ';' + ws.id)
    if message not in r.app['active_messages']:
        r.app['active_messages'].add(message)
    else:
        logging.error('request already active;' + message + ';' + address + ';' + ws.id)
        return
    try:
        request_json = json.loads(message)
        if request_json['action'] in allowed_rpc_actions:
            if 'request_id' in request_json:
                requestid = request_json['request_id']
            else:
                requestid = None

            # adjust counts so nobody can block the node with a huge request - disregard, we have three nodes to
            # load balance

            # if 'count' in natriumcast_request:
            # if (natriumcast_request['count'] < 0) or (natriumcast_request['count'] > 1000):
            #     natriumcast_request['count'] = 1000
            #     logging.info('requested count is <0 or >1000, correcting to 1000;'+self.request.remote_ip+';'+self.id)

            # rpc: account_subscribe
            if request_json['action'] == "account_subscribe":
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
                # already subscribed, reconnect
                if 'uuid' in request_json and resubscribe:
                    del r.app['clients'][ws.id]
                    ws.id = request_json['uuid']
                    r.app['clients'][ws.id] = ws
                    log.server_logger.info('reconnection request;' + address + ';' + ws.id)
                    try:
                        if 'currency' in request_json and request_json['currency'] in currency_list:
                            currency = request_json['currency']
                            r.app['cur_prefs'][ws.id] = currency
                            await r.app['rdata'].hset(ws.id, "currency", currency)
                        else:
                            setting = await r.app['rdata'].hget(ws.id, "currency")
                            if setting is not None:
                                r.app['cur_prefs'][ws.id] = setting
                            else:
                                r.app['cur_prefs'][ws.id] = 'usd'
                                await r.app['rdata'].hset(ws.id, "currency", 'usd')

                        # Get relevant account
                        account_list = json.loads(await r.app['rdata'].hget(ws.id, "account").decode('utf-8'))
                        if 'account' in request_json:
                            account = request_json['account']
                        else:
                            # Legacy connections
                            account = account_list[0]
                        if 'nano_' in account:
                            account_list.remove(account)
                            account_list.append(account.replace("nano_", "xrb_"))
                            account = account.replace('nano_', 'xrb_')
                            await r.app['rdata'].hset(ws.id, "account", json.dumps(account_list))
                        # rpc_reconnect(ws, account) // TODO implement
                        await r.app['rdata'].rpush("conntrack",
                                    str(float(time.time())) + ":" + ws.id + ":connect:" + address)
                        # Store FCM token for this account, for push notifications
                        if 'fcm_token' in request_json:
                            pass
                            #update_fcm_token_for_account(account, natriumcast_request['fcm_token'])
                        elif 'fcm_token_v2' in request_json and 'notification_enabled' in request_json:
                            if request_json['notification_enabled']:
                                pass
                                #update_fcm_token_for_account(account, natriumcast_request['fcm_token_v2'], v2=True)
                            else:
                                pass
                                #delete_fcm_token_for_account(account, natriumcast_request['fcm_token_v2']) 
                    except Exception as e:
                        log.server_logger.error('reconnect error;' + str(e) + ';' + address + ';' + ws.id)
                        reply = {'error': 'reconnect error', 'detail': str(e)}
                        if requestid is not None: reply['request_id'] = requestid
                        await ws.send_str(json.dumps(reply))
                # new user, setup uuid(or use existing if available) and account info
                else:
                    # TODO
                    pass
    except Exception:
        pass

async def websocket_handler(r):
    """Handler for websocket connections and messages"""

    ws = web.WebSocketResponse()
    await ws.prepare(r)

    # Connection Opened
    ws.id = str(uuid.uuid4())
    r.app['clients'][ws.id] = ws
    log.server_logger.info('new connection ')
    log.server_logger.info('new connection;' + get_request_ip(r) + ';' + ws.id + ';User-Agent:' + str(
        r.headers.get('User-Agent')))

    try:
        async for msg in ws:
            if msg.type == WSMsgType.TEXT:
                if msg.data == 'close':
                    await ws.close()
                else:
                    await handle_user_message(ws, r, msg)
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
        app['rfcm'] = await aioredis.create_redis(('localhost', 6379),
                                                db=1, encoding='utf-8')
        app['rdata'] = await aioredis.create_redis(('localhost', 6379),
                                                db=2, encoding='utf-8')
        # Global vars
        app['clients'] = {} # Keep track of connected clients
        app['last_msg'] = {} # Last time a client has sent a message
        app['active_messages'] = set() # Avoid duplicate messages from being processes simultaneously
        app['cur_prefs'] = {} # Client currency preferences

    # Setup logger
    if debug_mode > 0:
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
        global runner, site
        runner = web.AppRunner(app)
        await runner.setup()
        site = web.TCPSite(runner, '127.0.0.1', app_port)
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


