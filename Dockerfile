# syntax=docker/dockerfile:1.6
FROM golang:1.22 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags "-s -w" -o /out/claude-cli-gateway ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
	&& addgroup -S app \
	&& adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/claude-cli-gateway ./claude-cli-gateway
COPY configs/configs.example.json ./configs.json
COPY reporter ./reporter
COPY web/templates ./templates

RUN mkdir -p /app/logs /app/data \
	&& chown -R app:app /app

USER app
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["./claude-cli-gateway"]
