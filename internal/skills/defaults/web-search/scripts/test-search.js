#!/usr/bin/env node
/**
 * Integration test for Bridge Server and Bing Search
 */

const BridgeServer = require('../dist/server/index').default;

async function testSearchIntegration() {
  console.log('\n=== Web Search Skill - Integration Test ===\n');

  const server = new BridgeServer();
  let connectionId = null;

  try {
    // Start Bridge Server
    console.log('Step 1: Starting Bridge Server...');
    await server.start();
    console.log('✓ Bridge Server started\n');

    // Wait a moment for server to be ready
    await new Promise(resolve => setTimeout(resolve, 1000));

    // Launch browser via API
    console.log('Step 2: Launching browser via API...');
    const launchResponse = await fetch('http://127.0.0.1:8923/api/browser/launch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    const launchData = await launchResponse.json();
    console.log(`✓ Browser launched: PID ${launchData.data.pid}\n`);

    // Connect to browser
    console.log('Step 3: Connecting to browser...');
    const connectResponse = await fetch('http://127.0.0.1:8923/api/browser/connect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
    const connectData = await connectResponse.json();
    connectionId = connectData.data.connectionId;
    console.log(`✓ Connected: ${connectionId}\n`);

    // Perform search
    console.log('Step 4: Performing Bing search for "TypeScript tutorial"...');
    const searchResponse = await fetch('http://127.0.0.1:8923/api/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        connectionId,
        query: 'TypeScript tutorial',
        maxResults: 5
      })
    });
    const searchData = await searchResponse.json();
    console.log(`✓ Search completed in ${searchData.data.duration}ms\n`);

    // Display results
    console.log('Search Results:');
    console.log('─'.repeat(80));
    searchData.data.results.forEach((result, index) => {
      console.log(`\n${index + 1}. ${result.title}`);
      console.log(`   URL: ${result.url}`);
      console.log(`   Snippet: ${result.snippet.substring(0, 150)}...`);
    });
    console.log('\n' + '─'.repeat(80));

    // Take screenshot
    console.log('\nStep 5: Taking screenshot...');
    const screenshotResponse = await fetch('http://127.0.0.1:8923/api/page/screenshot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        connectionId,
        format: 'png'
      })
    });
    const screenshotData = await screenshotResponse.json();
    console.log(`✓ Screenshot captured: ${screenshotData.data.size} bytes\n`);

    // Get page text
    console.log('Step 6: Getting page text...');
    const textResponse = await fetch('http://127.0.0.1:8923/api/page/text', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ connectionId })
    });
    const textData = await textResponse.json();
    console.log(`✓ Page text retrieved: ${textData.data.text.length} chars\n`);

    // Check status
    console.log('Step 7: Checking server status...');
    const statusResponse = await fetch('http://127.0.0.1:8923/api/browser/status');
    const statusData = await statusResponse.json();
    console.log(`✓ Status: ${JSON.stringify(statusData.data, null, 2)}\n`);

    console.log('=== All integration tests passed! ===\n');
  } catch (error) {
    console.error('\n❌ Test failed:', error);
    process.exit(1);
  } finally {
    // Cleanup
    console.log('Cleaning up...');
    if (connectionId) {
      try {
        await fetch('http://127.0.0.1:8923/api/browser/disconnect', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ connectionId })
        });
      } catch (error) {
        console.warn('Failed to disconnect:', error.message);
      }
    }
    await server.stop();
    console.log('✓ Cleanup complete\n');
    process.exit(0);
  }
}

// Run test
testSearchIntegration().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
