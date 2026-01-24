"""
MindBalancer Python Example
============================
Uses the standard OpenAI SDK - just change the base_url!
"""

from openai import OpenAI

# Connect to MindBalancer instead of OpenAI directly
client = OpenAI(
    base_url="http://localhost:6034/v1",
    api_key="not-needed"  # MindBalancer manages API keys
)

# Example 1: Simple chat completion
print("=== Simple Chat ===")
response = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "user", "content": "What is MindBalancer?"}
    ],
    max_tokens=100
)
print(response.choices[0].message.content)
print()

# Example 2: Streaming response
print("=== Streaming ===")
stream = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "user", "content": "Count from 1 to 5"}
    ],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
print("\n")

# Example 3: Using Claude (routed to Anthropic automatically)
print("=== Claude via MindBalancer ===")
response = client.chat.completions.create(
    model="claude-3-haiku-20240307",  # Routes to Anthropic
    messages=[
        {"role": "user", "content": "Hello! Who are you?"}
    ],
    max_tokens=100
)
print(response.choices[0].message.content)
print()

# Example 4: Referee Mode - Consensus-based response
# Sends the same query to multiple AI providers and synthesizes the best answer
print("=== Referee Mode (Consensus) ===")
print("Querying multiple AI providers for consensus...")
import requests
import json

referee_response = requests.post(
    "http://localhost:6034/v1/chat/completions",
    headers={"Content-Type": "application/json"},
    json={
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "What are the key differences between REST and GraphQL APIs?"}
        ],
        "max_tokens": 500,
        "referee_mode": {
            "enabled": True,
            "referee_model": "gpt-4o",  # Model used to synthesize responses
            "providers": ["openai", "anthropic"],  # Providers to query
            "min_responses": 2  # Minimum successful responses required
        }
    }
)

if referee_response.status_code == 200:
    data = referee_response.json()
    print(f"Synthesized Answer: {data['choices'][0]['message']['content'][:300]}...")
    if 'referee_info' in data:
        info = data['referee_info']
        print(f"\nReferee Info:")
        print(f"  - Providers queried: {info['providers_queried']}")
        print(f"  - Successful responses: {info['successful_responses']}")
        print(f"  - Synthesis latency: {info['synthesis_latency_ms']}ms")
        if info.get('failed_providers'):
            print(f"  - Failed providers: {info['failed_providers']}")
else:
    print(f"Error: {referee_response.status_code} - {referee_response.text}")
