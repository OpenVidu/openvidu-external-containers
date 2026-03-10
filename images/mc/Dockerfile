FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mc .

FROM busybox:stable
RUN addgroup -S mc && adduser -S -D -H -h /home/mc -s /sbin/nologin -G mc mc
COPY --from=builder /mc /usr/local/bin/mc
RUN mkdir /home/mc && chown mc:mc /home/mc
USER mc
ENTRYPOINT ["/usr/local/bin/mc"]
