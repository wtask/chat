# chatsrv

Text chat server over TCP.

## Prerequisites

To build server locally by your own you need to install [Golang](https://golang.org/) distribution for your platform.

Or you may use ready _docker image_ of the chat server. In that case you need to install [Docker](https://www.docker.com/). See server image link below.

## Compile

Run next command in your console under `cmd/chatsrv` folder to compile server binary:

```CLI
go install .
```

The server binary will be placed into `bin` folder under Go projects root, identified by `GOPATH` environment variable.

## Running

To start server, run compiled binary or simply enter next command in console under `cmd/chatsrv` folder:

```CLI
go run .
```

## Docker image

1. Check [docker hub](https://hub.docker.com/r/wtask/chat) to select available server images: https://hub.docker.com/r/wtask/chat

2. Run server in docker container:

```CLI
docker run -d --rm -p 20000:20000 --name tcpchat wtask/chat:x.y.z
```

where:
* `20000` is a chat server port
* `wtask/chat:x.y.z` is an available docker image (https://hub.docker.com/r/wtask/chat/tags).

## CLI options

Run server binary with option `-help` to get help for available options:

```CLI
chatsrv -help

Launch text chat server over TCP

        chatsrv [options]
Options:

  -client-timeout int
        Idle duration in seconds before client is disconnected. (default 60)
  -help
        Print usage help
  -history-greets int
        Num of messages from chat history which is pushed to newly connected client. (default 10)
  -ip string
        Listen address
  -port uint
        Listen port (default 20000)
```

## Chat client

You may use `netcat`, `telnet` or similar utilities as chat clients. For example, run this command to connect to the chat server with `netcat` locally:

```CLI
nc localhost 20000
```
