# KidKazz Scrap

Go CLI tool and MCP server for scraping Indonesian marketplace data. Currently supports **Tokopedia**, with a pluggable architecture for adding more platforms.

Each product includes procurement and marketing fields: **original price**, **discount %**, **price range**, **promo labels** (Cashback, Flash Sale, etc.), **ad detection**, and **wishlist status** — useful for demand intelligence and price benchmarking.

## How It Works

Every scrape request runs through a **strategy fallback chain** — the tool tries the lightest method first and only escalates if it fails:

| Priority | Strategy | Method |
|----------|----------|--------|
| 1 | GraphQL | Hit Tokopedia's internal GraphQL API |
| 2 | Static | Fetch raw HTML, parse JSON-LD |
| 3 | Headless | Render with a headless browser (rod) |

Strategy 1 runs first as the **fast strategy**. Strategies 2 and 3 are **slow fallbacks** that only run sequentially if the fast strategy fails.

All HTTP requests pass through a **stealth pipeline**: robots.txt check, rate limiting, human-like delays, browser fingerprint rotation, and optional proxy routing.

## Requirements

- Go 1.21+
- Chromium-based browser (for headless strategy fallback — optional, auto-downloaded by rod)

## Installation

```bash
git clone https://github.com/lukmanha083/kidkazz_scrap.git
cd kidkazz_scrap
go build -o kidkazz .
```

Or run directly:

```bash
go run . search "laptop gaming"
```

## CLI Usage

### Search Products

```bash
# JSON output (default)
kidkazz search "laptop gaming" --limit 10

# Table output
kidkazz search "sepatu nike" --format table --page 2 --limit 20

# Filter out promoted/ad products — only organic results
kidkazz search "iphone 15" --format table --limit 20 --no-ads

# Specify platform explicitly
kidkazz search "iphone 15" --platform tokopedia --limit 5 --format json
```

### Trending Products

```bash
kidkazz trending --limit 10
kidkazz trending --category "elektronik" --format table

# Best sellers without paid promotion noise
kidkazz trending --category "kaos dalam anak" --format table --limit 20 --no-ads
```

### Discover Popular Categories

```bash
# Sample best-seller products and rank categories by count
kidkazz categories "action figure" --limit 60
```

Example output:

```
Popular categories for "action figure" (20 products sampled):

  1. Mainan Hobi > Figure > Action Figure                (5 products)
  2. Mainan Hobi > Figure > Figure Set                   (4 products)
  3. Mainan Hobi > Model Kit > Mecha Model Gunpla        (2 products)
  ...
```

### Start MCP Server (stdio)

```bash
kidkazz serve
```

This starts an MCP server over **stdio**, exposing three tools for use with Claude Desktop, Claude Code, or any MCP client.

### Start MCP Server (HTTP)

```bash
kidkazz serve-http --port 8080
```

Starts the MCP server over HTTP with optional Bearer token auth. Used for remote deployment (e.g. Fly.io). Set `KIDKAZZ_API_KEY` to enable authentication.

### Global Flags

These flags apply to all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--platform` | `tokopedia` | Target marketplace |
| `--delay-profile` | `normal` | Request delay: `cautious`, `normal`, `aggressive` |
| `--respect-robots` | `true` | Obey robots.txt rules |
| `--proxy-mode` | `direct` | Proxy backend: `direct`, `decodo`, `wireguard`, `custom` |
| `--wireguard-config` | | Path to WireGuard `.conf` file |
| `--proxy-file` | | Path to proxy list file (for `custom` mode) |

## MCP Server Setup

The `kidkazz serve` command runs an MCP server on stdio. It exposes three tools:

| Tool | Description | Required Params |
|------|-------------|-----------------|
| `search_products` | Search products by keyword | `keyword` |
| `get_trending` | Get trending/popular products | — |
| `product_detail` | Get full details for a product | `url` |

### Claude Code

Add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "kidkazz": {
      "command": "/path/to/kidkazz",
      "args": ["serve"],
      "env": {
        "KIDKAZZ_DELAY_PROFILE": "normal"
      }
    }
  }
}
```

### Claude Desktop

Add to your Claude Desktop config (`~/.config/claude-desktop/claude_desktop_config.json` on Linux, `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "kidkazz": {
      "command": "/path/to/kidkazz",
      "args": ["serve"],
      "env": {
        "KIDKAZZ_DELAY_PROFILE": "normal"
      }
    }
  }
}
```

Replace `/path/to/kidkazz` with the absolute path to the built binary.

### Remote HTTP Server

If deployed on Fly.io (or any HTTP host), configure with the URL-based transport:

```json
{
  "mcpServers": {
    "kidkazz": {
      "type": "streamable-http",
      "url": "https://kidkazz-scrap.fly.dev/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_KEY"
      }
    }
  }
}
```

### Tool Parameters

**search_products**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `keyword` | string | *(required)* | Search keyword |
| `platform` | string | `tokopedia` | Target platform |
| `page` | number | `1` | Page number |
| `limit` | number | `20` | Results per page |

**get_trending**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `platform` | string | `tokopedia` | Target platform |
| `category` | string | | Category filter |
| `limit` | number | `10` | Number of results |

**product_detail**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | *(required)* | Product page URL |

## Configuration

Configuration is loaded in order: **defaults** -> **`.env` file** -> **environment variables** -> **CLI flags**. Later sources override earlier ones.

A `.env` file in the working directory is loaded automatically at startup (if present). Variables already set in the environment take precedence over `.env` values.

### Environment Variables

**General**

| Variable | Default | Description |
|----------|---------|-------------|
| `KIDKAZZ_PLATFORM` | `tokopedia` | Default marketplace platform |
| `KIDKAZZ_DELAY_PROFILE` | `normal` | Delay profile: `cautious` (2-5s), `normal` (0.5-2s), `aggressive` (200-800ms) |
| `KIDKAZZ_RESPECT_ROBOTS` | `true` | Set to `false` to skip robots.txt checks |

**Rate Limiting**

| Variable | Default | Description |
|----------|---------|-------------|
| `KIDKAZZ_RATE_PER_SECOND` | `2.0` | Max requests per second |
| `KIDKAZZ_RATE_BURST` | `3` | Burst size for rate limiter |
| `KIDKAZZ_MAX_CONCURRENT` | `5` | Max concurrent page fetches |

**Proxy**

| Variable | Default | Description |
|----------|---------|-------------|
| `KIDKAZZ_PROXY_MODE` | `direct` | Proxy mode: `direct`, `decodo`, `wireguard`, `custom` |
| `DECODO_USERNAME` | | Decodo residential proxy username |
| `DECODO_PASSWORD` | | Decodo residential proxy password |
| `DECODO_COUNTRY` | `id` | Decodo country code (Indonesia) |
| `DECODO_CITY` | | Decodo city targeting (e.g. `jakarta`) |
| `KIDKAZZ_WG_CONFIG` | | Path to WireGuard config file |
| `KIDKAZZ_PROXIES` | | Path to proxy list file |

### Example `.env` File

```bash
# Stealth settings
KIDKAZZ_DELAY_PROFILE=normal
KIDKAZZ_RATE_PER_SECOND=2.0
KIDKAZZ_RATE_BURST=3
KIDKAZZ_MAX_CONCURRENT=5
KIDKAZZ_RESPECT_ROBOTS=true

# Decodo residential proxy (optional)
KIDKAZZ_PROXY_MODE=decodo
DECODO_USERNAME=your_username
DECODO_PASSWORD=your_password
DECODO_COUNTRY=id
DECODO_CITY=jakarta
```

> Place this `.env` file in the directory where you run the `kidkazz` binary. It is loaded automatically at startup.

**HTTP Server**

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port (set automatically by Fly.io) |
| `KIDKAZZ_API_KEY` | | Bearer token for HTTP MCP auth (empty = no auth) |

### Proxy Modes

**`direct`** (default) — No proxy. Relies on fingerprint rotation, delays, and rate limiting for stealth.

**`decodo`** — Decodo (formerly SmartProxy) residential proxies. Requires `DECODO_USERNAME` and `DECODO_PASSWORD`. Each request gets a different Indonesian IP. Set `DECODO_CITY=jakarta` for city-level targeting.

**`wireguard`** — Route traffic through a WireGuard VPN tunnel (e.g. ProtonVPN). Set `KIDKAZZ_WG_CONFIG` or `--wireguard-config` to the `.conf` file path.

**`custom`** — Load proxy URLs from a file. Set `KIDKAZZ_PROXIES` or `--proxy-file`. One `http://`, `https://`, or `socks5://` URL per line.

### Delay Profiles

Controls the random delay between requests (on top of the rate limiter):

| Profile | Min | Max | Use Case |
|---------|-----|-----|----------|
| `cautious` | 2s | 5s | Low-risk, long-running scrapes |
| `normal` | 500ms | 2s | General use |
| `aggressive` | 200ms | 800ms | Speed-focused, higher detection risk |

## Deploy to Fly.io

Deploy as an HTTP MCP server with auto-stop/start to minimize cost (~$2-3/month).

### 1. Install Fly CLI and authenticate

```bash
curl -L https://fly.io/install.sh | sh
fly auth login
```

### 2. Create the app

```bash
fly launch --no-deploy
```

Accept the defaults — the `fly.toml` is already configured for Singapore region (`sin`), closest to Indonesia.

### 3. Set the API key secret

```bash
fly secrets set KIDKAZZ_API_KEY=$(openssl rand -hex 32)
```

Save this key — you'll need it for MCP client configuration. The key is stored encrypted in Fly.io and injected as an env var at runtime.

### 4. Deploy

```bash
fly deploy
```

First deploy takes ~3-5 minutes (builds Docker image with Chromium).

### 5. Verify

```bash
# Health check (no auth)
curl https://kidkazz-scrap.fly.dev/healthz
# → {"status":"ok"}

# MCP call with auth
curl -X POST https://kidkazz-scrap.fly.dev/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

### 6. Monitor

```bash
fly status    # machine state
fly logs      # live logs
```

The machine auto-stops when idle and wakes on the next request (3-5s cold start).

## Project Structure

```
kidkazz_scrap/
├── main.go                         # Entry point
├── Dockerfile                      # Multi-stage build (Go + Chromium)
├── fly.toml                        # Fly.io deployment config
├── cmd/
│   ├── root.go                     # CLI root, global flags, platform init
│   ├── search.go                   # search subcommand
│   ├── trending.go                 # trending subcommand
│   ├── categories.go               # categories subcommand
│   ├── format.go                   # Shared table formatting helpers
│   ├── serve.go                    # serve subcommand (MCP stdio)
│   └── serve_http.go               # serve-http subcommand (MCP HTTP)
├── mcp/
│   ├── server.go                   # MCP stdio server
│   ├── server_http.go              # MCP HTTP server (StreamableHTTP + auth)
│   └── tools.go                    # Tool definitions + handlers
├── internal/
│   ├── platform/
│   │   ├── platform.go             # Scraper/Strategy interfaces
│   │   ├── progress.go             # Context-based progress callback
│   │   └── registry.go             # Platform registry
│   ├── ui/
│   │   └── spinner.go              # CLI progress spinner (stderr)
│   ├── models/
│   │   └── models.go               # Product, Shop, Label types
│   ├── tokopedia/
│   │   ├── tokopedia.go            # Scraper orchestration (strategy racing)
│   │   ├── graphql.go              # Strategy 1: GraphQL API (fast)
│   │   ├── static.go               # Strategy 2: HTML + JSON-LD
│   │   ├── queries.go              # GraphQL query strings
│   │   └── headless.go             # Strategy 3: Headless browser
│   ├── httputil/
│   │   ├── client.go               # HTTP client, retry, decompression
│   │   └── headers.go              # Browser-like header sets
│   └── stealth/
│       ├── transport.go            # StealthTransport (RoundTripper pipeline)
│       ├── robots.go               # robots.txt compliance
│       ├── fingerprint.go          # Browser fingerprint rotation
│       ├── delay.go                # Human-like random delays
│       └── proxy.go                # Proxy rotation (Decodo, HTTP, SOCKS5)
└── config/
    └── config.go                   # Config from env + flags
```

## License

MIT
