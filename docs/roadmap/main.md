# Roadmap

For the detailed first execution checklist, see `docs/roadmap/initial-implementation-plan.md`.

## Phase 0 — Foundation planning

- Define core architecture.
- Define plugin model.
- Define JSON-RPC protocol draft.
- Define runtime modes.
- Define repository structure.
- Define initial domain types.

## Phase 1 — Core skeleton

- Create Go module.
- Add `auxitalkd` entrypoint.
- Add config loader.
- Add structured logging.
- Add internal event bus.
- Add plugin manifest parser.
- Add stdio JSON-RPC transport.
- Add plugin supervisor lifecycle.

## Phase 2 — Domain model

- Define event model.
- Define session model.
- Define message model.
- Define suggestion model.
- Define action request model.
- Add session manager.
- Add context builder interface.

## Phase 3 — Plugin MVPs

- `mock-input` plugin.
- `mock-ai` plugin.
- `console-output` plugin.
- `file-memory` plugin.

## Phase 4 — Runtime loop

- Connect mock input to event bus.
- Update sessions from events.
- Build context.
- Call AI capability.
- Emit suggestions.
- Display suggestions through output plugin.
- Record user feedback events.

## Phase 5 — Safety and robustness

- Plugin timeouts.
- Payload size limits.
- Heartbeat checks.
- Restart backoff.
- Permission checks.
- Action gate.
- Runtime modes: `dev`, `local`, `strict`.

## Phase 6 — First real integrations

- Browser/WhatsApp Web input prototype.
- Desktop notification or overlay prototype.
- Real LLM provider plugin.
- SQLite memory plugin.
