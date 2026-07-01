# Capability Router

The capability router lives in `internal/capabilities/router.go`.

## Responsibilities

- Register capabilities from plugin manifests.
- Map capability name to owning plugin and handler.
- Route calls to the correct handler.
- Enforce that only the owning plugin can invoke its capabilities.
- Prevent duplicate capability registration.

## Design

- One capability name can only be registered once.
- Handler is provided at registration time.
- Router does not know how the handler is implemented (local or remote via JSON-RPC).
- Permission check is simple: caller plugin ID must match the registered owner.

## Future integration

When the plugin supervisor and JSON-RPC client are ready, the router will be wired to forward capability calls to the correct plugin process.

Current implementation is in-memory and synchronous for testing purposes.
