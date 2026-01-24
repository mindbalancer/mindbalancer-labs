# MindBalancer Examples

This directory contains example code showing how to use MindBalancer with different languages and tools.

## Prerequisites

1. MindBalancer running on `localhost:6034`
2. At least one AI provider configured (OpenAI, Anthropic, etc.)

## Examples

### Python

```bash
# Install OpenAI SDK
pip install openai

# Run example
python python_example.py
```

### Node.js

```bash
# Install OpenAI SDK
npm install openai

# Run example
node nodejs_example.js
```

### curl

```bash
# Make executable
chmod +x curl_examples.sh

# Run examples
./curl_examples.sh
```

## Key Point

**The only change needed is the base URL!**

Instead of:
```python
client = OpenAI()  # Uses api.openai.com
```

Use:
```python
client = OpenAI(
    base_url="http://localhost:6034/v1",
    api_key="not-needed"
)
```

MindBalancer handles:
- ✅ API key management
- ✅ Load balancing across providers
- ✅ Automatic failover
- ✅ Model-based routing (gpt-* → OpenAI, claude-* → Anthropic)
- ✅ Rate limiting
- ✅ Metrics and monitoring
- ✅ **Referee Mode** - Consensus-based responses from multiple AI providers

## Referee Mode

Referee Mode sends the same query to multiple AI providers in parallel, then uses a "referee" model to synthesize the best answer from all responses. This is ideal for:

- Critical decisions requiring high accuracy
- Reducing hallucination risk
- Getting consensus on complex topics

### Example (curl)

```bash
curl -X POST "http://localhost:6034/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "What is the CAP theorem?"}],
    "referee_mode": {
      "enabled": true,
      "referee_model": "gpt-4o",
      "providers": ["openai", "anthropic", "google"],
      "min_responses": 2
    }
  }'
```

### Referee Mode Options

| Option | Description | Default |
|--------|-------------|---------|
| `enabled` | Enable referee mode | `false` |
| `referee_model` | Model to synthesize responses | Config default |
| `providers` | Provider types to query (empty = all) | All available |
| `min_responses` | Minimum successful responses | `2` |
| `timeout_ms` | Per-provider timeout | Config default |

### Response

The response includes a `referee_info` object with metadata:

```json
{
  "choices": [...],
  "referee_info": {
    "providers_queried": 3,
    "successful_responses": 3,
    "failed_providers": [],
    "referee_model": "gpt-4o",
    "synthesis_latency_ms": 2345
  }
}
```
