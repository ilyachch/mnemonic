package tools

import sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterAll registers all MCP tools on the given server. The description
// parameter is an optional custom knowledge-base description that gets
// prepended to primary entry-point tools (search_notes, create_note) to help
// agents route queries to the correct memory instance.
func RegisterAll(server *sdkmcp.Server, deps Dependencies, description string) {
	// Read-only
	RegisterListNotes(server, deps)
	RegisterListTags(server, deps)
	RegisterSearchNotes(server, deps, description)
	RegisterReadNote(server, deps)
	RegisterListBacklinks(server, deps)

	// Write (Destructive)
	RegisterCreateNote(server, deps, description)
	RegisterEditNote(server, deps)
	RegisterDeleteNote(server, deps)
}
