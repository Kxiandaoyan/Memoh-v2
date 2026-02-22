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

### File operations
Expose read/write/list operations scoped to a safe directory.

### Database queries
Wrap SQL queries with parameterized inputs to prevent injection.

## Testing

```bash
# Run server directly
python server.py

# Test with MCP inspector
npx @modelcontextprotocol/inspector python server.py
```
