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
    console.log();

    // Example 4: Referee Mode - Consensus-based response
    // Sends the same query to multiple AI providers and synthesizes the best answer
    console.log('=== Referee Mode (Consensus) ===');
    console.log('Querying multiple AI providers for consensus...');
    
    const refereeResponse = await fetch('http://localhost:6034/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            model: 'gpt-4o-mini',
            messages: [
                { role: 'user', content: 'What are the key differences between REST and GraphQL APIs?' }
            ],
            max_tokens: 500,
            referee_mode: {
                enabled: true,
                referee_model: 'gpt-4o',  // Model used to synthesize responses
                providers: ['openai', 'anthropic'],  // Providers to query
                min_responses: 2  // Minimum successful responses required
            }
        })
    });

    if (refereeResponse.ok) {
        const data = await refereeResponse.json();
        console.log('Synthesized Answer:', data.choices[0].message.content.substring(0, 300) + '...');
        if (data.referee_info) {
            console.log('\nReferee Info:');
            console.log(`  - Providers queried: ${data.referee_info.providers_queried}`);
            console.log(`  - Successful responses: ${data.referee_info.successful_responses}`);
            console.log(`  - Synthesis latency: ${data.referee_info.synthesis_latency_ms}ms`);
            if (data.referee_info.failed_providers?.length > 0) {
                console.log(`  - Failed providers: ${data.referee_info.failed_providers.join(', ')}`);
            }
        }
    } else {
        console.log(`Error: ${refereeResponse.status} - ${await refereeResponse.text()}`);
    }
}

main().catch(console.error);
