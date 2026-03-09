FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /minio-bucket-checker .

FROM busybox:stable
COPY --from=builder /minio-bucket-checker /usr/local/bin/minio-bucket-checker
ENTRYPOINT ["/usr/local/bin/minio-bucket-checker"]
