# MindBalancer

<p align="center">
  <strong>The ProxySQL for AI — High-performance load balancer for LLM APIs</strong>
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#contributing">Contributing</a>
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
| No visibility into AI usage | Comprehensive metrics and query logging |
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
- **Real-time Metrics** — Prometheus-compatible metrics endpoint
- **Query Logging** — Detailed request/response logging
- **Statistics Tables** — SQL-queryable stats (ProxySQL-style)
- **Grafana Dashboard** — Pre-built visualization

---

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mindbalancer/mindbalancer.git
cd mindbalancer

# Build
make build

# Install (optional)
sudo make install
```

### Using Go

```bash
go install github.com/mindbalancer/mindbalancer/cmd/mindbalancer@latest
go install github.com/mindbalancer/mindbalancer/cmd/mindsql@latest
```

---

## Quick Start

### 1. Start MindBalancer

```bash
./bin/mindbalancer
```

This starts:
- **Proxy API** on port `6033` (OpenAI-compatible)
- **Admin Interface** on port `6032` (MySQL protocol)
- **Metrics** on port `9090` (Prometheus)

### 2. Add Your First Server

Connect with mindsql:

```bash
./bin/mindsql -h 127.0.0.1 -P 6032
```

Add an OpenAI server:

```sql
INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, weight)
VALUES ('openai-1', 'openai', 'https://api.openai.com', 'sk-your-api-key', 5);

LOAD AI SERVERS TO RUNTIME;
```

### 3. Use the API

Point your OpenAI SDK to MindBalancer:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:6033/v1",
    api_key="any-key"  # MindBalancer handles authentication
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                            MindBalancer                                │
│  ┌──────────────┐     ┌──────────────────────────────────────────┐    │
│  │   mindsql    │────▶│            Admin Interface               │    │
│  │     CLI      │     │         (MySQL Protocol :6032)           │    │
│  └──────────────┘     └──────────────────────────────────────────┘    │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                        Core Engine                               │  │
│  │                                                                  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │  │
│  │  │   Config    │  │    Load     │  │      Health Check       │  │  │
│  │  │   Manager   │  │  Balancer   │  │    & Circuit Breaker    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │  │
│  │                                                                  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │  │
│  │  │   Query     │  │   Stats     │  │     Provider Pool       │  │  │
│  │  │   Router    │  │  Collector  │  │   (Connection Mgmt)     │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                OpenAI-Compatible API (:6033)                     │  │
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
admin_bind_address = 0.0.0.0
admin_port = 6032
proxy_bind_address = 0.0.0.0
proxy_port = 6033

# Storage
data_dir = /var/lib/mindbalancer

# Logging
log_level = info    # debug, info, warn, error

# Failover
failover_enabled = true
max_retries = 3
circuit_breaker_threshold = 3

# Health Check
health_check_interval_ms = 5000

# Metrics
prometheus_enabled = true
prometheus_port = 9090
```

### Runtime Configuration via mindsql

```sql
-- View all variables
SELECT * FROM global_variables;

-- Change settings
SET ai-max-retries = 5;
SET ai-health-check-interval = 10000;

-- Apply changes
LOAD VARIABLES TO RUNTIME;
```

---

## mindsql Reference

### Server Management

```sql
-- List servers
SELECT * FROM ai_servers;

-- Add server
INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, weight)
VALUES ('server-name', 'openai', 'https://api.openai.com', 'sk-xxx', 5);

-- Update weight (shift traffic)
UPDATE ai_servers SET weight = 10 WHERE name = 'server-name';

-- Disable server
UPDATE ai_servers SET status = 'OFFLINE' WHERE name = 'server-name';

-- Remove server
DELETE FROM ai_servers WHERE name = 'server-name';

-- Apply changes
LOAD AI SERVERS TO RUNTIME;
```

### User Management

```sql
-- List users
SELECT * FROM ai_users;

-- Add user with rate limits
INSERT INTO ai_users (username, password_hash, max_requests_per_minute, max_tokens_per_minute)
VALUES ('app-name', SHA256('password'), 100, 100000);

-- Update limits
UPDATE ai_users SET max_requests_per_minute = 200 WHERE username = 'app-name';

LOAD AI USERS TO RUNTIME;
```

### Routing Rules

```sql
-- Route specific models
INSERT INTO ai_routing_rules (match_model, destination_hostgroup)
VALUES ('gpt-4*', 1);

-- Route by prompt pattern
INSERT INTO ai_routing_rules (match_pattern, destination_hostgroup, priority)
VALUES ('^(code|program|debug)', 1, 50);

LOAD AI ROUTING RULES TO RUNTIME;
```

### Statistics

```sql
-- Server stats
SELECT * FROM stats_ai_servers;

-- Recent requests
SELECT * FROM stats_ai_requests ORDER BY timestamp DESC LIMIT 20;

-- Connection pool status
SELECT * FROM runtime_ai_servers;
```

### Admin Commands

```sql
SHOW PROCESSLIST;           -- Active requests
SHOW STATS;                 -- Summary statistics  
SHOW HOSTGROUPS;            -- Hostgroup overview
KILL CONNECTION <id>;       -- Terminate request
FLUSH LOGS;                 -- Rotate log files
SHUTDOWN;                   -- Graceful shutdown
```

---

## Supported Providers

| Provider | Status | Streaming | Embeddings | Notes |
|----------|--------|-----------|------------|-------|
| OpenAI | ✅ Full | ✅ | ✅ | GPT-3.5, GPT-4, etc. |
| Anthropic | ✅ Full | ✅ | ❌ | Claude models |
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

# Connection pool
mindbalancer_connections_active{server="openai-1"} 45
mindbalancer_connections_idle{server="openai-1"} 55
```

### Grafana Dashboard

Import our pre-built dashboard from `grafana/mindbalancer-dashboard.json`

---

## Roadmap

- [x] Core load balancing (weighted round-robin)
- [x] OpenAI-compatible API
- [x] SQLite storage with ProxySQL-style tables
- [x] mindsql CLI
- [x] Health checks and failover
- [ ] Response caching
- [ ] Semantic caching (embedding-based)
- [ ] Web UI dashboard
- [ ] Cluster mode (multi-node)
- [ ] Request queuing
- [ ] Cost tracking and budgets

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Development setup
git clone https://github.com/mindbalancer/mindbalancer.git
cd mindbalancer
make dev-setup
make test
make run
```

---

## License

Apache License 2.0 — See [LICENSE](LICENSE) for details.

---

## Support

- 📖 [Documentation](https://docs.mindbalancer.io)
- 💬 [Discord Community](https://discord.gg/mindbalancer)
- 🐛 [Issue Tracker](https://github.com/mindbalancer/mindbalancer/issues)
- 📧 [Email](mailto:support@mindbalancer.io)

---

<p align="center">
  Built with ❤️ for the AI infrastructure community
</p>
