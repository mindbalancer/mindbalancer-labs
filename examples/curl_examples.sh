#!/bin/bash
# MindBalancer curl Examples
# ==========================

MINDBALANCER_URL="http://localhost:6034"

echo "=== Simple Chat Completion ==="
curl -s -X POST "$MINDBALANCER_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "What is 2+2?"}],
    "max_tokens": 50
  }' | jq -r '.choices[0].message.content'

echo ""
echo "=== Streaming Response ==="
curl -s -X POST "$MINDBALANCER_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Count from 1 to 5"}],
    "max_tokens": 50,
    "stream": true
  }'

echo ""
echo ""
echo "=== Using Claude (routed to Anthropic) ==="
curl -s -X POST "$MINDBALANCER_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-haiku-20240307",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 50
  }' | jq -r '.choices[0].message.content'

echo ""
echo "=== List Models ==="
curl -s "$MINDBALANCER_URL/v1/models" | jq '.data[].id'

echo ""
echo "=== Health Check ==="
curl -s "$MINDBALANCER_URL/health" | jq

echo ""
echo "=== Referee Mode (Consensus-based response) ==="
echo "Sending same query to multiple AI providers, then synthesizing the best answer..."
curl -s -X POST "$MINDBALANCER_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "What are the main differences between Kubernetes and Docker Swarm?"}],
    "max_tokens": 500,
    "referee_mode": {
      "enabled": true,
      "referee_model": "gpt-4o",
      "providers": ["openai", "anthropic"],
      "min_responses": 2
    }
  }' | jq '{
    content: .choices[0].message.content,
    referee_info: .referee_info
  }'

echo ""
echo "=== Referee Mode with All Providers ==="
echo "Query all available providers and get the most accurate answer..."
curl -s -X POST "$MINDBALANCER_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Explain the CAP theorem in distributed systems"}],
    "max_tokens": 500,
    "referee_mode": {
      "enabled": true,
      "referee_model": "claude-3-5-sonnet-20241022",
      "min_responses": 2
    }
  }' | jq '{
    content: .choices[0].message.content[0:200],
    providers_queried: .referee_info.providers_queried,
    successful_responses: .referee_info.successful_responses,
    synthesis_latency_ms: .referee_info.synthesis_latency_ms
  }'
