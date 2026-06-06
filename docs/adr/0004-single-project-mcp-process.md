# ADR 0004: Single-Project MCP Process

## Context

MCP tools need a note root, index path, and project metadata. Allowing every tool call to choose a project independently would complicate schemas, authorization boundaries, and client behavior. The CLI already resolves project context before starting the server.

## Decision

Each MCP server process serves one resolved project selected at startup. Tool schemas do not accept a `project` argument. All tool calls operate within the server's fixed project context.

## Consequences

- MCP tool contracts stay smaller and easier for clients to use.
- Read and write tools inherit the same resolution and safety rules as the CLI.
- Cross-project operations require separate server processes instead of in-band routing.
- Future MCP changes must preserve project scoping unless the core transport model is redesigned.
