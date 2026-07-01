# Core Architecture

AuxiTalk core is an orchestration runtime. It should not know about WhatsApp, Telegram, OCR, OpenAI, local models, overlays, or any specific integration.

## Runtime flow

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

## Core responsibilities

- Load configuration.
- Start, stop, monitor, and restart plugins.
- Validate plugin manifests.
- Maintain sessions and normalized conversation state.
- Route capability calls.
- Apply action gates for sensitive operations.
- Expose runtime events to plugins and clients.
- Store audit-friendly event history when enabled.

## Non-responsibilities

The core must not directly implement:

- WhatsApp-specific logic.
- Browser automation logic.
- AI provider-specific APIs.
- Long-term memory backend internals.
- Desktop overlay UI logic.
- App-specific message sending.

Those belong in plugins.

## Runtime modes

```txt
dev      permissive mode for fast experiments
local    safe local mode with confirmation for sensitive actions
strict   explicit permissions, stronger validation, audit-oriented behavior
```

## Internal modules

```txt
config          configuration loading and validation
events          internal pub/sub and event log
plugins         plugin registry, lifecycle, supervision
rpc             JSON-RPC stdio transport
sessions        normalized conversation/session state
context         context building and reduction
capabilities    capability registry and routing
actions         action requests, gates, confirmations
memory          memory interfaces and routing
ai              AI interfaces and routing
permissions     capability-based permission model
logging         structured runtime/plugin logs
```
