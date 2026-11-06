FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mc .

FROM busybox:stable@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662
RUN addgroup -S mc && adduser -S -D -H -h /home/mc -s /sbin/nologin -G mc mc
COPY --from=builder /mc /usr/local/bin/mc
RUN mkdir /home/mc && chown mc:mc /home/mc
USER mc
ENTRYPOINT ["/usr/local/bin/mc"]
