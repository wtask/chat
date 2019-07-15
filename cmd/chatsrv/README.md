# chatsrv

Text chat server over TCP connections.

## Prerequisites

To build server locally by your own you need to install [Golang](https://golang.org/) distribution for your platform.

Or you may use ready [Docker](https://www.docker.com/) image of the chat server. In that case you need to install _docker_, see image link below.

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

1. Check this repository to select available server images: https://cloud.docker.com/u/wtask/repository/docker/wtask/chatsrv

2. Run server as docker container:

```CLI
docker run -d --rm -p 20000:20000 --name tcpchat wtask/chat:x.y.z
```

where:
* `20000` is a chat server port
* `wtask/chat:x.y.z` is an available docker image.

## CLI options

Run server with option `-help` to get help for available server options

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