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

RUN apk add --no-cache ca-certificates curl bash tar nodejs npm python3 \
	&& npm install -g \
		@anthropic-ai/claude-code \
		@openai/codex \
	&& npm cache clean --force \
	&& UV_INSTALL_DIR=/usr/local/bin curl -fsSL https://astral.sh/uv/install.sh | bash \
	&& addgroup -S app \
	&& adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/claude-cli-gateway ./claude-cli-gateway
COPY configs/configs.example.json ./configs.json
COPY reporter ./reporter
COPY web/templates ./templates

RUN mkdir -p /app/logs /app/data /home/app \
	&& chown -R app:app /app /home/app

USER app
ENV HOME=/home/app
ENV PATH="/usr/local/bin:/home/app/.local/bin:${PATH}"

RUN mkdir -p "$HOME/.claude" "$HOME/.codex" "$HOME/.cursor" "$HOME/.cursor-agent"

RUN curl -fsS https://cursor.com/install -o /tmp/cursor-install.sh \
	&& bash /tmp/cursor-install.sh \
	&& rm /tmp/cursor-install.sh \
	&& command -v cursor-agent >/dev/null

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["./claude-cli-gateway"]
