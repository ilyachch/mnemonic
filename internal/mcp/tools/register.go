package tools

import sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterReadOnly registers the non-destructive MCP tools on the given server.
func RegisterReadOnly(server *sdkmcp.Server, deps Dependencies, description string) {
	RegisterListNotes(server, deps)
	RegisterListTags(server, deps)
	RegisterSearchNotes(server, deps, description)
	RegisterReadNote(server, deps)
	RegisterListBacklinks(server, deps)
}

// RegisterWrite registers the destructive MCP tools on the given server.
func RegisterWrite(server *sdkmcp.Server, deps Dependencies, description string) {
	RegisterCreateNote(server, deps, description)
	RegisterEditNote(server, deps)
	RegisterDeleteNote(server, deps)
}

// RegisterAll registers MCP tools on the given server. The description
// parameter is an optional custom knowledge-base description that gets
// prepended to primary entry-point tools (search_notes, create_note) to help
// agents route queries to the correct memory instance.
func RegisterAll(server *sdkmcp.Server, deps Dependencies, description string, readOnly bool) {
	RegisterReadOnly(server, deps, description)
	if !readOnly {
		RegisterWrite(server, deps, description)
	}
}
