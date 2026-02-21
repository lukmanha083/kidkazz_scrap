# === Build stage ===
FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o kidkazz .

# === Runtime stage ===
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    ca-certificates \
    fonts-liberation \
    libnss3 \
    libatk-bridge2.0-0 \
    libdrm2 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    && rm -rf /var/lib/apt/lists/*

ENV ROD_BROWSER_BIN=/usr/bin/chromium

RUN useradd -m -s /bin/sh appuser
USER appuser

COPY --from=builder /app/kidkazz /usr/local/bin/kidkazz

EXPOSE 8080

CMD ["kidkazz", "serve-http"]
