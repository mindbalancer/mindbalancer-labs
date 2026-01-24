# MindBalancer

<p align="center">
  <strong>The ProxySQL for AI — High-performance load balancer for LLM APIs</strong>
</p>

<p align="center">
  <a href="https://www.mindbalancer.org">🌐 Website</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="https://www.mindbalancer.org/docs">📖 Docs</a>
</p>

<p align="center">
  <a href="https://www.mindbalancer.org"><img src="https://img.shields.io/badge/Website-mindbalancer.org-7c3aed?style=flat-square" alt="Website"></a>
  <a href="https://github.com/mindbalancer/mindbalancer-labs/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License"></a>
  <a href="mailto:burak1607@gmail.com"><img src="https://img.shields.io/badge/Contact-burak1607%40gmail.com-blue?style=flat-square" alt="Contact"></a>
</p>

---

## What is MindBalancer?

MindBalancer is a high-performance, on-premise load balancer and reverse proxy for AI/LLM APIs. Think of it as **ProxySQL, but for AI models** — it provides intelligent request routing, automatic failover, connection pooling, and comprehensive monitoring for your AI infrastructure.

```
┌─────────────────┐     ┌─────────────────────────────────────┐     ┌──────────────┐
│   Your App      │────▶│           MindBalancer              │────▶│   OpenAI     │
│                 │     │                                     │────▶│   Anthropic  │
│  (OpenAI SDK)   │◀────│  • Load Balancing                   │────▶│   Ollama     │
│                 │     │  • Automatic Failover               │────▶│   Azure      │
└─────────────────┘     │  • Rate Limiting                    │────▶│   Groq       │
                        │  • Request Routing                  │────▶│   Custom     │
                        └─────────────────────────────────────┘     └──────────────┘
```

### Why MindBalancer?

| Challenge | MindBalancer Solution |
|-----------|----------------------|
| Single point of failure with one AI provider | Automatic failover across multiple providers |
| Vendor lock-in | Provider-agnostic API (OpenAI-compatible) |
| No visibility into AI usage | Comprehensive metrics, Web UI dashboard, and query logging |
| Complex client-side load balancing | Centralized, weighted load distribution |
| Managing multiple API keys | Single entry point with encrypted key storage |
| Rate limit management | Built-in rate limiting per user/application |

---

## Features

### 🔄 Intelligent Load Balancing
- **Weighted Round-Robin** — Distribute traffic based on provider capacity
- **Least Connections** — Route to the least busy server
- **Latency-based** — Prefer faster responding providers
- **Hostgroups** — Logical grouping (e.g., premium vs. economy models)

### 🛡️ High Availability
- **Automatic Failover** — Seamlessly switch to healthy providers
- **Circuit Breaker** — Prevent cascade failures
- **Health Checks** — Continuous provider monitoring
- **Retry with Backoff** — Smart retry logic with exponential backoff

### 🎯 Request Routing
- **Model-based Routing** — Route specific models to specific providers
- **Pattern Matching** — Route based on prompt content
- **A/B Testing** — Mirror requests for comparison
- **Sticky Sessions** — Maintain conversation context

### 📊 Observability
- **Web UI Dashboard** — Real-time monitoring interface
- **Real-time Metrics** — Prometheus-compatible metrics endpoint
- **Query Logging** — Detailed request/response logging
- **Statistics Tables** — SQL-queryable stats (ProxySQL-style)
- **Grafana Dashboard** — Pre-built visualization

### 🚦 Rate Limiting
- **Per-user limits** — Control requests and tokens per minute
- **Global limits** — Default limits for all users
- **Rate limit headers** — `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### 💾 Response Caching
- **Automatic caching** — Cache deterministic requests (temperature=0)
- **LRU eviction** — Least Recently Used cache management
- **TTL expiration** — 5 minute default TTL
- **Cache headers** — `X-Cache: HIT` or `X-Cache: MISS`
- **Admin control** — Enable/disable/clear via mindsql or API

---

## Installation

### Requirements

- **Go 1.20+** ([download](https://go.dev/dl/))
- **Make** (usually pre-installed on macOS/Linux)

### From Source

```bash
# 1. Clone the repository
git clone https://github.com/mindbalancer/mindbalancer-labs.git
cd mindbalancer

# 2. Build binaries
make build

# 3. Verify installation
./bin/mindbalancer -version
./bin/mindsql -version
```

This creates:
- `./bin/mindbalancer` — Main server
- `./bin/mindsql` — Admin CLI

---

## Quick Start

### 1. Create Configuration

```bash
# Copy example config
cp configs/mindbalancer.example.cnf mindbalancer.cnf
```

Edit `mindbalancer.cnf` with minimum settings:

```ini
[mindbalancer]
# Network - localhost only for development
admin_bind_address = 127.0.0.1
admin_port = 6032
proxy_bind_address = 127.0.0.1
proxy_port = 6034
admin_http_port = 6033

# Storage - use /tmp for quick testing (no sudo needed)
data_dir = /tmp/mindbalancer

# Logging
log_level = info

# Health Check
health_check_enabled = true
health_check_interval_ms = 5000

# Metrics
prometheus_enabled = true
prometheus_port = 9090
```

### 2. Start MindBalancer

```bash
./bin/mindbalancer -config mindbalancer.cnf
```

You should see:
```
Starting MindBalancer version=xxx admin_port=6032 proxy_port=6034
Starting proxy server on 127.0.0.1:6034
Starting admin HTTP server on 127.0.0.1:6033
Starting admin MySQL server on 127.0.0.1:6032
Starting metrics server on :9090
```

**Services:**
| Port | Service | Description |
|------|---------|-------------|
| 6034 | Proxy API | OpenAI-compatible endpoint |
| 6033 | Admin HTTP | REST API + Web Dashboard |
| 6032 | Admin MySQL | mindsql CLI connection |
| 9090 | Metrics | Prometheus metrics |

### 3. Open Web Dashboard

Open in browser: **http://localhost:6033/**

### 4. Add Your First Server

Connect with mindsql:

```bash
./bin/mindsql
```

Add an OpenAI server:

```sql
-- Add OpenAI
INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, hostgroup, weight)
VALUES ('openai-main', 'openai', 'https://api.openai.com', 'sk-proj-YOUR-API-KEY', 0, 5);

-- Add Anthropic (optional)
INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, hostgroup, weight)
VALUES ('anthropic-main', 'anthropic', 'https://api.anthropic.com', 'sk-ant-YOUR-API-KEY', 1, 5);

-- Add routing rules (route by model)
INSERT INTO ai_routing_rules (match_model, destination_hostgroup, priority)
VALUES ('gpt-*', 0, 10);

INSERT INTO ai_routing_rules (match_model, destination_hostgroup, priority)
VALUES ('claude-*', 1, 10);

-- Apply changes
LOAD AI SERVERS TO RUNTIME;
LOAD AI ROUTING RULES TO RUNTIME;

-- Verify
SELECT * FROM ai_servers;
SHOW HEALTH STATUS;
```

### 4. Use the API

Point your OpenAI SDK to MindBalancer:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:6034/v1",
    api_key="any-key"  # MindBalancer handles authentication
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

Or with curl:

```bash
curl -X POST http://localhost:6034/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## Web Dashboard

MindBalancer includes a built-in web dashboard for real-time monitoring.

**Access:** `http://localhost:6033/`

### Features:
- 📊 Server status and health monitoring
- 📈 Request statistics and latency metrics
- 🔄 Auto-refresh every 5 seconds
- 🌙 Beautiful dark theme UI

![Dashboard Preview](docs/dashboard-preview.png)

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                            MindBalancer                                │
│  ┌──────────────┐     ┌──────────────────────────────────────────┐    │
│  │   mindsql    │────▶│      Admin Interface (MySQL :6032)       │    │
│  │     CLI      │     │      HTTP API + Dashboard (:6033)        │    │
│  └──────────────┘     └──────────────────────────────────────────┘    │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                        Core Engine                               │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │  │
│  │  │   Config    │  │    Load     │  │      Health Check       │  │  │
│  │  │   Manager   │  │  Balancer   │  │    & Circuit Breaker    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │  │
│  │  │   Query     │  │    Rate     │  │     Provider Pool       │  │  │
│  │  │   Router    │  │   Limiter   │  │   (Connection Mgmt)     │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                OpenAI-Compatible API (:6034)                     │  │
│  │   POST /v1/chat/completions  |  POST /v1/embeddings             │  │
│  │   POST /v1/completions       |  GET  /v1/models                 │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │              Storage: SQLite (mindbalancer.db)                   │  │
│  └─────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Configuration

### Main Config File (`mindbalancer.cnf`)

```ini
[mindbalancer]
# Network
admin_bind_address = 127.0.0.1
admin_port = 6032
proxy_bind_address = 127.0.0.1
proxy_port = 6034
admin_http_port = 6033

# Storage
data_dir = /var/lib/mindbalancer

# Logging
log_level = info    # debug, info, warn, error

# Failover
failover_enabled = true
max_retries = 3
circuit_breaker_threshold = 5

# Health Check
health_check_interval_ms = 5000

# Rate Limiting
rate_limit_enabled = true
default_requests_per_minute = 1000
default_tokens_per_minute = 100000

# Metrics
prometheus_enabled = true
prometheus_port = 9090
```

See `configs/mindbalancer.example.cnf` for all options.

---

## mindsql Reference

mindsql is a MySQL-compatible CLI for managing MindBalancer.

### Features

| Key | Function |
|-----|----------|
| ↑ / ↓ | Navigate command history |
| Tab | Auto-complete commands |
| Ctrl+C | Cancel current input |

History is saved to `~/.mindsql_history`

### Server Management

```sql
-- List servers
SELECT * FROM ai_servers;

-- Show API keys (masked)
SHOW API KEYS;

-- Show health status
SHOW HEALTH STATUS;

-- Add server
INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, hostgroup, weight)
VALUES ('openai-1', 'openai', 'https://api.openai.com', 'sk-xxx', 0, 5);

-- Remove server
DELETE FROM ai_servers WHERE name = 'openai-1';

-- Apply changes
LOAD AI SERVERS TO RUNTIME;
```

### Routing Rules

```sql
-- List rules
SELECT * FROM ai_routing_rules;

-- Route GPT models to hostgroup 0
INSERT INTO ai_routing_rules (match_model, destination_hostgroup, priority)
VALUES ('gpt-*', 0, 10);

-- Route Claude models to hostgroup 1
INSERT INTO ai_routing_rules (match_model, destination_hostgroup, priority)
VALUES ('claude-*', 1, 10);

LOAD AI ROUTING RULES TO RUNTIME;
```

### Statistics & Monitoring

```sql
-- Server stats
SELECT * FROM stats_ai_servers;

-- Recent requests
SELECT * FROM stats_ai_requests;

-- Connection pool status
SHOW PROCESSLIST;

-- Summary
SHOW STATS;

-- Hostgroup overview
SHOW HOSTGROUPS;
```

### Admin Commands

```sql
SHOW PROCESSLIST;           -- Active requests
SHOW STATS;                 -- Summary statistics  
SHOW HOSTGROUPS;            -- Hostgroup overview
SHOW API KEYS;              -- API keys (masked)
SHOW HEALTH STATUS;         -- Server health
```

### Cache Management

```sql
-- View cache status and statistics
SHOW CACHE STATUS;

-- Enable caching
CACHE ENABLE;

-- Disable caching
CACHE DISABLE;

-- Clear all cached responses
CACHE CLEAR;
```

**Cache Status Output:**
```
+------------------+------------------+
| Variable         | Value            |
+------------------+------------------+
| status           | enabled          |
| hits             | 1523             |
| misses           | 342              |
| hit_rate         | 0.82             |
| evictions        | 12               |
| size_bytes       | 2451678          |
| item_count       | 156              |
+------------------+------------------+
```

---

## HTTP Admin API

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/servers` | List all servers |
| POST | `/api/servers` | Add a server |
| DELETE | `/api/servers/{name}` | Remove a server |
| GET | `/api/stats` | Get statistics |
| GET | `/api/health` | Health status |
| POST | `/api/reload` | Reload configuration |
| GET | `/api/cache` | Cache status & statistics |
| PUT | `/api/cache` | Enable/disable cache |
| POST | `/api/cache/clear` | Clear all cached responses |
| GET | `/` | Web Dashboard |

### Example: Add Server via API

```bash
curl -X POST http://localhost:6033/api/servers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "openai-backup",
    "provider_type": "openai",
    "endpoint": "https://api.openai.com",
    "api_key_encrypted": "sk-your-key",
    "hostgroup": 0,
    "weight": 3
  }'

# Reload to apply
curl -X POST http://localhost:6033/api/reload
```

### Example: Cache Management via API

```bash
# Get cache status
curl http://localhost:6033/api/cache

# Response:
# {
#   "enabled": true,
#   "hits": 1523,
#   "misses": 342,
#   "hit_rate": 0.816,
#   "evictions": 12,
#   "size_bytes": 2451678,
#   "item_count": 156
# }

# Disable cache
curl -X PUT http://localhost:6033/api/cache \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Enable cache
curl -X PUT http://localhost:6033/api/cache \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# Clear cache
curl -X POST http://localhost:6033/api/cache/clear
```

---

## Supported Providers

| Provider | Status | Streaming | Embeddings | Notes |
|----------|--------|-----------|------------|-------|
| OpenAI | ✅ Full | ✅ | ✅ | GPT-3.5, GPT-4, GPT-4o |
| Anthropic | ✅ Full | ✅ | ❌ | Claude 3, Claude 3.5 |
| Ollama | ✅ Full | ✅ | ✅ | Local models |
| Azure OpenAI | ✅ Full | ✅ | ✅ | Enterprise Azure |
| Groq | ✅ Full | ✅ | ❌ | Fast inference |
| Google AI | 🚧 Beta | ✅ | ✅ | Gemini models |
| AWS Bedrock | 🚧 Beta | ✅ | ✅ | Multiple providers |
| Custom | ✅ Full | ✅ | ✅ | Any OpenAI-compatible |

---

## Metrics & Monitoring

### Prometheus Metrics

MindBalancer exposes metrics at `http://localhost:9090/metrics`:

```
# Request metrics
mindbalancer_requests_total{server="openai-1", status="success"} 15420
mindbalancer_request_duration_seconds{server="openai-1", quantile="0.99"} 1.25
mindbalancer_tokens_total{server="openai-1", direction="input"} 5420000

# Server health
mindbalancer_server_status{server="openai-1"} 1
mindbalancer_circuit_breaker_state{server="openai-1"} 0

# Rate limiting
mindbalancer_rate_limit_remaining{user="default"} 950
```

### Grafana Dashboard

Import the pre-built dashboard from `grafana/mindbalancer-dashboard.json`

---

## Roadmap

- [x] Core load balancing (weighted round-robin)
- [x] OpenAI-compatible API
- [x] SQLite storage with ProxySQL-style tables
- [x] mindsql CLI with readline support
- [x] Health checks and failover
- [x] Rate limiting
- [x] Web UI Dashboard
- [x] Response caching (with enable/disable control)
- [x] Cost tracking metrics (per model/provider)
- [x] Retry with exponential backoff
- [x] API key encryption (AES-256)
- [x] Connection pooling
- [x] Graceful shutdown
- [x] Hot config reload (SIGHUP)
- [ ] Semantic caching (embedding-based)
- [ ] Cluster mode (multi-node)
- [ ] Request queuing

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Development setup
git clone https://github.com/mindbalancer/mindbalancer-labs.git
cd mindbalancer
make build
make test
```

---

## License

MIT License — See [LICENSE](LICENSE) for details.

---

## Support & Contact

- 🌐 **Website:** [www.mindbalancer.org](https://www.mindbalancer.org)
- 📖 **Documentation:** [www.mindbalancer.org/docs](https://www.mindbalancer.org/docs)
- 📧 **Email:** [burak1607@gmail.com](mailto:burak1607@gmail.com)
- 🐛 **Issues:** [GitHub Issues](https://github.com/mindbalancer/mindbalancer-labs/issues)
- 💬 **Discussions:** [GitHub Discussions](https://github.com/mindbalancer/mindbalancer-labs/discussions)

---

<p align="center">
  Built with ❤️ for the AI infrastructure community<br>
  <a href="https://www.mindbalancer.org">www.mindbalancer.org</a>
</p>
