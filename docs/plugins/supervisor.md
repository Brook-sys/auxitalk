# Plugin Supervisor

The plugin supervisor lives in `internal/plugins/supervisor.go`.

## Responsibilities

- Start a plugin process.
- Capture stdin, stdout, stderr.
- Run handshake on startup.
- Perform periodic health checks.
- Capture stderr logs.
- Detect process crashes.
- Apply restart backoff.
- Enforce max restart limit.

## Lifecycle

```txt
NewSupervisor(plugin, options)
  -> Start(ctx)
       -> spawn process
       -> handshake
       -> start health loop
  -> monitor crashes
  -> Stop()
       -> kill process
       -> cleanup
```

## Options

- `HealthInterval`: interval between health checks.
- `HealthTimeout`: timeout for each health call.
- `RestartBackoff`: delay before restart.
- `MaxRestarts`: maximum restart attempts.

## Current scope

This phase provides the basic process lifecycle and restart policy skeleton.

Full handshake, health RPC, and log streaming will be added when the JSON-RPC client is integrated with the supervisor.
