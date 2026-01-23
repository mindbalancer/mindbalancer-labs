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
