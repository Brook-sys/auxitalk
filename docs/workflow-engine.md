# Workflow Engine

The workflow engine is the first core primitive for event-driven automation beyond chat assistance.

A workflow rule maps an incoming event to an action request:

```json
{
  "id": "auto-reply",
  "enabled": true,
  "trigger": {
    "eventType": "message.received",
    "source": "whatsapp"
  },
  "action": {
    "type": "message.reply.suggest",
    "risk": "medium",
    "payload": {
      "agent": "support"
    }
  }
}
```

Initial scope:

- exact event type matching, or `*` wildcard;
- optional source matching;
- one action request per matching rule;
- action requests keep the original session id and event metadata;
- actions still flow through the existing action gate/store when integrated by runtime code.

## Mock executor

The first executor is intentionally safe and dry-run only. It records simulated executions for these workflow action types:

- `send_message`
- `run_command`
- `call_plugin`
- `emit_event`

The mock executor never sends messages, runs shell commands, calls plugins, or emits real events. It returns an `ActionExecution` with `dryRun: true` so workflow behavior can be tested before real executors are added behind policy gates.

Planned next steps:

- load workflow rules from config;
- subscribe the engine to the runtime event bus;
- route generated actions through the runtime action gate;
- add richer conditions over payload fields;
- support multiple actions per rule.
