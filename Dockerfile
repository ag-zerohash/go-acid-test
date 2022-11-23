ARG GO_VER="1.19"

FROM docker.infra.seedcx.net/golang:${GO_VER}-alpine as builder

ARG VER="HEAD"
ARG NOW
ARG GOPROXY="https://go.infra.seedcx.net"

ENV GOPROXY="${GOPROXY}" \
    CGO_ENABLED=0

WORKDIR /go/src

COPY go.* ./
RUN go mod download -x

COPY . .
RUN go build -v -o ./go-acid-test ./go-acid-test.go

CMD ["/go/src/go-acid-test"]

