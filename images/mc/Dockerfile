FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mc .

FROM busybox:stable
COPY --from=builder /mc /usr/local/bin/mc
ENTRYPOINT ["/usr/local/bin/mc"]
