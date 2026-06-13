package tools

import sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterAll registers all MCP tools on the given server.
func RegisterAll(server *sdkmcp.Server, deps Dependencies) {
	// Read-only
	RegisterListNotes(server, deps)
	RegisterListTags(server, deps)
	RegisterSearchNotes(server, deps)
	RegisterReadNote(server, deps)
	RegisterListBacklinks(server, deps)

	// Write (Destructive)
	RegisterCreateNote(server, deps)
	RegisterEditNote(server, deps)
	RegisterDeleteNote(server, deps)
}
