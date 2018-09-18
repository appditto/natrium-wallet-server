# -*- coding: utf-8 -*-

import json
import logging
import os
import ssl
import sys
import time
import uuid
from logging.handlers import WatchedFileHandler

import aiofcm
import redis
import tornado.gen
import tornado.httpclient
import tornado.httpserver
import tornado.ioloop
import tornado.web
import tornado.websocket
from bitstring import BitArray

import natriumcast


# future use for caching blocks
# rblock  = redis.StrictRedis(host='localhost', port=6379, db=0)

# future use for pending blocks for accounts, cached work
# racct   = redis.StrictRedis(host='localhost', port=6379, db=1)

rdata = redis.StrictRedis(host='localhost', port=6379, db=2)  # used for price data and subscriber uuid info


# get environment
rpc_url = os.getenv('NANO_RPC_URL', 'http://127.0.0.1:7076')  # use env, else default to localhost rpc port
callback_port = os.getenv('NANO_CALLBACK_PORT', 17076)
socket_port = os.getenv('NANO_SOCKET_PORT', 443)
cert_dir = os.getenv('NANO_CERT_DIR')  # use /home/username instead of /home/username/
cert_key_file = os.getenv('NANO_KEY_FILE')  # TLS certificate private key
cert_crt_file = os.getenv('NANO_CRT_FILE')  # full TLS certificate bundle
fcm_api_key = os.getenv('FCM_API_KEY')
fcm_sender_id = os.getenv('FCM_SENDER_ID')

# whitelisted commands, disallow anything used for local node-based wallet as we may be using multiple back ends
allowed_rpc_actions = ["account_balance", "account_block_count", "account_check", "account_info", "account_history",
                       "account_representative", "account_subscribe", "account_weight", "accounts_balances",
                       "accounts_frontiers", "accounts_pending", "available_supply", "block", "block_hash",
                       "block_create", "blocks", "blocks_info", "block_account", "block_count", "block_count_type",
                       "chain", "delegators", "delegators_count", "frontiers", "frontier_count", "history",
                       "key_expand", "process", "representatives", "republish", "peers", "version", "pending",
                       "pending_exists", "price_data", "work_generate"]

# all currencies polled on CMC
currency_list = ["BTC", "AUD", "BRL", "CAD", "CHF", "CLP", "CNY", "CZK", "DKK", "EUR", "GBP", "HKD", "HUF", "IDR",
                 "ILS", "INR", "JPY", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "PKR", "PLN", "RUB", "SEK", "SGD",
                 "THB", "TRY", "TWD", "USD", "ZAR"]

# ephemeral data
clients = {}  # store websocket sessions
subscriptions = {}  # store subscription ids
sub_pref_cur = {}  # store currency subscription preferences [change to use redis next]
conn_count = {}  # track number of open connections per IP
mesg_last = {}  # track time of last message from IP
active_messages = set()  # track messages in-flight - combats duplicate requests while one is active

# track work requests active, eliminate client requesting multiples on the
# same hash (drops work server efficiency as it hasnt had time to cache yet, this way it doesnt queue)
active_work = set()


def address_decode(address):
    # Given a string containing an XRB/NANO address, confirm validity and provide resulting hex address
    if address[:4] == 'xrb_' or address[:5] == 'nano_':
        account_map = "13456789abcdefghijkmnopqrstuwxyz"  # each index = binary value, account_lookup[0] == '1'
        account_lookup = {}
        for i in range(0, 32):  # populate lookup index with prebuilt bitarrays ready to append
            account_lookup[account_map[i]] = BitArray(uint=i, length=5)
        data = address.split('_')[1]
        acrop_key = data[:-8]  # we want everything after 'xrb_' or 'nano_' but before the 8-char checksum
        acrop_check = data[-8:]  # extract checksum

        # convert base-32 (5-bit) values to byte string by appending each 5-bit value to the bitstring,
        # essentially bitshifting << 5 and then adding the 5-bit value.
        number_l = BitArray()
        for x in range(0, len(acrop_key)):
            number_l.append(account_lookup[acrop_key[x]])

        number_l = number_l[4:]  # reduce from 260 to 256 bit (upper 4 bits are never used as account is a uint256)
        check_l = BitArray()

        for x in range(0, len(acrop_check)):
            check_l.append(account_lookup[acrop_check[x]])
        check_l.byteswap()  # reverse byte order to match hashing format
        result = number_l.hex.upper()
        return result

    return False

def update_fcm_token_for_account(account, token):
    """Store device FCM registration tokens in redis"""
    rdata.set(token, account, ex=2592000) # Expire after 30-day inactivity
    # Keep a list of tokens associated with this account
    cur_list = rdata.get(account)
    if cur_list is not None:
        cur_list = json.loads(cur_list.decode('utf-8'))
    else:
        cur_list = {}
    if 'data' not in cur_list:
        cur_list['data'] = []
    if token not in cur_list['data']:
        cur_list['data'].append(token)
        rdata.set(account, json.dumps(cur_list))


def get_fcm_tokens(account):
    """Return list of FCM tokens that belong to this account"""
    ret = []
    tokens = rdata.get(account)
    if tokens is None:
        return None
    tokens = json.loads(tokens.decode('utf-8'))
    if 'data' not in tokens:
        return None
    for t in tokens['data']:
        fcm_account = rdata.get(t)
        if fcm_account is None or account != fcm_account.decode('utf-8'):
            continue
        ret.append(t)
    return ret

# strip whitespace, conform to string output
def strclean(instr):
    if type(instr) is str:
        return ' '.join(instr.split())
    elif type(instr) is bytes:
        return ' '.join(instr.decode('utf-8').split())


@tornado.gen.coroutine
def send_prices():
    # global active_work
    # active_work = set()
    # empty out this set periodically, to ensure clients dont somehow get stuck when an error causes their
    # work not to return
    if len(clients):
        print('[' + str(int(time.time())) + '] Pushing price data to ' + str(len(clients)) + ' subscribers...')
        logging.info('pushing price data to ' + str(len(clients)) + ' connections')
        btc = float(rdata.hget("prices", "coinmarketcap:nano-btc").decode('utf-8'))
        for client in clients:
            try:
                try:
                    currency = sub_pref_cur[client]
                except:
                    currency = 'usd'
                price = float(rdata.hget("prices", "coinmarketcap:nano-" + currency.lower()).decode('utf-8'))

                clients[client].write_message(
                    '{"currency":"' + currency.lower() + '","price":' + str(price) + ',"btc":' + str(btc) + '}')
            except:
                print(' > Error pushing prices for client ' + client)
                logging.error('error pushing prices for client;' + client)


@tornado.gen.coroutine
def rpc_request(http_client, body):
    response = yield http_client.fetch(rpc_url, method='POST', body=body)
    raise tornado.gen.Return(response)


@tornado.gen.coroutine
def rpc_defer(handler, message):
    rpc = tornado.httpclient.AsyncHTTPClient()
    response = yield rpc_request(rpc, message)
    logging.info('rpc request return code;' + str(response.code))
    if response.error:
        logging.error('rpc defer request failure;' + str(
            response.error) + ';' + rpc_url + ';' + message + ';' + handler.request.remote_ip + ';' + handler.id)
        reply = "rpc defer error"
    else:
        logging.info('rpc defer response sent;' + str(
            strclean(response.body)) + ';' + rpc_url + ';' + handler.request.remote_ip + ';' + handler.id)
        reply = response.body

    handler.write_message(reply)


# Since someone might get cute and attempt to spam users with low-value transactions in an effort to deny them the
# ability to receive, we will take the performance hit for them and pull all pending block data. Then we will sort by
# most valuable to least valuable. Finally, to save the client processing burden and give the server room to breathe,
# we return only the top 10.
@tornado.gen.coroutine
def pending_defer(handler, request):
    rpc = tornado.httpclient.AsyncHTTPClient()
    requested = json.loads(request)
    response = yield rpc_request(rpc, request)

    if response.error:
        logging.error('pending defer request failure;' + str(
            response.error) + ';' + rpc_url + ';' + request + ';' + handler.request.remote_ip + ';' + handler.id)
        reply = "pending defer error"
    else:
        data = json.loads(response.body.decode('ascii'))
        # sort dict keys by amount value within, descending
        newlist = sorted(data['blocks'], key=lambda x: (int(data['blocks'][x]['amount'])), reverse=True)
        # only provide the first 10
        newlist = newlist[:10]
        # build a new json structure
        if len(newlist) > 0:
            newdict = {"blocks": {}}
            for x in newlist:
                newdict['blocks'][x] = data['blocks'][x]
        else:
            # returning {} as the value for blocks causes issues with clients, RPC provides "", lets do the same.
            newdict = {
                "blocks": ""}

        reply = json.dumps(newdict)
        logging.info('pending defer response sent;' + str(
            strclean(reply)) + ';' + rpc_url + ';' + handler.request.remote_ip + ';' + handler.id)

    # return to client
    handler.write_message(reply)


# Server-side check for any incidental mixups due to race conditions or misunderstanding protocol.
# Check blocks submitted for processing to ensure the user or client has not accidentally created a send to an unknown
# address due to balance miscalculation leading to the state block being interpreted as a send rather than a receive.
@tornado.gen.coroutine
def process_defer(handler, block):
    rpc = tornado.httpclient.AsyncHTTPClient()

    # Let's cache the link because, due to callback delay it's possible a client can receive
    # a push notification for a block it already knows about
    if 'link' in block:
        rdata.set(f"link_{block['link']}", "1", ex=3600)

    # check for receive race condition
    # if block['type'] == 'state' and block['previous'] and block['balance'] and block['link']:
    if block['type'] == 'state' and {'previous', 'balance', 'link'} <= set(block):
        try:
            prev_response = yield rpc_request(rpc, json.dumps({
                'action': 'blocks_info',
                'hashes': [block['previous']],
                'balance': 'true'
            }))
            prev_response = json.loads(prev_response.body.decode('ascii'))

            try:
                prev_block = json.loads(prev_response['blocks'][block['previous']]['contents'])

                if prev_block['type'] != 'state' and ('balance' in prev_block):
                    prev_balance = int(prev_block['balance'], 16)
                elif prev_block['type'] != 'state' and ('balance' not in prev_block):
                    prev_balance = int(prev_response['blocks'][block['previous']]['balance'])
                else:
                    prev_balance = int(prev_block['balance'])

                if int(block['balance']) < prev_balance:
                    link_hash = block['link']
                    if link_hash.startswith('xrb_') or link_hash.startswith('nano_'):
                        link_hash = address_decode(link_hash)
                    # this is a send
                    link_response = yield rpc_request(rpc, json.dumps({
                        'action': 'block',
                        'hash': link_hash
                    }))
                    link_response = json.loads(link_response.body.decode('ascii'))
                    # print('link_response',link_response)
                    if 'error' not in link_response and 'contents' in link_response:
                        logging.error(
                            'rpc process receive race condition detected;' + handler.request.remote_ip +
                            ';' + handler.id + ';User-Agent:' + str(handler.request.headers.get('User-Agent')))
                        handler.write_message('{"error":"receive race condition detected"}')
                        return
            except:
                # no contents, means an error was returned for previous block. no action needed
                if 'error' not in prev_response:
                    exc_type, exc_obj, exc_tb = sys.exc_info()
                    print('exception', exc_type, exc_obj, exc_tb.tb_lineno)
                    print('prev_response', prev_response)
                # print('prev_block',prev_block)
                pass
        except:
            logging.error('rpc process receive race condition exception;' + str(
                sys.exc_info()) + ';' + handler.request.remote_ip + ';' + handler.id + ';User-Agent:' + str(
                handler.request.headers.get('User-Agent')))
            pass

    response = yield rpc_defer(handler, json.dumps({
        'action': 'process',
        'block': json.dumps(block)
    }))


@tornado.gen.coroutine
def work_request(http_client, body):
    response = yield http_client.fetch(rpc_url, method='POST', body=body)
    raise tornado.gen.Return(response)


@tornado.gen.coroutine
def work_defer(handler, message):
    request = json.loads(message)
    if request['hash'] in active_work:
        logging.error('work already requested;' + handler.request.remote_ip + ';' + handler.id)
        return
    else:
        active_work.add(request['hash'])
        request['use_peers'] = True
        message = json.dumps(request)
    try:
        rpc = tornado.httpclient.AsyncHTTPClient()
        response = yield work_request(rpc, message)
        logging.info('work request return code;' + str(response.code))
        if response.error:
            logging.error('work defer error;' + handler.request.remote_ip + ';' + handler.id)
            handler.write_message("work defer error")
        else:
            logging.info('work defer response sent:;' + str(
                strclean(response.body)) + ';' + handler.request.remote_ip + ';' + handler.id)
            handler.write_message(response.body)
        active_work.remove(request['hash'])
    except:
        logging.error(
            'work defer exception;' + str(sys.exc_info()) + ';' + handler.request.remote_ip + ';' + handler.id)
        active_work.remove(request['hash'])


@tornado.gen.coroutine
def rpc_subscribe(handler, account, currency):
    logging.info('subscribing;' + handler.request.remote_ip + ';' + handler.id)
    rpc = tornado.httpclient.AsyncHTTPClient()
    message = '{\"action\":\"account_info",\"account\":\"' + account + '\",\"pending\":true,\"representative\":true}'
    logging.info('sending request;' + message + ';' + handler.request.remote_ip + ';' + handler.id)
    response = yield rpc_request(rpc, message)

    if response.error:
        logging.error('subscribe error;' + handler.request.remote_ip + ';' + handler.id)
        handler.write_message('{"error":"subscribe error"}')
    else:
        subscriptions[account] = handler.id
        rdata.hset(handler.id, "account", account)
        sub_pref_cur[handler.id] = currency
        rdata.hset(handler.id, "currency", currency)
        rdata.hset(handler.id, "last-connect", float(time.time()))
        info = json.loads(response.body)
        info['uuid'] = handler.id
        price_cur = rdata.hget("prices", "coinmarketcap:nano-" + sub_pref_cur[handler.id].lower()).decode('utf-8')
        price_btc = rdata.hget("prices", "coinmarketcap:nano-btc").decode('utf-8')
        info['currency'] = sub_pref_cur[handler.id].lower()
        info['price'] = price_cur
        info['btc'] = price_btc
        info = json.dumps(info)
        logging.info('subscribe response sent;' + str(
            strclean(response.body)) + ';' + handler.request.remote_ip + ';' + handler.id)
        handler.write_message(info)


@tornado.gen.coroutine
def rpc_reconnect(handler):
    logging.info('reconnecting;' + handler.request.remote_ip + ';' + handler.id)
    rpc = tornado.httpclient.AsyncHTTPClient()
    try:
        account = rdata.hget(handler.id, "account").decode('utf-8')
    except:
        logging.error(
            'reconnect error, account not seen on this server before;' + handler.request.remote_ip + ';' + handler.id)
        handler.write_message('{"error":"reconnect error","detail":"account not seen on this server before"}')
        return

    message = '{\"action\":\"account_info",\"account\":\"' + account + '\",\"pending\":true,\"representative\":true}'
    logging.info('sending request;' + message + ';' + handler.request.remote_ip + ';' + handler.id)
    response = yield rpc_request(rpc, message)

    if response.error:
        logging.error('reconnect error;' + handler.request.remote_ip + ';' + handler.id)
        handler.write_message('{"error":"reconnect error"}')
    else:
        subscriptions[
            account] = handler.id  # may be an issue here if we are to allow multiple clients with the same address...
        sub_pref_cur[handler.id] = rdata.hget(handler.id, "currency").decode('utf-8')
        rdata.hset(handler.id, "last-connect", float(time.time()))
        info = json.loads(response.body.decode('ascii'))
        price_cur = rdata.hget("prices", "coinmarketcap:nano-" + sub_pref_cur[handler.id].lower()).decode('utf-8')
        price_btc = rdata.hget("prices", "coinmarketcap:nano-btc").decode('utf-8')
        info['currency'] = sub_pref_cur[handler.id].lower()
        info['price'] = float(price_cur)
        info['btc'] = float(price_btc)
        info = json.dumps(info)

        logging.info(
            'reconnect response sent ' + str(len(info)) + 'bytes;' + handler.request.remote_ip + ';' + handler.id)

        handler.write_message(info)


@tornado.gen.coroutine
def rpc_accountcheck(handler, account):
    logging.info('checking account;' + handler.request.remote_ip + ';' + handler.id)
    rpc = tornado.httpclient.AsyncHTTPClient()
    message = '{\"action\":\"account_info",\"account\":\"' + account + '\"}'
    logging.info('sending request;' + message + ';' + handler.request.remote_ip + ';' + handler.id)
    response = yield rpc_request(rpc, message)
    if response.error:
        logging.error('account check error;' + handler.request.remote_ip + ';' + handler.id)
        handler.write_message('{"error":"account check error"}')
    else:
        info = json.loads(response.body.decode('ascii'))
        try:
            if info['error'] == "Account not found":
                info = '{"ready":false}'
        except:
            info = '{"ready":true}'

        logging.info('account check response sent;' + handler.request.remote_ip + ';' + handler.id)
        handler.write_message(info)


class WSHandler(tornado.websocket.WebSocketHandler):
    def open(self):
        self.id = str(uuid.uuid4())
        clients[self.id] = self
        logging.info('new connection;' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
            self.request.headers.get('User-Agent')))

    def on_message(self, message):
        address = str(self.request.remote_ip)
        now = int(round(time.time() * 1000))
        if address in mesg_last:
            if (now - mesg_last[address]) < 25:
                logging.error('client messaging too quickly: ' + str(
                    now - mesg_last[address]) + 'ms;' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
                    self.request.headers.get('User-Agent')))
        mesg_last[address] = now
        logging.info('request;' + message + ';' + self.request.remote_ip + ';' + self.id)
        if message not in active_messages:
            active_messages.add(message)
        else:
            logging.error('request already active;' + message + ';' + self.request.remote_ip + ';' + self.id)
            return
        try:
            natriumcast_request = json.loads(message)
            if natriumcast_request['action'] in allowed_rpc_actions:
                if 'request_id' in natriumcast_request:
                    requestid = natriumcast_request['request_id']
                else:
                    requestid = None

                # adjust counts so nobody can block the node with a huge request - disregard, we have three nodes to
                # load balance

                # if 'count' in natriumcast_request:
                # if (natriumcast_request['count'] < 0) or (natriumcast_request['count'] > 1000):
                #     natriumcast_request['count'] = 1000
                #     logging.info('requested count is <0 or >1000, correcting to 1000;'+self.request.remote_ip+';'+self.id)

                # rpc: account_subscribe
                if natriumcast_request['action'] == "account_subscribe":
                    # already subscribed, reconnect
                    if 'uuid' in natriumcast_request:
                        del clients[self.id]
                        self.id = natriumcast_request['uuid']
                        clients[self.id] = self
                        logging.info('reconnection request;' + self.request.remote_ip + ';' + self.id)
                        try:
                            if 'currency' in natriumcast_request and natriumcast_request['currency'] in currency_list:
                                currency = natriumcast_request['currency']
                                sub_pref_cur[self.id] = currency
                                rdata.hset(self.id, "currency", currency)
                            else:
                                setting = rdata.hget(self.id, "currency")
                                if setting is not None:
                                    sub_pref_cur[self.id] = setting
                                else:
                                    sub_pref_cur[self.id] = 'usd'
                                    rdata.hset(self.id, "currency", 'usd')

                            rpc_reconnect(self)
                            rdata.rpush("conntrack",
                                        str(float(time.time())) + ":" + self.id + ":connect:" + self.request.remote_ip)
                            # Store FCM token if available, for push notifications
                            if 'fcm_token' in natriumcast_request:
                                update_fcm_token_for_account(rdata.hget(self.id, "account").decode('utf-8'), natriumcast_request['fcm_token'])
                        except Exception as e:
                            logging.error('reconnect error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
                            reply = {'error': 'reconnect error', 'detail': str(e)}
                            if requestid is not None: reply['request_id'] = requestid
                            self.write_message(json.dumps(reply))
                    # new user, setup uuid(or use existing if available) and account info
                    else:
                        logging.info('subscription request;' + self.request.remote_ip + ';' + self.id)
                        try:
                            if 'currency' in natriumcast_request and natriumcast_request['currency'] in currency_list:
                                currency = natriumcast_request['currency']
                            else:
                                currency = "usd"
                            rpc_subscribe(self, natriumcast_request['account'], currency)
                            rdata.rpush("conntrack",
                                        str(float(time.time())) + ":" + self.id + ":connect:" + self.request.remote_ip)
                            # Store FCM token if available, for push notifications
                            if 'fcm_token' in natriumcast_request:
                                update_fcm_token_for_account(natriumcast_request['account'], natriumcast_request['fcm_token'])
                        except Exception as e:
                            logging.error('subscribe error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
                            reply = {'error': 'subscribe error', 'detail': str(e)}
                            if requestid is not None: reply['request_id'] = requestid
                            self.write_message(json.dumps(reply))
                # rpc: price_data
                elif natriumcast_request['action'] == "price_data":
                    logging.info('price data request;' + self.request.remote_ip + ';' + self.id)
                    try:
                        if natriumcast_request['currency'].upper() in currency_list:
                            try:
                                price = rdata.hget("prices",
                                                   "coinmarketcap:nano-" + natriumcast_request['currency'].lower()).decode(
                                    'utf-8')
                                self.write_message(
                                    '{"currency":"' + natriumcast_request['currency'].lower() + '","price":' + str(
                                        price) + '}')
                            except:
                                logging.error(
                                    'price data error, unable to get price;' + self.request.remote_ip + ';' + self.id)
                                self.write_message('{"error":"price data error - unable to get price"}')
                        else:
                            logging.error(
                                'price data error, unknown currency;' + self.request.remote_ip + ';' + self.id)
                            self.write_message('{"error":"unknown currency"}')
                    except Exception as e:
                        logging.error('price data error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
                        self.write_message('{"error":"price data error","detail":"' + str(e) + '"}')

                # rpc: account_check
                elif natriumcast_request['action'] == "account_check":
                    logging.info('account check request;' + self.request.remote_ip + ';' + self.id)
                    try:
                        rpc_accountcheck(self, natriumcast_request['account'])
                    except Exception as e:
                        logging.error('account check error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
                        self.write_message('{"error":"account check error","detail":"' + str(e) + '"}')

                # rpc: work_generate
                elif natriumcast_request['action'] == "work_generate":
                    if self.request.headers.get('X-Client-Version') is None:
                        xcver = 0
                    else:
                        xcver = int(self.request.headers.get('X-Client-Version'))
                    # logging.debug(self.request.headers)
                    if (self.request.headers.get('User-Agent') != 'SwiftWebSocket') and (xcver < 28):
                        logging.error(
                            'work rpc denied;work disable;' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
                                self.request.headers.get('User-Agent')))
                        self.write_message(
                            '{"error":"work rpc denied","detail":"you are using an old client, please update"}')
                    else:
                        try:
                            work_defer(self, json.dumps(natriumcast_request))
                        except Exception as e:
                            logging.error('work rpc error;' + str(
                                e) + ';' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
                                self.request.headers.get('User-Agent')))
                            self.write_message('{"error":"work rpc error","detail":"' + str(e) + '"}')

                # rpc: process
                elif natriumcast_request['action'] == "process":
                    try:
                        process_defer(self, json.loads(natriumcast_request['block']))
                    except Exception as e:
                        logging.error('process rpc error;' + str(
                            e) + ';' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
                            self.request.headers.get('User-Agent')))
                        self.write_message('{"error":"process rpc error","detail":"' + str(e) + '"}')

                # rpc: pending
                elif natriumcast_request['action'] == "pending":
                    try:
                        pending_defer(self, json.dumps(natriumcast_request))
                    except Exception as e:
                        logging.error('pending rpc error;' + str(
                            e) + ';' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
                            self.request.headers.get('User-Agent')))
                        self.write_message('{"error":"pending rpc error","detail":"' + str(e) + '"}')
                elif natriumcast_request['action'] == 'account_history':
                    if rdata.hget(self.id, "account") is None:
                        rdata.hset(self.id, "account", natriumcast_request['account'])
                    try:
                        rpc_defer(self, json.dumps(natriumcast_request))
                    except Exception as e:
                        logging.error('rpc error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
                        self.write_message('{"error":"rpc error","detail":"' + str(e) + '"}')

                # rpc: fallthrough and error catch
                else:
                    try:
                        rpc_defer(self, json.dumps(natriumcast_request))
                    except Exception as e:
                        logging.error('rpc error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
                        self.write_message('{"error":"rpc error","detail":"' + str(e) + '"}')
            else:
                logging.error(
                    'rpc not allowed;' + natriumcast_request['action'] + ';' + self.request.remote_ip + ';' + self.id)
                self.write_message('{"error":"rpc command not allowed"}')
        except Exception as e:
            logging.error('uncaught error;' + str(e) + ';' + self.request.remote_ip + ';' + self.id)
            self.write_message('{"error":"general error","detail":"' + str(e) + '"}')
            active_messages.remove(message)
        # cleanup when done, allow repeats after done processing the first
        active_messages.remove(message)

    def on_close(self):
        logging.info('connection closed;' + self.request.remote_ip + ';' + self.id + ';User-Agent:' + str(
            self.request.headers.get('User-Agent')))
        rdata.rpush("conntrack", str(float(time.time())) + ":" + self.id + ":disconnect:" + self.request.remote_ip)
        rdata.hset(self.id, "last-disconnect", float(time.time()))
        if self.id in clients: del clients[self.id]
        for account in subscriptions:
            if subscriptions[account] == self.id:
                del subscriptions[account]
                break


class Callback(tornado.web.RequestHandler):
    async def post(self):
        data = self.request.body.decode('utf-8')
        data = json.loads(data)
        data['block'] = json.loads(data['block'])

        if data['block']['type'] == 'send':
            target = data['block']['destination']
            if subscriptions.get(target):
                print("             Pushing to client %s" % subscriptions[target])
                logging.info('push to client;' + json.dumps(data['block']) + ';' + subscriptions[target])
                clients[subscriptions[target]].write_message(json.dumps(data))
        elif data['block']['type'] == 'state':
            link = data['block']['link_as_account']
            if subscriptions.get(link):
                print("             Pushing to client %s" % subscriptions[link])
                logging.info('push to client;' + json.dumps(data) + ';' + subscriptions[link])
                clients[subscriptions[link]].write_message(json.dumps(data))
            # Push FCM notification if this is a send
            fcm_tokens = get_fcm_tokens(link)
            if fcm_tokens is None or len(fcm_tokens) == 0:
                return
            rpc = tornado.httpclient.AsyncHTTPClient()
            response = await rpc_request(rpc, json.dumps({"action":"block", "hash":data['block']['previous']}))
            if response is None or response.error:
                return
            # See if this block was already pocketed
            cached_hash = rdata.get(f"link_{data['hash']}")
            if cached_hash is not None:
                return
            prev_data = json.loads(response.body.decode('ascii'))
            prev_data = prev_data['contents'] = json.loads(prev_data['contents'])
            prev_balance = int(prev_data['contents']['balance'])
            cur_balance = int(data['block']['balance'])
            if prev_balance - cur_balance > 0:
                # This is a send, push notifications
                fcm = aiofcm.FCM(fcm_sender_id, fcm_api_key)
                # Send notification with generic title, send amount as body. App should have localizations and use this information at its discretion
                for t in fcm_tokens:
                    message = aiofcm.Message(
                                device_token=t,
                                notification= {
                                    "title": "nc_notification",
                                    "body": str(prev_balance-cur_balance)
                                }
                    )
                    await fcm.send_message(message)
        elif subscriptions.get(data['account']):
            print("             Pushing to client %s" % subscriptions[data['account']])
            logging.info('push to client;' + json.dumps(data) + ';' + subscriptions[data['account']])
            clients[subscriptions[data['account']]].write_message(json.dumps(data))


application = tornado.web.Application([
    (r"/", WSHandler),
])

nodecallback = tornado.web.Application([
    (r"/", Callback),
])

if __name__ == "__main__":
    handler = WatchedFileHandler(os.environ.get("NANO_LOG_FILE", "natriumcast.log"))
    formatter = logging.Formatter("%(asctime)s;%(levelname)s;%(message)s", "%Y-%m-%d %H:%M:%S %z")
    handler.setFormatter(formatter)
    root = logging.getLogger()
    root.setLevel(os.environ.get("NANO_LOG_LEVEL", "INFO"))
    root.addHandler(handler)
    print("[" + str(int(time.time())) + "] Starting NatriumCast Server...")
    logging.info('Starting NatriumCast Server')
    logging.getLogger('tornado.access').disabled = True

    cert = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
    print(os.path.join(cert_dir, cert_crt_file), os.path.join(cert_dir, cert_key_file))
    cert.load_cert_chain(os.path.join(cert_dir, cert_crt_file), os.path.join(cert_dir, cert_key_file))

    https_server = tornado.httpserver.HTTPServer(application, ssl_options=cert)
    https_server.listen(socket_port)

    nodecallback.listen(callback_port)  # set in config.json as follows:
    # 	"callback_address": "127.0.0.1",
    # 	"callback_port": "17076",
    # 	"callback_target": "/"

    # push latest price data to all subscribers every minute
    tornado.ioloop.PeriodicCallback(send_prices, 60000).start()

    tornado.ioloop.IOLoop.instance().start()
