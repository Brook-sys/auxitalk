# Plugin System

AuxiTalk plugins are external processes communicating with the Go core through JSON-RPC over stdio.

This keeps the core language-agnostic and allows plugins to be written in Go, TypeScript, Python, Rust, or any language that can read stdin and write stdout.

## Initial transport

```txt
core process
  stdin/stdout JSON-RPC
plugin process
```

## Benefits

- Language independent.
- Easy to debug.
- Friendly for community plugins.
- Friendly for AI-generated plugins.
- Plugin crashes do not crash the core.
- No binary ABI coupling.

## Risks and mitigations

| Risk | Core mitigation |
| --- | --- |
| Plugin hangs | call timeout, heartbeat, kill policy |
| Payload abuse | max payload size |
| Event spam | rate limit per plugin |
| Invalid data | schema validation |
| Process leak | supervised lifecycle |
| Excess permission | manifest permissions and runtime mode |
| Crash loops | restart backoff and disable threshold |

## Plugin kinds

```txt
input       observes conversations, screens, apps, audio, webhooks
output      displays, copies, sends, or notifies
ai          LLM, embeddings, classifiers, tone analysis
memory      local DB, vector DB, profile memory, conversation memory
ui          panel, overlay, tray, CLI, mobile companion
policy      response timing, safety rules, custom action gates
tool        calendar, CRM, translator, browser, search
profile     user identity, style, contacts, social context
```

## Manifest draft

```json
{
  "id": "mock-input",
  "name": "Mock Input",
  "version": "0.1.0",
  "runtime": "node",
  "entry": "index.js",
  "kind": "input",
  "permissions": [
    "event.emit",
    "message.read"
  ],
  "capabilities": [
    "conversation.observe"
  ]
}
```

## Initial JSON-RPC methods

Core to plugin:

```txt
plugin.handshake
plugin.start
plugin.stop
plugin.health
capability.call
```

Plugin to core:

```txt
event.emit
action.request
memory.query
memory.write
ai.complete
log.write
```
