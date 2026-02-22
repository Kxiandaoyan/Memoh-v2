#!/usr/bin/env node
/**
 * Basic test script for Playwright manager and browser launcher
 */

const { launchBrowser, closeBrowser } = require('../dist/server/playwright/browser');
const { PlaywrightManager } = require('../dist/server/playwright/manager');
const { defaultConfig } = require('../dist/server/config');

async function testBasicFunctionality() {
  console.log('\n=== Web Search Skill - Basic Functionality Test ===\n');

  let browserInstance = null;
  let connectionId = null;
  const manager = new PlaywrightManager();

  try {
    // Step 1: Launch browser
    console.log('Step 1: Launching browser...');
    browserInstance = await launchBrowser(defaultConfig.browser);
    console.log('✓ Browser launched successfully\n');

    // Step 2: Connect via Playwright
    console.log('Step 2: Connecting via Playwright...');
    connectionId = await manager.connectToCDP(browserInstance.cdpPort);
    console.log(`✓ Connected successfully (ID: ${connectionId})\n`);

    // Step 3: Get page
    console.log('Step 3: Getting page...');
    const page = await manager.getPage(connectionId);
    console.log(`✓ Page obtained: ${page.url()}\n`);

    // Step 4: Navigate to a test URL
    console.log('Step 4: Navigating to example.com...');
    await page.goto('https://example.com', { waitUntil: 'domcontentloaded', timeout: 15000 });
    console.log(`✓ Navigation complete: ${page.url()}\n`);

    // Step 5: Get page title
    console.log('Step 5: Getting page title...');
    const title = await page.title();
    console.log(`✓ Page title: "${title}"\n`);

    // Step 6: Take screenshot
    console.log('Step 6: Taking screenshot...');
    const screenshot = await page.screenshot({ type: 'png' });
    console.log(`✓ Screenshot captured (${screenshot.length} bytes)\n`);

    // Step 7: Get text content
    console.log('Step 7: Getting text content...');
    const text = await page.textContent('body');
    console.log(`✓ Text content: ${text?.substring(0, 100)}...\n`);

    console.log('=== All tests passed! ===\n');
  } catch (error) {
    console.error('\n❌ Test failed:', error);
    process.exit(1);
  } finally {
    // Cleanup
    console.log('Cleaning up...');
    if (connectionId) {
      await manager.disconnect(connectionId);
    }
    if (browserInstance) {
      await closeBrowser(browserInstance);
    }
    console.log('✓ Cleanup complete\n');
  }
}

// Run test
testBasicFunctionality().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
