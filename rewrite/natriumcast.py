import uuid
import asyncio
import os
import logging
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

# Primary handler for all websocket connections
async def handle_user_message(ws, msg):
    return None

async def websocket_handler(r):
    ws = web.WebSocketResponse()
    await ws.prepare(r)

    r.app['clients']

    try:
        async for msg in ws:
            if msg.type == WSMsgType.TEXT:
                if msg.data == 'close':
                    await ws.close()
                else:
                    await handle_user_message(ws, msg)
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
        await ws.close()

    return ws

async def send_prices():
    """Send price updates"""
    while True:
        pass
        await asyncio.sleep(60)

# Get application
async def init_app():
    async def close_redis(app):
        """Close redis connection"""
        log.server_logger.info('Closing redis connections')
        app['rfcm'].close()
        app['rdata'].close()

    async def open_redis(app):
        """Open redis connection"""
        log.server_logger.info("Opening redis connection")
        app['rfcm'] = await aioredis.create_redis(('localhost', 6379),
                                                db=1, encoding='utf-8')
        app['rdata'] = await aioredis.create_redis(('localhost', 6379),
                                                db=1, encoding='utf-8')
        # Global vars
        app['clients'] = []


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
    app['busy'] = False
    app.add_routes([web.get('/', websocket_handler)])
    #app.add_routes([web.post('/callback', callback)])
    app.on_startup.append(open_redis)
    app.on_shutdown.append(close_redis)

    return app

app = loop.run_until_complete(init_app())

def main():
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


