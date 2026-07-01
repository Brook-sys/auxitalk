# Protocol Draft

This document defines the initial AuxiTalk plugin protocol draft.

## Base

Protocol: JSON-RPC 2.0 over stdio.

Each message must be a single JSON object encoded as one line.

## Message limits

Initial defaults:

```txt
max_payload_size = 1 MiB
request_timeout = 10s
health_timeout = 2s
max_events_per_second = 50
```

## Handshake

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "plugin.handshake",
  "params": {
    "protocolVersion": "0.1",
    "coreVersion": "0.1.0"
  }
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "pluginId": "mock-input",
    "protocolVersion": "0.1",
    "capabilities": ["conversation.observe"]
  }
}
```

## Event emit

```json
{
  "jsonrpc": "2.0",
  "id": "evt-1",
  "method": "event.emit",
  "params": {
    "type": "message.received",
    "sessionId": "session-1",
    "payload": {
      "text": "hello"
    }
  }
}
```

## Action request

```json
{
  "jsonrpc": "2.0",
  "id": "act-1",
  "method": "action.request",
  "params": {
    "type": "message.send",
    "risk": "high",
    "sessionId": "session-1",
    "payload": {
      "text": "Hello!"
    }
  }
}
```
