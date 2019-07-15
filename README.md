# Chat

Text chat over __standard__ Golang net.Conn __interface__. The code is based only on Golang standard library and built around internal chat broker. The broker is a core as for chat-server and as far as for a chat client.

Chat server:

- usable with `netcat`, `telnet` and similar utilities
- accepts connections from clients
- can relay message of each client to other clients
- can notify all clients when other client joins or leaves
- maintains timeouts for inactive users, disconnecting them
- maintains history of messages and join/part events

Server is available under `cmd\chatsrv`.

To play with chat you can use ready docker image.
