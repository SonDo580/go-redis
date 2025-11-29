# Mini Redis

A simple Redis-like server implemented in Go.

## Guide

[Build Redis from scratch](https://www.build-redis-from-scratch.dev/)

## Features

- RESP (Redis serialization protocol) for communication.
- In-memory database with persistence using AOF (append-only file)
- Redis commands: `PING`, `SET`, `GET`, `HSET`, `HGET`

## Prerequisites

- Go 1.24+
- redis-cli _(for testing)_

## Usage

### Run server

```bash
# Clone the repository
git clone git@github.com:SonDo580/mini-redis.git

# Run the server
go run .
```

### Connect with `redis-cli`

```bash
redis-cli -p 6379
```

Example commands:

```
PING
SET animal tiger
GET animal
HSET user name Son
HGET user name
```

## TODO (Self-implemented)

- handle multiple client connections
- handle more Redis commands
- ...
