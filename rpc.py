import rapidjson as json
import sys
import time

from aiohttp import ClientSession, log, web

from util import Util

# whitelisted commands, disallow anything used for local node-based wallet as we may be using multiple back ends
allowed_rpc_actions = ["account_balance", "account_block_count", "account_check", "account_info", "account_history",
                       "account_representative", "account_subscribe", "account_weight", "accounts_balances",
                       "accounts_frontiers", "accounts_pending", "available_supply", "block", "block_hash",
                       "blocks", "block_info", "blocks_info", "block_account", "block_count", "block_count_type",
                       "chain", "delegators", "delegators_count", "frontiers", "frontier_count", "history",
                       "key_expand", "process", "representatives", "republish", "peers", "version", "pending",
                       "pending_exists", "price_data", "fcm_update"]

class RPC:
    def __init__(self, node_url : str, banano_mode : bool, work_url : str = None, price_prefix : str = None):
        self.node_url = node_url
        self.work_url = work_url
        self.banano_mode = banano_mode
        self.util = Util(banano_mode)
        self.price_prefix = price_prefix

    async def json_post(self, request_json : dict, timeout : int = 90, is_work : bool = False) -> dict:
        try:
            async with ClientSession() as session:
                async with session.post(self.work_url if is_work and self.work_url is not None else self.node_url, json=request_json, timeout=timeout) as resp:
                    if resp.status > 299:
                        log.server_logger.error('Received status code %d from request %s', resp.status, json.dumps(request_json))
                        raise Exception
                    return await resp.json(content_type=None)
        except Exception:
            log.server_logger.exception("exception in json_post")
            return None

    async def get_pending_count(self, r : web.Request, account : str, uid : str = '0') -> int:
        """This returns how many pending blocks an account has, up to 51, for anti-spam measures"""
        message = {
            "action":"pending",
            "account":account,
            "threshold":str(10**24) if not self.banano_mode else str(10**27),
            "count":51,
            "include_only_confirmed": True
        }
        log.server_logger.info('sending get_pending_count; %s; %s', self.util.get_request_ip(r), uid)
        response = await self.json_post(message)
        if response is None or 'blocks' not in response:
            return 0
        log.server_logger.debug('received response for pending %s', json.dumps(response))        
        return len(response['blocks'])

    async def rpc_reconnect(self, ws : web.WebSocketResponse, r : web.Response, account : str):
        """When a websocket connection sends a subscribe request, do this reconnection step"""
        log.server_logger.info('reconnecting;' + self.util.get_request_ip(r) + ';' + ws.id)

        rpc = {
            "action":"account_info",
            "account":account,
            "pending":True,
            "representative": True
        }
        log.server_logger.info('sending account_info %s', account)
        response = await self.json_post(rpc)

        if response is None:
            log.server_logger.error('reconnect error; %s ; %s', self.util.get_request_ip(r), ws.id)
            ws.send_str('{"error":"reconnect error"}')
        else:
            log.server_logger.debug('received response for account_info %s', json.dumps(response))     
            if account in r.app['subscriptions']:
                r.app['subscriptions'][account].add(ws.id)
            else:
                r.app['subscriptions'][account] = set()
                r.app['subscriptions'][account].add(ws.id)
            r.app['cur_prefs'][ws.id] = await r.app['rdata'].hget(ws.id, "currency")
            await r.app['rdata'].hset(ws.id, "last-connect", float(time.time()))
            price_cur = await r.app['rdata'].hget("prices", f"{self.price_prefix}-" + r.app['cur_prefs'][ws.id].lower())
            price_btc = await r.app['rdata'].hget("prices", f"{self.price_prefix}-btc")
            response['currency'] = r.app['cur_prefs'][ws.id].lower()
            response['price'] = float(price_cur)
            response['btc'] = float(price_btc)
            if self.banano_mode:
                response['nano'] = float(await r.app['rdata'].hget("prices", f"{self.price_prefix}-nano"))
            response['pending_count'] = await self.get_pending_count(r, account, uid = ws.id)
            response = json.dumps(response)

            log.server_logger.info(
                'reconnect response sent ; %s bytes; %s; %s', str(len(response)), self.util.get_request_ip(r), ws.id)

            await ws.send_str(response)

    async def rpc_subscribe(self, ws : web.WebSocketResponse, r : web.Response, account : str, currency : str):
        """Clients subscribing for the first time"""
        log.server_logger.info('subscribing;%s;%s', self.util.get_request_ip(r), ws.id)

        rpc = {
            'action':'account_info',
            'account':account,
            'pending':True,
            'representative':True
        }
        log.server_logger.info('sending account_info;%s;%s', self.util.get_request_ip(r), ws.id)
        response = await self.json_post(rpc)

        if response is None:
            log.server_logger.error('reconnect error; %s ; %s', self.util.get_request_ip(r), ws.id)
            await ws.send_str('{"error":"subscribe error"}')
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
            price_cur = await r.app['rdata'].hget("prices", f"{self.price_prefix}-" + r.app['cur_prefs'][ws.id].lower())
            price_btc = await r.app['rdata'].hget("prices", f"{self.price_prefix}-btc")
            response['currency'] = r.app['cur_prefs'][ws.id].lower()
            response['price'] = float(price_cur)
            response['btc'] = float(price_btc)
            if self.banano_mode:
                response['nano'] = float(await r.app['rdata'].hget("prices", f"{self.price_prefix}-nano"))
            response['pending_count'] = await self.get_pending_count(r, account)
            response = json.dumps(response)

            log.server_logger.info(
                'subscribe response sent ; %s bytes; %s; %s', str(len(response)), self.util.get_request_ip(r), ws.id)

            await ws.send_str(response)

    async def rpc_accountcheck(self, r : web.Response, uid : str, account : str) -> str:
        """See if account is open or not, return 'ready':True if it is open"""
        log.server_logger.info('rpc_accountcheck;%s;%s', self.util.get_request_ip(r), uid)
        rpc = {
            'action':'account_info',
            'account':account
        }
        log.server_logger.debug('sending request;%s;%s;%s', json.dumps(rpc), self.util.get_request_ip(r), uid)
        response = await self.json_post(rpc)
        if response is None:
            log.server_logger.error('account check error;%s;%s', self.util.get_request_ip(r), uid)
            return {
                'error': 'account_check error'
            }
        else:
            info = {'ready': True}
            try:
                if response['error'] == "Account not found":
                    info = {'ready': False}
            except Exception:
                pass
            return info

    async def work_request(self, request_json : dict) -> dict:
        """Send work_generate with use_peers injected"""
        if 'use_peers' not in request_json and self.work_url is None:
            request_json['use_peers'] = True
        return await self.json_post(request_json, is_work=True)

    async def work_defer(self, r : web.Request, uid : str, request_json : dict) -> str:
        """Request work_generate, but avoid duplicate requests"""
        if request_json['hash'] in r.app['active_work']:
            log.server_logger.error('work already requested;%s;%s', self.util.get_request_ip(r), uid)
            return None
        else:
            r.app['active_work'].add(request_json['hash'])
        try:
            log.server_logger.info('Requesting work for %s;%s', self.util.get_request_ip(r), uid)
            response = await self.work_request(request_json)
            if response is None:
                log.server_logger.error('work defer error; %s;%s', self.util.get_request_ip(r), uid)
                return json.dumps({
                    'error':'work defer error'
                })
            r.app['active_work'].remove(request_json['hash'])
            return response
        except Exception:
            log.server_logger.exception('in work defer')
            r.app['active_work'].remove(request_json['hash'])
            return None

    # Server-side check for any incidental mixups due to race conditions or misunderstanding protocol.
    # Check blocks submitted for processing to ensure the user or client has not accidentally created a send to an unknown
    # address due to balance miscalculation leading to the state block being interpreted as a send rather than a receive.
    async def process_defer(self, r : web.Request, uid : str, block : dict, do_work : bool, subtype: str = None) -> dict:
        # Let's cache the link because, due to callback delay it's possible a client can receive
        # a push notification for a block it already knows about
        is_change = True if subtype == 'change' else False
        if not is_change and 'link' in block:
            if block['link'].replace('0', '') == '':
                is_change = True
            else:
                await r.app['rdata'].set(f"link_{block['link']}", "1", expire=3600)

        # check for receive race condition
        # if block['type'] == 'state' and block['previous'] and block['balance'] and block['link']:
        if block['type'] == 'state' and {'previous', 'balance', 'link'} <= set(block):
            try:
                prev_response = await self.json_post({
                    'action': 'blocks_info',
                    'hashes': [block['previous']],
                    'balance': 'true'
                })

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
                        link_hash = self.util.address_decode(link_hash)
                        # this is a send
                        link_response = await self.json_post({
                            'action': 'block',
                            'hash': link_hash
                        })
                        # print('link_response',link_response)
                        if 'error' not in link_response and 'contents' in link_response:
                            log.server_logger.error(
                                'rpc process receive race condition detected;%s;%s;%s',
                                self.util.get_request_ip(r), uid, str(r.headers.get('User-Agent')))
                            return {
                                'error':'receive race condition detected'
                            }
                except Exception:
                    # no contents, means an error was returned for previous block. no action needed
                    log.server_logger.exception('in process_defer')
                    pass
            except Exception:
                log.server_logger.error('rpc process receive race condition exception;%s;%s;%s;User-Agent:%s',
                str(sys.exc_info()), self.util.get_request_ip(r), uid, str(r.headers.get('User-Agent')))
                pass

        # Do work if we're told to
        if 'work' not in block and do_work:
            try:
                if block['previous'] == '0' or block['previous'] == '0000000000000000000000000000000000000000000000000000000000000000':
                    workbase = self.util.pubkey(block['account'])
                else:
                    workbase = block['previous']
                if self.banano_mode:
                    difficulty = 'fffffe0000000000'
                    work_response = await self.work_request({
                        'action': 'work_generate',
                        'hash': workbase,
                        'difficulty': difficulty,
                        'reward': False
                    })
                else:
                    work_response = await self.work_request({
                        'action': 'work_generate',
                        'hash': workbase,
                        'subtype': subtype
                    })                    
                if work_response is None or 'work' not in work_response:
                    return {
                        'error':'failed work_generate in process request'
                    }
                block['work'] = work_response['work']
            except Exception:
                log.server_logger.exception('in work process_defer')
                return {
                    'error':"Failed work_generate in process request"
                }

        process_request = {
            'action':'process',
            'block': json.dumps(block)
        }
        if subtype is not None:
            process_request['subtype'] = subtype
        elif is_change:
            process_request['subtype'] = 'change'

        return await self.json_post(process_request)

    # Since someone might get cute and attempt to spam users with low-value transactions in an effort to deny them the
    # ability to receive, we will take the performance hit for them and pull all pending block data. Then we will sort by
    # most valuable to least valuable. Finally, to save the client processing burden and give the server room to breathe,
    # we return only the top 10.
    async def pending_defer(self, r : web.Request, uid : str, request : dict) -> dict:
        if 'include_only_confirmed' not in request:
            request['include_only_confirmed'] = True
        response = await self.json_post(request)

        if response is None:
            log.server_logger.error('pending defer request failure;%s;%s', self.util.get_request_ip(r), uid)
            return {
                'error':'rpc pending error'
            }
        else:
            return response
            # TODO - fix me
            # sort dict keys by amount value within, descending
            newlist = sorted(response['blocks'], key=lambda x: (int(response['blocks'][x]['amount'])), reverse=True)
            # only provide the first 10
            newlist = newlist[:10]
            # build a new json structure
            if len(newlist) > 0:
                newdict = {"blocks": {}}
                for x in newlist:
                    newdict['blocks'][x] = response['blocks'][x]
            else:
                # returning {} as the value for blocks causes issues with clients, RPC provides "", lets do the same.
                newdict = {
                    "blocks": ""}

            reply = newdict
            log.server_logger.info('pending defer response sent;%s;%s', self.util.get_request_ip(r), uid)

        # return to client
        return reply
