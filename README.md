# AuxiTalk Core

AuxiTalk Core is an event-driven automation runtime for connecting people, AI agents, tools, and communication channels through safe, observable workflows.

It observes events, keeps normalized state, builds compact context, routes plugin capabilities, coordinates workflows, and controls user-approved actions through a language-agnostic plugin system.

> Portuguese documentation: [README.pt-BR.md](README.pt-BR.md)

## What AuxiTalk is

AuxiTalk is the orchestration layer for an automation ecosystem that can connect chats, terminals, dashboards, APIs, plugins, and AI agents.

It is designed to support workflows such as:

- observing messages from channels such as WhatsApp, Telegram, Discord, browser, email, or another source;
- reacting to terminal output, command failures, logs, webhooks, file changes, or scheduled events;
- coordinating AI agents and tool plugins;
- suggesting, approving, or executing actions;
- showing state and pending approvals through a dashboard, CLI, chat, or another interface;
- executing sensitive actions only through explicit user-controlled gates;
- learning from feedback and future memory plugins.

AuxiTalk Core is not tied to a specific app, AI provider, database, or UI.

For the broader product direction, see [Product Vision](docs/product/vision.md).

## Design goals

- Lightweight Go runtime.
- Modular architecture.
- Language-agnostic plugins.
- JSON-RPC 2.0 over stdio as the initial plugin protocol.
- Clear contracts for events, sessions, messages, suggestions, actions, manifests, and capabilities.
- Safe defaults for sensitive actions.
- Documentation-first development.
- Easy to understand for humans and AI coding agents.

## Architecture

```txt
Input Plugin
  -> Event Bus
  -> Session Manager
  -> Context Builder
  -> Action Gate / Policy
  -> Capability Router
  -> AI / Memory / Tool Plugins
  -> Suggestion Event
  -> UI / Output Plugin
  -> User Feedback
  -> Memory Update
```

## Current modules

| Module | Path | Purpose |
| --- | --- | --- |
| Runtime | `internal/runtime` | Runtime lifecycle entrypoint |
| Config | `internal/config` | Config loading, defaults, runtime modes |
| Events | `internal/events` | Internal pub/sub event bus |
| Plugin registry | `internal/plugins` | Manifest loading, registry, supervisor skeleton |
| JSON-RPC codec | `internal/rpc` | Line-delimited stdio JSON-RPC codec |
| Capabilities | `internal/capabilities` | Capability registration and routing |
| Sessions | `internal/sessions` | Normalized conversation state |
| Context | `internal/context` | Compact context builder |
| Actions | `internal/actions` | Risk/mode-based action gate |
| Types | `pkg/types` | Public domain contracts |
| Protocol | `pkg/protocol` | Public JSON-RPC message types |

## Runtime modes

```txt
dev      permissive mode for fast local experiments
local    safer local mode with protection for sensitive actions
strict   restricted mode with stronger validation
```

## Plugin system

Plugins run as external processes and communicate with the core using JSON-RPC 2.0 over stdio.

This allows plugins to be written in Go, TypeScript/Node.js, Python, Rust, or any language that can read stdin and write stdout.

Important rules:

- `stdout` is reserved for JSON-RPC messages.
- `stderr` is reserved for human-readable logs.
- every JSON-RPC message must be one line.
- plugin manifests declare permissions and capabilities.
- sensitive actions must use the action request flow.

### Manifest example

```json
{
  "id": "mock-ai",
  "name": "Mock AI",
  "version": "0.1.0",
  "runtime": "node",
  "entry": "index.js",
  "kind": "ai",
  "permissions": [],
  "capabilities": [
    {
      "name": "ai.complete"
    }
  ]
}
```

## Repository structure

```txt
cmd/                 binaries: auxitalkd and auxitalkctl
configs/             example config files
docs/                architecture, roadmap, decisions, plugin docs
examples/            flow examples
internal/            private core packages
pkg/                 public types and protocol packages
plugins/             minimal example plugins
FINAL_STATUS.md      initial foundation report
```

## Getting started

### Requirements

- Go installed.

### Run the daemon

```sh
go run ./cmd/auxitalkd
```

### Run with config

```sh
go run ./cmd/auxitalkd --config configs/auxitalk.example.json
```

### Run the CLI placeholder

```sh
go run ./cmd/auxitalkctl
```

### Run tests

```sh
go test ./...
```

## Current status

The initial foundation is complete.

Implemented:

- configuration system;
- core domain types;
- internal event bus;
- plugin manifest registry;
- JSON-RPC protocol and stdio codec;
- plugin supervisor skeleton;
- capability router;
- session manager;
- context builder;
- action gate;
- mock plugin manifests;
- first full loop documentation.

## Next steps

Recommended next work:

1. wire the supervisor to real JSON-RPC request/response calls;
2. make the full mock loop executable from `auxitalkd`;
3. create official plugin repositories;
4. build the first AI provider plugin;
5. build the first memory plugin;
6. build the first UI/overlay plugin;
7. build the first real conversation input plugin.

## Related repositories

- `AuxiTalk/auxitalk` — core runtime.
- `AuxiTalk/plugin-template` — official plugin templates and authoring documentation.

## Documentation

Important docs:

- `docs/architecture/core.md`
- `docs/architecture/core-types.md`
- `docs/architecture/configuration.md`
- `docs/architecture/event-bus.md`
- `docs/architecture/capability-router.md`
- `docs/architecture/session-context.md`
- `docs/architecture/action-gate.md`
- `docs/plugins/authoring-guide.md`
- `docs/plugins/protocol-draft.md`
- `docs/plugins/system.md`
- `docs/roadmap/initial-implementation-plan.md`
- `docs/decisions/0001-go-core-jsonrpc-plugins.md`

## License

License to be defined.
