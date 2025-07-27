#!/usr/bin/env node

/**
 * Simple JavaScript client to test the Alpine SSE endpoint
 * Usage: node sse-client.js [port]
 * Default port: 3001
 */

const port = process.argv[2] || 3001;
const url = `http://localhost:${port}/events`;

console.log(`Connecting to SSE endpoint at ${url}...`);

// Create EventSource connection
const eventSource = new EventSource(url);

// Handle connection open
eventSource.onopen = () => {
    console.log('✅ Connected to SSE endpoint');
};

// Handle incoming messages
eventSource.onmessage = (event) => {
    console.log('📨 Received event:', event.data);
    
    // Check if we received the expected "hello world" message
    if (event.data.includes('hello world')) {
        console.log('✅ Successfully received "hello world" event!');
        console.log('Test passed - closing connection...');
        eventSource.close();
        process.exit(0);
    }
};

// Handle errors
eventSource.onerror = (error) => {
    console.error('❌ Error:', error);
    eventSource.close();
    process.exit(1);
};

// Timeout after 5 seconds
setTimeout(() => {
    console.error('❌ Timeout: Did not receive expected event within 5 seconds');
    eventSource.close();
    process.exit(1);
}, 5000);

console.log('Waiting for events...');