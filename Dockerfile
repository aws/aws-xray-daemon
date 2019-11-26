# build stage
FROM golang:1.13-alpine AS build-env

RUN apk update && apk add git ca-certificates && rm -rf /var/cache/apk/*

RUN mkdir -p /go/src/github.com/aws/aws-xray-daemon
WORKDIR /go/src/github.com/aws/aws-xray-daemon

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY  . .
RUN adduser -D -u 10001 xray
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' \
    -o daemon ./cmd/tracing/daemon.go ./cmd/tracing/tracing.go

FROM scratch
COPY --from=build-env /go/src/github.com/aws/aws-xray-daemon/daemon .
COPY --from=build-env /etc/passwd /etc/passwd
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY pkg/cfg.yaml /etc/amazon/xray/cfg.yaml
USER xray
ENTRYPOINT ["/daemon"]
