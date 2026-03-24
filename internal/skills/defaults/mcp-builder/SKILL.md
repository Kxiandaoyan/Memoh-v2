---
name: mcp-builder
description: Build Model Context Protocol (MCP) servers that expose tools, resources, and prompts to AI agents. Use when asked to create an MCP server, add new tools to an existing MCP, or integrate external APIs as MCP tools.
---

# MCP Builder

Create Model Context Protocol (MCP) servers to expose tools and resources to AI agents.

## MCP Concepts

- **Tools**: Functions the AI can call (e.g., `search_database`, `send_email`)
- **Resources**: Data the AI can read (e.g., files, DB records)
- **Prompts**: Reusable prompt templates

## Quick Start (Python)

```python
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp import types

server = Server("my-server")

@server.list_tools()
async def list_tools() -> list[types.Tool]:
    return [
        types.Tool(
            name="my_tool",
            description="What this tool does and when to use it",
            inputSchema={
                "type": "object",
                "properties": {
                    "param": {"type": "string", "description": "Parameter description"}
                },
                "required": ["param"]
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict) -> list[types.TextContent]:
    if name == "my_tool":
        result = do_something(arguments["param"])
        return [types.TextContent(type="text", text=str(result))]
    raise ValueError(f"Unknown tool: {name}")

async def main():
    async with stdio_server() as streams:
        await server.run(*streams, server.create_initialization_options())

if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
```

## Tool Design Guidelines

- **Name**: lowercase_underscore, descriptive verb_noun
- **Description**: Explain what it does AND when the AI should call it
- **Parameters**: Use JSON Schema; mark required fields; add descriptions
- **Returns**: Plain text or structured JSON as TextContent

## Common Patterns

### API wrapper
Wrap REST APIs as MCP tools. Handle auth, rate limits, and error responses gracefully.

```python
import httpx

async def call_api(endpoint: str, params: dict) -> str:
    async with httpx.AsyncClient(timeout=10) as client:
        try:
            resp = await client.get(f"https://api.example.com/{endpoint}", params=params)
            resp.raise_for_status()
            return resp.text
        except httpx.HTTPStatusError as e:
            return f"API error {e.response.status_code}: {e.response.text}"
        except httpx.RequestError as e:
            return f"Request failed: {e}"
```

### File operations
Expose read/write/list operations scoped to a safe directory.

### Database queries
Wrap SQL queries with parameterized inputs to prevent injection.

## Pre-Flight Validation Checklist

Before deploying an MCP server, verify:

- [ ] Every tool has a clear `description` explaining what it does and when to call it
- [ ] All required parameters are marked in `inputSchema.required`
- [ ] Error cases return informative `TextContent` messages (not raw exceptions)
- [ ] The server starts cleanly: `python server.py` exits without errors
- [ ] Tools respond correctly when tested via `npx @modelcontextprotocol/inspector`

## Testing

```bash
# Run server directly
python server.py

# Test with MCP inspector
npx @modelcontextprotocol/inspector python server.py
```
