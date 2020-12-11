# build stage
FROM golang:1.15-alpine AS build-env

RUN apk update && apk add ca-certificates

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN adduser -D -u 10001 xray
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' \
    -o xray ./cmd/tracing

FROM scratch
COPY --from=build-env /workspace/xray .
COPY --from=build-env /etc/passwd /etc/passwd
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY pkg/cfg.yaml /etc/amazon/xray/cfg.yaml
USER xray
ENTRYPOINT ["/xray"]
