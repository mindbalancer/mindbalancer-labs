/**
 * MindBalancer Node.js Example
 * ============================
 * Uses the standard OpenAI SDK - just change the baseURL!
 * 
 * Install: npm install openai
 */

import OpenAI from 'openai';

// Connect to MindBalancer instead of OpenAI directly
const client = new OpenAI({
    baseURL: 'http://localhost:6034/v1',
    apiKey: 'not-needed'  // MindBalancer manages API keys
});

async function main() {
    // Example 1: Simple chat completion
    console.log('=== Simple Chat ===');
    const response = await client.chat.completions.create({
        model: 'gpt-4o-mini',
        messages: [
            { role: 'user', content: 'What is MindBalancer?' }
        ],
        max_tokens: 100
    });
    console.log(response.choices[0].message.content);
    console.log();

    // Example 2: Streaming response
    console.log('=== Streaming ===');
    const stream = await client.chat.completions.create({
        model: 'gpt-4o-mini',
        messages: [
            { role: 'user', content: 'Count from 1 to 5' }
        ],
        stream: true
    });

    for await (const chunk of stream) {
        const content = chunk.choices[0]?.delta?.content;
        if (content) {
            process.stdout.write(content);
        }
    }
    console.log('\n');

    // Example 3: Using Claude (routed to Anthropic automatically)
    console.log('=== Claude via MindBalancer ===');
    const claudeResponse = await client.chat.completions.create({
        model: 'claude-3-haiku-20240307',  // Routes to Anthropic
        messages: [
            { role: 'user', content: 'Hello! Who are you?' }
        ],
        max_tokens: 100
    });
    console.log(claudeResponse.choices[0].message.content);
}

main().catch(console.error);
