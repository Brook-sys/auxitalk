# Initial Implementation Plan

This plan defines the first organized implementation cycle for AuxiTalk.

## Guiding principles

- Build the core before real integrations.
- Keep every module small and documented.
- Prefer stable contracts over fast hacks.
- Validate architecture with mock plugins first.
- Keep the core independent from app-specific behavior.
- Update documentation in the same change as code.

## Phase 1 — Repository foundation

### Tasks

- Initialize Go module.
- Create base directory structure.
- Add initial `auxitalkd` command.
- Add initial `auxitalkctl` command placeholder.
- Add repository-level configuration example.
- Add basic development instructions.

### Expected result

The repository can build an empty runtime binary.

### Acceptance criteria

- `go test ./...` runs successfully.
- `go run ./cmd/auxitalkd` starts and exits cleanly or prints basic runtime info.
- Documentation explains the repository layout.

## Phase 2 — Core types and event model

### Tasks

- Define base event type.
- Define session type.
- Define message type.
- Define suggestion type.
- Define action request type.
- Define plugin manifest type.
- Define capability type.
- Add validation helpers.

### Expected result

The core has stable domain primitives that plugins and internal modules can share.

### Acceptance criteria

- Types live in predictable packages.
- Validation has unit tests.
- Docs explain each core type.

## Phase 3 — Configuration system

### Tasks

- Define runtime config schema.
- Support runtime modes: `dev`, `local`, `strict`.
- Support plugin declarations.
- Support limits: timeout, payload size, rate limit.
- Load config from file.
- Provide sane defaults.

### Expected result

The runtime can start with a config file or defaults.

### Acceptance criteria

- Invalid config returns useful errors.
- Example config is documented.
- Unit tests cover defaults and validation.

## Phase 4 — Event bus

### Tasks

- Implement internal publish/subscribe bus.
- Support event handlers.
- Support context cancellation.
- Add basic event validation.
- Add optional in-memory event history.

### Expected result

Internal modules can communicate through normalized events.

### Acceptance criteria

- Event bus has unit tests.
- Publishing does not block forever.
- Handler errors are observable.

## Phase 5 — Plugin manifest and registry

### Tasks

- Parse `plugin.json`.
- Validate plugin id, kind, entry, runtime, permissions, capabilities.
- Register plugin metadata.
- Detect duplicate plugin ids.
- Document manifest fields.

### Expected result

The core can discover and validate plugins before running them.

### Acceptance criteria

- Manifest parser has tests.
- Invalid manifests produce clear errors.
- Plugin authoring guide matches implementation.

## Phase 6 — JSON-RPC stdio transport

### Tasks

- Implement JSON-RPC request/response types.
- Implement line-delimited stdio transport.
- Enforce max payload size.
- Enforce call timeout.
- Route method calls.
- Separate stdout protocol from stderr logs.

### Expected result

The core can communicate with a child process plugin through JSON-RPC.

### Acceptance criteria

- Transport has unit tests.
- Malformed JSON does not crash runtime.
- Timeout behavior is tested.
- Protocol docs are updated.

## Phase 7 — Plugin supervisor

### Tasks

- Start plugin process.
- Stop plugin process.
- Run handshake.
- Run health checks.
- Capture stderr logs.
- Detect crashes.
- Add restart backoff.
- Disable repeated crash loops.

### Expected result

Plugin failures are isolated and managed by the core.

### Acceptance criteria

- Plugin crash does not crash the core.
- Health failure is detected.
- Restart policy is configurable.
- Logs identify the plugin id.

## Phase 8 — Capability router

### Tasks

- Register plugin capabilities.
- Route capability calls to provider plugins.
- Validate permissions before calls.
- Return structured errors.
- Support multiple providers for the same capability.

### Expected result

Internal modules can call capabilities without knowing which plugin implements them.

### Acceptance criteria

- Missing capability returns clear error.
- Permission denial is explicit.
- Routing behavior has tests.

## Phase 9 — Session manager and context builder

### Tasks

- Create/update sessions from message events.
- Track participants and message direction.
- Build compact conversation context.
- Add simple context reduction strategy.
- Emit session update events.

### Expected result

The runtime can maintain normalized conversation state.

### Acceptance criteria

- Message events update sessions correctly.
- Context builder output is deterministic.
- Tests cover multiple sessions.

## Phase 10 — Action Gate

### Tasks

- Define action risk levels: `low`, `medium`, `high`.
- Define action decisions: allow, confirm, deny.
- Implement behavior per runtime mode.
- Add confirmation placeholder interface.
- Log action decisions.

### Expected result

Sensitive actions have a controlled path without a complex policy engine.

### Acceptance criteria

- `dev`, `local`, and `strict` modes behave differently.
- High-risk actions are not silently executed in `local`.
- Tests cover risk/mode matrix.

## Phase 11 — Mock plugins

### Tasks

- Create `mock-input` plugin.
- Create `mock-ai` plugin.
- Create `console-output` plugin.
- Create `file-memory` plugin.
- Document each plugin.

### Expected result

The architecture can be tested without WhatsApp or real AI providers.

### Acceptance criteria

- Mock input emits a message event.
- Mock AI returns a suggestion.
- Console output displays suggestion.
- File memory records feedback/events.

## Phase 12 — First complete runtime loop

### Tasks

- Wire mock input to event bus.
- Update session from incoming message.
- Build context.
- Call AI capability.
- Emit suggestion event.
- Display suggestion through output plugin.
- Record feedback event.

### Expected result

AuxiTalk demonstrates the complete architecture with mock plugins.

### Acceptance criteria

- One command runs the full mock loop.
- Logs show each stage clearly.
- Documentation includes a walkthrough.
- Tests cover core orchestration path where practical.

## Documentation checklist for every implementation change

Every new module or behavior must update at least one of:

- `README.md`
- `docs/architecture/core.md`
- `docs/plugins/system.md`
- `docs/plugins/protocol-draft.md`
- `docs/plugins/authoring-guide.md`
- `docs/roadmap/main.md`
- `docs/decisions/*.md`

## Initial execution order

1. Repository foundation.
2. Core types.
3. Config.
4. Event bus.
5. Manifest registry.
6. JSON-RPC transport.
7. Plugin supervisor.
8. Capability router.
9. Sessions/context.
10. Action Gate.
11. Mock plugins.
12. Full mock loop.
