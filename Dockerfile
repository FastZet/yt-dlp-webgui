# syntax=docker/dockerfile:1

FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/yt-dlp-webgui ./cmd/yt-dlp-webgui

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    python3 \
    python3-pip \
    ffmpeg \
    curl \
    unzip \
 && rm -rf /var/lib/apt/lists/*

# Install official yt-dlp in the runtime image
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    -o /usr/local/bin/yt-dlp \
 && chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app

COPY --from=builder /out/yt-dlp-webgui /app/yt-dlp-webgui

# Runtime data directories
RUN mkdir -p /app/data/downloads /app/data/runtime/custom /app/data/uploads /app/data/logs

EXPOSE 8080

CMD ["/app/yt-dlp-webgui"]
