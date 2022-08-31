# Natrium (NANO) and Kalium (BANANO) Wallet Server

## What is Natrium, Kalium, NANO, BANANO?

Natrium and Kalium are mobile wallets written with Flutter. NANO and BANANO are cryptocurrencies.

| Link                                         | Description       |
| :------------------------------------------- | :---------------- |
| [natrium.io](https://natrium.io)             | Natrium Homepage  |
| [kalium.banano.cc](https://kalium.banano.cc) | Kalium Homepage   |
| [appditto.com](https://appditto.com)         | Appditto Homepage |

## Requirements

**GOLang**

Install the latest version of [GO](https://go.dev)

**NANO/BANANO Node with RPC enabled.**

Configured by the environment variable `RPC_URL`

e.g.

```
export RPC_URL=http://localhost:7076
```

**Redis server**

Configured with env variables:

```
REDIS_HOST  # default localhost
REDIST_PORT # default 6379
REDIS_DB    # default 0
```

**PostgreSQL**

Configured with:

```
DB_HOST # The host of the database
DB_PORT # The port to connect to on the database
DB_NAME # The name of the database
DB_USER # The user
DB_PASS # The password
```

**Other Configuration**

```
FCM_API_KEY # For push notifications
BPOW_KEY    # To use BoomPoW for work generation
```

## Running

Compile with `go build -o natrium-server`

Then run `./natrium-server` or `./natrium-server -banano` for banano mode.

## Work Generation

Configuring a service for work is required. You have two options.

- `WORK_URL` can be set in the environment to a work server (either the same as `RPC_URL`) or something like [nano-work-server](https://github.com/nanocurrency/nano-work-server)
- `BPOW_KEY` can be set in the environment to use [BoomPoW](https://boompow.banano.cc), BANANO's distributed proof of work system.

If both are set, `BPOW` will be preferred, followed by `WORK_URL` in the event of failure.

You can also override `BPOW_URL`, you would never want to do this, unless you are using a forked or self-hosted version of the service.

## Callback

Callback is required for push notifications and pushing new blocks to clients. This can be configured in the node's config.json as follows:

```
"callback_address": "::ffff:127.0.0.1",
"callback_port": "3000",
"callback_target": "\/callback",
```
