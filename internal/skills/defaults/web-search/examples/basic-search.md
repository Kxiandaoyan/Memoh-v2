# Web Search Skill - Basic Usage Examples

This document provides practical examples of using the Web Search skill.

## Quick Start

### 1. Start the Bridge Server

```bash
bash SKILLs/web-search/scripts/start-server.sh
```

Expected output:
```
✓ Bridge Server started successfully (PID: 12345)
  Health check: http://127.0.0.1:8923/api/health
  Logs: SKILLs/web-search/.server.log
```

### 2. Perform a Simple Search

```bash
bash SKILLs/web-search/scripts/search.sh "TypeScript tutorial" 5
```

Expected output:
```
🔍 Searching for: "TypeScript tutorial"

✓ Found 5 results in 834ms

# Search Results: TypeScript tutorial

**Query:** TypeScript tutorial
**Results:** 5
**Time:** 834ms

---

## TypeScript Tutorial - W3Schools
...
```

### 3. Stop the Server

```bash
bash SKILLs/web-search/scripts/stop-server.sh
```

## Common Use Cases

### Example 1: Research Latest Information

**Scenario:** Find the latest React 19 features

```bash
bash SKILLs/web-search/scripts/search.sh "React 19 new features" 10
```

**Use Case:** When you need up-to-date information beyond Claude's knowledge cutoff.

### Example 2: Technical Documentation

**Scenario:** Search for Next.js App Router documentation

```bash
bash SKILLs/web-search/scripts/search.sh "Next.js App Router documentation" 5
```

**Use Case:** Find official documentation for specific frameworks or libraries.

### Example 3: News and Current Events

**Scenario:** Find recent AI news

```bash
bash SKILLs/web-search/scripts/search.sh "AI news 2026" 10
```

**Use Case:** Get real-time information about current events.

### Example 4: Troubleshooting Errors

**Scenario:** Search for error solutions

```bash
bash SKILLs/web-search/scripts/search.sh "TypeError: Cannot read property of undefined" 5
```

**Use Case:** Find solutions to specific error messages.

### Example 5: Comparative Research

**Scenario:** Compare technologies

```bash
bash SKILLs/web-search/scripts/search.sh "Vue vs React 2026 comparison" 8
```

**Use Case:** Gather information for comparative analysis.

## Advanced API Usage

### Direct API Calls (for advanced users)

#### Health Check

```bash
curl http://127.0.0.1:8923/api/health
```

Response:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "uptime": 123.45,
    "connections": 1
  }
}
```

#### Launch Browser

```bash
curl -X POST http://127.0.0.1:8923/api/browser/launch \
  -H "Content-Type: application/json"
```

Response:
```json
{
  "success": true,
  "data": {
    "pid": 12345,
    "cdpPort": 9222,
    "startTime": 1707363600000
  }
}
```

#### Connect to Browser

```bash
curl -X POST http://127.0.0.1:8923/api/browser/connect \
  -H "Content-Type: application/json" \
  -d '{}'
```

Response:
```json
{
  "success": true,
  "data": {
    "connectionId": "e2421754-0091-450d-a54c-7bc58498bfec",
    "cdpPort": 9222
  }
}
```

#### Perform Search

```bash
curl -X POST http://127.0.0.1:8923/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "connectionId": "e2421754-0091-450d-a54c-7bc58498bfec",
    "query": "TypeScript tutorial",
    "maxResults": 5
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "query": "TypeScript tutorial",
    "results": [
      {
        "title": "TypeScript Tutorial - W3Schools",
        "url": "https://www.w3schools.com/typescript/",
        "snippet": "Learn TypeScript with examples...",
        "source": "bing",
        "position": 1
      }
    ],
    "totalResults": 5,
    "timestamp": 1707363600000,
    "duration": 834
  }
}
```

#### Take Screenshot

```bash
curl -X POST http://127.0.0.1:8923/api/page/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "connectionId": "e2421754-0091-450d-a54c-7bc58498bfec",
    "format": "png",
    "fullPage": false
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "screenshot": "iVBORw0KGgoAAAANSUhEUgAA...",
    "format": "png",
    "size": 387122
  }
}
```

#### Navigate to URL

```bash
curl -X POST http://127.0.0.1:8923/api/page/navigate \
  -H "Content-Type: application/json" \
  -d '{
    "connectionId": "e2421754-0091-450d-a54c-7bc58498bfec",
    "url": "https://example.com",
    "waitUntil": "domcontentloaded",
    "timeout": 15000
  }'
```

#### Get Page Text

```bash
curl -X POST http://127.0.0.1:8923/api/page/text \
  -H "Content-Type: application/json" \
  -d '{
    "connectionId": "e2421754-0091-450d-a54c-7bc58498bfec"
  }'
```

#### Disconnect

```bash
curl -X POST http://127.0.0.1:8923/api/browser/disconnect \
  -H "Content-Type: application/json" \
  -d '{
    "connectionId": "e2421754-0091-450d-a54c-7bc58498bfec"
  }'
```

## Workflow Example: Complete Research Session

```bash
# 1. Start server
bash SKILLs/web-search/scripts/start-server.sh

# 2. Search for topic
bash SKILLs/web-search/scripts/search.sh "React Server Components" 5

# 3. Search for related topic
bash SKILLs/web-search/scripts/search.sh "Next.js 14 features" 5

# 4. Search for comparisons
bash SKILLs/web-search/scripts/search.sh "RSC vs traditional React" 3

# 5. Stop server when done
bash SKILLs/web-search/scripts/stop-server.sh
```

## Integration with Cowork Sessions

When using this skill in Cowork sessions, Claude will automatically:

1. Check if the Bridge Server is running
2. Start the server if needed (via Electron service manager)
3. Execute searches using the simplified CLI
4. Parse and analyze results
5. Provide answers based on real-time information

Example Cowork interaction:

```
User: What are the new features in React 19?

Claude: Let me search for the latest information about React 19.
        [Calls: bash SKILLs/web-search/scripts/search.sh "React 19 new features" 5]

        Based on the search results, React 19 introduces several key features:
        1. React Compiler - automatic optimization
        2. Actions - simplified form handling
        3. Document metadata - built-in SEO support
        ...
```

## Troubleshooting

### Server Won't Start

**Problem:** Bridge Server fails to start

**Solution:**
```bash
# Check if port 8923 is already in use
lsof -i :8923

# Check logs
cat SKILLs/web-search/.server.log

# Reinstall dependencies
cd SKILLs/web-search
npm install
npm run build
```

### Chrome Not Found

**Problem:** Browser fails to launch

**Solution:**
- Install Google Chrome or Chromium
- macOS: Download from https://www.google.com/chrome/
- Linux: `sudo apt install chromium-browser`
- Windows: Download from https://www.google.com/chrome/

### Connection Failed

**Problem:** Failed to connect to browser

**Solution:**
```bash
# Stop the server
bash SKILLs/web-search/scripts/stop-server.sh

# Clear cache
rm SKILLs/web-search/.connection
rm SKILLs/web-search/.server.pid

# Restart
bash SKILLs/web-search/scripts/start-server.sh
```

### Search Timeout

**Problem:** Search takes too long or times out

**Solution:**
- Check your internet connection
- Try a different search query
- Reduce max results (e.g., 3 instead of 10)
- Restart the browser

## Best Practices

1. **Start server once** - Keep server running during research sessions
2. **Use specific queries** - Better results with focused search terms
3. **Limit results** - Request only what you need (5-10 results)
4. **Clean up** - Stop server when done to free resources
5. **Check logs** - Review `.server.log` if issues occur

## Performance Tips

- **Connection caching** - Reuses browser connections for faster searches
- **Background server** - Server runs independently, no startup delay
- **Concurrent searches** - Multiple searches can run simultaneously
- **Resource cleanup** - Automatic cleanup on shutdown

## Security Notes

- Server only listens on `127.0.0.1` (localhost)
- No external network exposure
- Isolated browser profile (separate from your main Chrome)
- All operations visible in the browser window
- No credential storage or sensitive operations
