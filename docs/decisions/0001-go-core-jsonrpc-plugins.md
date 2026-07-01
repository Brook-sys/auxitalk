# ADR 0001: Go core with external JSON-RPC plugins

## Status

Accepted draft

## Context

AuxiTalk needs a lightweight, robust, modular runtime that can run on low-resource hardware and remain easy to understand for humans and AI agents.

The project also needs community plugins that can be written in languages other than Go.

## Decision

The core runtime will be written in Go.

Plugins will initially run as external processes communicating with the core using JSON-RPC over stdio.

## Consequences

Positive:

- Lower core resource usage.
- Simple deployment model.
- Plugin language independence.
- Plugin crashes are isolated from the core.
- JSON-RPC is easy to inspect and debug.
- AI agents can generate and modify plugins more easily.

Negative:

- Process supervision is required.
- Protocol validation is required.
- Performance is lower than in-process plugins.
- Permission boundaries must be carefully designed.

## Initial mitigations

- Per-call timeouts.
- Payload size limits.
- Heartbeat checks.
- Restart backoff.
- Manifest permissions.
- Runtime modes.
- Schema validation.
