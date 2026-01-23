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
