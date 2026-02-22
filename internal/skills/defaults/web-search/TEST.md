# End-to-End Test Guide

## Testing the Complete Integration

This guide walks through testing the entire Web Search Skill integration with LobsterAI.

## Prerequisites

1. LobsterAI built and ready to run
2. Google Chrome installed
3. Internet connection available

## Test 1: Service Auto-Start

**Objective:** Verify that the Bridge Server starts automatically with LobsterAI.

**Steps:**

1. Start LobsterAI in development mode:
   ```bash
   npm run electron:dev
   ```

2. Check the console output for:
   ```
   [SkillServices] Starting skill services...
   [SkillServices] Starting Web Search Bridge Server...
   [SkillServices] Web Search Bridge Server started (PID: XXXXX)
   ```

3. Verify the server is running:
   ```bash
   curl http://127.0.0.1:8923/api/health
   ```

   Expected response:
   ```json
   {
     "success": true,
     "data": {
       "status": "healthy",
       "uptime": 123.45,
       "connections": 0
     }
   }
   ```

**Expected Result:** ✅ Bridge Server starts automatically within 3 seconds of LobsterAI launch.

## Test 2: CLI Search from Terminal

**Objective:** Test the search functionality directly from command line.

**Steps:**

1. Open a terminal while LobsterAI is running

2. Execute a search:
   ```bash
   bash SKILLs/web-search/scripts/search.sh "React 19 features" 5
   ```

3. Verify output contains:
   - Search query
   - Result count
   - Duration in ms
   - Markdown-formatted results with titles, URLs, and snippets

**Expected Result:** ✅ Search completes in < 3 seconds, returns 5 results.

## Test 3: Cowork Session Integration

**Objective:** Test Claude's ability to use the skill in a Cowork session.

**Steps:**

1. Start LobsterAI
2. Create a new Cowork session
3. Send the following message:

   ```
   Search for the latest information about Next.js 14 new features.
   ```

4. Observe:
   - Claude should recognize the need for real-time information
   - Claude should execute: `bash SKILLs/web-search/scripts/search.sh "Next.js 14 new features" 5`
   - Search results should appear in the tool execution output
   - Claude should synthesize information from the results
   - Claude should provide a summary with source citations

**Expected Result:** ✅ Claude uses web-search skill automatically, provides current information with sources.

## Test 4: Multiple Consecutive Searches

**Objective:** Verify connection caching and performance optimization.

**Steps:**

1. In a Cowork session, ask:
   ```
   1. Search for "TypeScript 5.0 features"
   2. Search for "React Server Components guide"
   3. Search for "Vite 5.0 changes"
   ```

2. Observe:
   - First search: ~2-3 seconds (includes browser launch)
   - Second search: ~1 second (reuses connection)
   - Third search: ~1 second (reuses connection)

**Expected Result:** ✅ Subsequent searches are faster due to connection caching.

## Test 5: Service Cleanup on Exit

**Objective:** Verify graceful shutdown of services when LobsterAI quits.

**Steps:**

1. With LobsterAI running and searches completed, quit the application
2. Check console output for:
   ```
   [SkillServices] Stopping skill services...
   [SkillServices] Stopping Web Search Bridge Server...
   [SkillServices] Web Search Bridge Server stopped
   ```

3. Verify server is stopped:
   ```bash
   curl http://127.0.0.1:8923/api/health
   ```

   Expected: Connection refused

4. Check no orphaned processes:
   ```bash
   ps aux | grep "web-search"
   ```

**Expected Result:** ✅ All services stop cleanly, no orphaned processes.

## Test 6: Error Handling - Server Not Running

**Objective:** Test behavior when Bridge Server is manually stopped.

**Steps:**

1. Start LobsterAI
2. Manually stop the Bridge Server:
   ```bash
   bash SKILLs/web-search/scripts/stop-server.sh
   ```

3. In Cowork session, ask Claude to search
4. Observe error message:
   ```
   ✗ Bridge Server is not running
     Please start the server first:
     bash SKILLs/web-search/scripts/start-server.sh
   ```

5. Manually restart:
   ```bash
   bash SKILLs/web-search/scripts/start-server.sh
   ```

6. Retry search

**Expected Result:** ✅ Clear error message, easy recovery path.

## Test 7: Browser Visibility

**Objective:** Verify all browser operations are visible and transparent.

**Steps:**

1. Start LobsterAI (ensure headless is false in config)
2. Execute a search via CLI or Cowork
3. Observe:
   - Chrome window appears
   - Navigates to Bing search page
   - Search query visible in URL bar
   - Results page loads visibly

**Expected Result:** ✅ All browser operations visible to user, transparent behavior.

## Test 8: Cross-Platform Compatibility

**Objective:** Verify skill works across different platforms.

**Platform-Specific Steps:**

### macOS
```bash
# Verify Chrome path detection
bash SKILLs/web-search/scripts/search.sh "test" 1

# Should find Chrome at: /Applications/Google Chrome.app/Contents/MacOS/Google Chrome
```

### Linux
```bash
# Verify Chrome/Chromium detection
bash SKILLs/web-search/scripts/search.sh "test" 1

# Should find at: /usr/bin/google-chrome or /usr/bin/chromium
```

### Windows
```bash
# Verify Chrome detection
bash SKILLs/web-search/scripts/search.sh "test" 1

# Should find at: C:\Program Files\Google\Chrome\Application\chrome.exe
```

**Expected Result:** ✅ Chrome detection works on all platforms.

## Test 9: Concurrent Searches

**Objective:** Test multiple searches happening in parallel.

**Steps:**

1. Open two terminal windows
2. Execute searches simultaneously:
   - Terminal 1: `bash scripts/search.sh "React" 3`
   - Terminal 2: `bash scripts/search.sh "Vue" 3`

3. Both should complete successfully

**Expected Result:** ✅ Multiple searches can run concurrently.

## Test 10: Result Quality

**Objective:** Verify search results are relevant and well-formatted.

**Steps:**

1. Search for a specific topic:
   ```bash
   bash SKILLs/web-search/scripts/search.sh "Playwright documentation" 5
   ```

2. Verify results include:
   - Official Playwright documentation (playwright.dev)
   - Recent tutorials and guides
   - Relevant Stack Overflow or GitHub discussions

3. Check Markdown formatting:
   - Headers for each result
   - Clickable URLs
   - Clean snippets

**Expected Result:** ✅ High-quality, relevant results with proper formatting.

## Performance Benchmarks

| Operation | Target | Acceptable |
|-----------|--------|------------|
| Server startup | < 2s | < 3s |
| Browser launch | < 3s | < 5s |
| First search | < 3s | < 5s |
| Subsequent search | < 1s | < 2s |
| Server shutdown | < 2s | < 3s |

## Common Issues and Solutions

### Issue 1: Server Won't Start

**Symptoms:** No PID file created, health check fails

**Debug:**
```bash
cat SKILLs/web-search/.server.log
npm run build --prefix SKILLs/web-search
```

### Issue 2: Chrome Not Found

**Symptoms:** "Chrome not found" error

**Solution:**
- macOS: Install from https://www.google.com/chrome/
- Linux: `sudo apt install chromium-browser`
- Windows: Install Chrome

### Issue 3: Port Already in Use

**Symptoms:** "Address already in use" error

**Solution:**
```bash
lsof -i :8923
kill -9 <PID>
bash SKILLs/web-search/scripts/start-server.sh
```

### Issue 4: Stale Connection

**Symptoms:** "Connection not found" error

**Solution:**
```bash
rm SKILLs/web-search/.connection
```

## Success Criteria

All tests pass when:

- ✅ Server auto-starts with LobsterAI
- ✅ Searches complete in < 3 seconds
- ✅ Claude uses skill automatically when appropriate
- ✅ Connection caching improves performance
- ✅ Services cleanup gracefully on exit
- ✅ Error messages are clear and actionable
- ✅ Browser operations are visible
- ✅ Works on macOS, Linux, Windows
- ✅ Concurrent searches supported
- ✅ Results are relevant and well-formatted

## Final Checklist

Before considering the integration complete:

- [ ] All 10 tests pass
- [ ] Performance benchmarks met
- [ ] No console errors or warnings
- [ ] Documentation is complete
- [ ] Code is compiled without errors
- [ ] Skills config includes web-search
- [ ] SKILL.md is comprehensive
- [ ] README.md is accurate
- [ ] Examples work as documented
- [ ] Service manager integrates cleanly

## Next Steps

After all tests pass:

1. Create commit with all changes
2. Test in production build
3. Document any platform-specific quirks
4. Gather user feedback
5. Consider Phase 2 enhancements (Google search, caching, etc.)
