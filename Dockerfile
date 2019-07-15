FROM golang:1.12 as builder

ENV \
	# golang env
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64

WORKDIR /src
COPY ./ ./

RUN	groupadd -r goapp && useradd -r -g goapp goapp \
	&& go mod tidy \
	&& go mod download \
	&& mkdir /build \
	&& go build -a -ldflags '-extldflags "-static"' -o /build/chatsrv ./cmd/chatsrv/.

FROM scratch
LABEL maintainer="<Mike Mikhaylov, webtask@gmail.com>"

# import user accounts 
COPY --from=builder /etc/group /etc/passwd /etc/
# copy application binary
COPY --from=builder --chown=goapp:goapp /build/ /app/

USER goapp
WORKDIR /app

EXPOSE 20000

CMD ["/app/chatsrv"]
