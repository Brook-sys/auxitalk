# Configuration

AuxiTalk configuration is loaded by `internal/config`.

The daemon accepts an optional config file:

```sh
go run ./cmd/auxitalkd --config configs/auxitalk.example.json
```

If no config path is provided, the runtime uses defaults.

## Runtime modes

```txt
dev      permissive mode for fast local experiments
local    safer local mode with confirmation for sensitive actions
strict   explicit permissions and stronger validation
```

## Default values

```txt
mode = dev
requestTimeout = 10s
healthTimeout = 2s
maxPayloadSize = 1048576
maxEventsPerSecond = 50
plugins = []
```

## Example

```json
{
  "mode": "dev",
  "runtime": {
    "requestTimeout": "10s",
    "healthTimeout": "2s",
    "maxPayloadSize": 1048576,
    "maxEventsPerSecond": 50
  },
  "plugins": []
}
```

## Plugin entries

A plugin can be declared by manifest path:

```json
{
  "manifest": "plugins/examples/mock-input/plugin.json",
  "enabled": true,
  "config": {}
}
```

Inline manifests are supported internally by the config model for tests and future tooling.
