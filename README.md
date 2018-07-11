# nano-wallet-server

Requires **Python 3.6** 
[Download here](https://www.python.org/downloads/)

Minimum of one **NANO Node** with RPC enabled. See
[Build Instructions](https://github.com/nanocurrency/raiblocks/wiki/Build-rai_node-samples) and
[Installing as a Service](https://github.com/nanocurrency/raiblocks/wiki/Running-rai_node-as-a-service)
Once installed as a service, make sure the systemd service file has the following entry:
```
[Service]
LimitNOFILE=65536
```
This will help prevent your system from running out of file handles due to may connections.

**Redis server** running on the default port 6379
[Installation](https://redis.io/topics/quickstart)

## Installation
```git clone https://github.com/nano-wallet-company/nano-wallet-server/ nanocast```

Use virtualenv if desired, else ensure python3.6 and pip/pip3 are installed (debian) and install the following modules:
```sudo pip install pyblake2 redis tornado bitstring```

You must configure using environment variables. You may do this manually, as part of a launching script, in your bash settings, or within a systemd service.
```
export NANO_RPC_URL=http://<host>:<rpcport>
export NANO_WORK_URL=http://<host>:<workport>
export NANO_CALLBACK_PORT=17076
export NANO_SOCKET_PORT=443
export NANO_CERT_DIR=/home/<username>
export NANO_KEY_FILE=<yourdomain>.key
export NANO_CRT_FILE=<yourdomain>.crt
export NANO_LOG_FILE=/home/<username>/nanocast.log
export NANO_LOG_LEVEL=INFO
```
### Configure node for RPC
Ensure rpc is enabled as well as control (security over internal wallet is provided in whitelisted commands)

~/RaiBlocks/config.json:
```
    "rpc_enable": "true",
    "rpc": {
        "address": "::1",
        "port": "7076",
        "enable_control": "true",
```


### Configure node callback for new block publication
Set config.json for your node

~/RaiBlocks/config.json:
```
        "callback_address": "127.0.0.1",
        "callback_port": "17076",
        "callback_target": "\/",
```

## Setup cron job for price retrieval
```
pip install coinmarketcap requests certifi
```
within the ```bitcoin-price-api``` subfolder:
```python setup.py install```

Run ```crontab -e``` and add the following:
```
*/5 * * * * /usr/local/bin/python3.6 /home/<username>/nanocast/prices.py >/dev/null 2>&1
```

## systemd service file
Remember to change ```NANO_RPC_URL``` port if using haproxy.

/etc/systemd/system/nanocast.service
```
[Unit]
Description=nanocast
After=network.target
After=systemd-user-sessions.service
After=network-online.target

[Service]
Environment=NANO_RPC_URL=http://<host>:<rpcport>
Environment=NANO_WORK_URL=http://<host>:<workport>
Environment=NANO_CALLBACK_PORT=17076
Environment=NANO_SOCKET_PORT=443
Environment=NANO_CERT_DIR=/home/user
Environment=NANO_KEY_FILE=yourdomain.key
Environment=NANO_CRT_FILE=yourdomain.crt
Environment=NANO_LOG_FILE=/home/user/nanocast.log
Environment=NANO_LOG_LEVEL=INFO
LimitNOFILE=65536
ExecStart=/usr/local/bin/python3.6 /home/user/nanocast.py
Restart=always

[Install]
WantedBy=multi-user.target
```
Enable by running ```sudo systemctl enable nanocast.service``` run using ```sudo systemctl start nanocast.service```

## [optional] haproxy node load balancing
Multiple nodes may run on the same server as long as you change the RPC binding port for each. Same for the peering port.
```
global
        log /dev/log    local0
        log /dev/log    local1 notice
        chroot /var/lib/haproxy
        stats socket /run/haproxy/admin.sock mode 660 level admin
        stats timeout 30s
        user haproxy
        group haproxy
        daemon

        # Default SSL material locations
        ca-base /etc/ssl/certs
        crt-base /etc/ssl/private

        # Default ciphers to use on SSL-enabled listening sockets.
        # For more information, see ciphers(1SSL). This list is from:
        #  https://hynek.me/articles/hardening-your-web-servers-ssl-ciphers/
        # An alternative list with additional directives can be obtained from
        #  https://mozilla.github.io/server-side-tls/ssl-config-generator/?server=haproxy
        ssl-default-bind-ciphers ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:RSA+AESGCM:RSA+AES:!aNULL:!MD5:!DSS
        ssl-default-bind-options no-sslv3

defaults
        log     global
        mode    http
        option  httplog
        option  dontlognull
        timeout connect 5000
        timeout client  50000
        timeout server  50000
        errorfile 400 /etc/haproxy/errors/400.http
        errorfile 403 /etc/haproxy/errors/403.http
        errorfile 408 /etc/haproxy/errors/408.http
        errorfile 500 /etc/haproxy/errors/500.http
        errorfile 502 /etc/haproxy/errors/502.http
        errorfile 503 /etc/haproxy/errors/503.http
        errorfile 504 /etc/haproxy/errors/504.http

frontend rpc-frontend
        bind <this host IP or 127.0.0.1 if same host>:<port>         # different than the default RPC port on a single node
        mode http
        default_backend rpc-backend
        
backend rpc-backend
        balance first
        mode http
        option forwardfor
        timeout server 1000
        option redispatch
        server rpcbackend1 <node 1 server or localhost>:<rpc port> check
        server rpcbackend2 <node 2 server or localhost>:<rpc port> check
        server rpcbackend3 <node 3 server or localhost>:<rpc port> check
```
