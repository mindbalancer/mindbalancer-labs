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
