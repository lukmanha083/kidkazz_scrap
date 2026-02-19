# KidKazz Scrap

Go CLI tool and MCP server for scraping Indonesian marketplace data. Currently supports **Tokopedia**, with a pluggable architecture for adding more platforms.

## How It Works

Every scrape request runs through a **strategy fallback chain** — the tool tries the lightest method first and only escalates if it fails:

| Priority | Strategy | Method |
|----------|----------|--------|
| 1 | Static | Fetch raw HTML, parse JSON-LD |
| 2 | GraphQL | Hit Tokopedia's internal GraphQL API |
| 3 | Headless | Render with a headless browser (rod) |

Strategies 1 and 2 race **concurrently** — whichever succeeds first wins. Strategy 3 only runs if both fast strategies fail.

All HTTP requests pass through a **stealth pipeline**: robots.txt check, rate limiting, human-like delays, browser fingerprint rotation, and optional proxy routing.

## Requirements

- Go 1.21+
- Chromium-based browser (for headless strategy fallback — optional, auto-downloaded by rod)

## Installation

```bash
git clone https://github.com/lukman83/kidkazz-scrap.git
cd kidkazz-scrap
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

# Specify platform explicitly
kidkazz search "iphone 15" --platform tokopedia --limit 5 --format json
```

### Trending Products

```bash
kidkazz trending --limit 10
kidkazz trending --category "elektronik" --format table
```

### Start MCP Server

```bash
kidkazz serve
```

This starts an MCP server over **stdio**, exposing three tools for use with Claude Desktop, Claude Code, or any MCP client.

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

Configuration is loaded in order: **defaults** -> **environment variables** -> **CLI flags**. Later sources override earlier ones.

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

> The app does not load `.env` files automatically. Source it in your shell (`source .env` or `export $(cat .env | xargs)`) or set the variables through your MCP server config's `env` block.

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

## Project Structure

```
kidkazz_scrap/
├── main.go                         # Entry point
├── cmd/
│   ├── root.go                     # CLI root, global flags, platform init
│   ├── search.go                   # search subcommand
│   ├── trending.go                 # trending subcommand
│   └── serve.go                    # serve subcommand (MCP stdio)
├── mcp/
│   ├── server.go                   # MCP server factory
│   └── tools.go                    # Tool definitions + handlers
├── internal/
│   ├── platform/
│   │   ├── platform.go             # Scraper/Strategy interfaces
│   │   └── registry.go             # Platform registry
│   ├── models/
│   │   └── models.go               # Product, Shop types
│   ├── tokopedia/
│   │   ├── tokopedia.go            # Scraper orchestration (strategy racing)
│   │   ├── static.go               # Strategy 1: HTML + JSON-LD
│   │   ├── graphql.go              # Strategy 2: GraphQL API
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
