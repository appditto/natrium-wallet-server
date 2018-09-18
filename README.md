# natrium-wallet-server (NANO)

**Requires Python 3.6**

Install requirements on Ubuntu 18.04:
```
apt install python3 python3-dev libdpkg-perl virtualenv
```

Minimum of one **NANO Node** with RPC enabled.
Once installed as a service, make sure the systemd service file has the following entry:
```
[Service]
LimitNOFILE=65536
```
This will help prevent your system from running out of file handles due to may connections.

**Redis server** running on the default port 6379

On debian-based systems
```
sudo apt install redis-server
```

## Let's Encrypt
Setup SSL with let's encrypt and cert bot.

On ubuntu:

```
sudo add-apt-repository ppa:certbot/certbot
sudo apt update
sudo apt install certbot
sudo certbot certonly --standalone --preferred-challenges http -d <domain>
```

## Firebase (FCM)

For push notifications, get your legacy FCM api key from the firebase console.

## Installation
```git clone https://github.com/BananoCoin/natrium-wallet-server.git natriumcast```

Ensure python3.6 is installed and
```
cd natriumcast
virtualenv -p python3.6 venv
source venv/bin/activate
pip install -r requirements.txt
```

You must configure using environment variables. You may do this manually, as part of a launching script, in your bash settings, or within a systemd service.
```
export NANO_RPC_URL=http://<host>:<rpcport>
export NANO_CALLBACK_PORT=17076
export NANO_SOCKET_PORT=443
export NANO_CERT_DIR=/etc/letsencrypt/live/<domain>
export NANO_KEY_FILE=privkey.pem
export NANO_CRT_FILE=fullchain.pem
export NANO_LOG_FILE=/home/<username>/natriumcast.log
export NANO_LOG_LEVEL=INFO
export FCM_API_KEY=<firebase_api_key>
export FCM_SENDER_ID<firebase sender id>
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

Run ```crontab -e``` and add the following:
```
*/5 * * * * /home/<username>/natriumcast/venv/bin/python /home/<username>/natriumcast/prices.py >/dev/null 2>&1
```

## systemd service file
Remember to change ```NANO_RPC_URL``` port if using haproxy.

/etc/systemd/system/natriumcast.service
```
[Unit]
Description=natriumcast
After=network.target
After=systemd-user-sessions.service
After=network-online.target

[Service]
Environment=NANO_RPC_URL=http://<host>:<rpcport>
Environment=NANO_CALLBACK_PORT=17076
Environment=NANO_SOCKET_PORT=443
Environment=NANO_CERT_DIR=/etc/letsencrypt/live/<domain>
Environment=NANO_KEY_FILE=privkey.pem
Environment=NANO_CRT_FILE=fullchain.pem
Environment=NANO_LOG_FILE=/home/user/natriumcast.log
Environment=NANO_LOG_LEVEL=INFO
Environment=FCM_API_KEY=<firebase_api_key>
Environment=FCM_SENDER_ID=<firebase sender id>
LimitNOFILE=65536
ExecStart=/usr/local/bin/python3.6 /home/user/natriumcast/natriumcast.py
Restart=always

[Install]
WantedBy=multi-user.target
```
Enable by running ```sudo systemctl enable natriumcast.service``` run using ```sudo systemctl start natriumcast.service```

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
