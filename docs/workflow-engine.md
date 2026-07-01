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

Planned next steps:

- load workflow rules from config;
- subscribe the engine to the runtime event bus;
- route generated actions through the runtime action gate;
- add richer conditions over payload fields;
- support multiple actions per rule.
