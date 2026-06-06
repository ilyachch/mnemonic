# HTTP Boundary

This package is reserved for a future HTTP adapter.

Do not add server implementation, routing, handlers, or transport-specific business logic here yet.

The core application is designed to remain transport-independent. CLI and MCP are the active adapters today; HTTP will be introduced later as a thin wrapper over the service layer.

Keep this directory free of executable HTTP code until the HTTP phase is explicitly scheduled.
