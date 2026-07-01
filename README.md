# AuxiTalk

AuxiTalk is a modular conversation-assistant runtime focused on observing conversations, building context, suggesting responses, and coordinating user-approved actions through extensible plugins.

## Goals

- Provide a robust core runtime written in Go.
- Support plugins written in multiple languages.
- Keep the core independent from specific apps, AI providers, memory backends, and UIs.
- Prefer explicit user control for sensitive actions.
- Stay lightweight enough for low-resource hardware.
- Keep architecture readable for humans and AI agents.

## Initial direction

The core starts as an orchestration runtime with JSON-RPC plugins over stdio.

Initial plugin categories:

- input plugins
- output plugins
- AI plugins
- memory plugins
- UI/control plugins
- policy/action plugins
- tool plugins

## Repository structure

```txt
cmd/                 executable entrypoints
internal/            private Go packages
pkg/                 public protocol/types packages
docs/                architecture, roadmap, plugin docs
plugins/             example and first-party plugins
configs/             example configuration files
```

## Development

Run the daemon:

```sh
go run ./cmd/auxitalkd
```

Run the CLI placeholder:

```sh
go run ./cmd/auxitalkctl
```

Run tests:

```sh
go test ./...
```
