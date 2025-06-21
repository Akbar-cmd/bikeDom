FROM golang:1.24.4-alpine3.21 AS builder

COPY . /bikeDomain/source/
WORKDIR /bikeDomain/source/

RUN go mod download
RUN go build -o ./bin/bikeServer cmd/gRPC/grpc.go

FROM alpine:3.21

WORKDIR /root/
COPY --from=builder /bikeDomain/source/bin/bikeServer .



CMD ["./bikeServer"]