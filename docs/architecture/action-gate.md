# Action Gate

The Action Gate lives in `internal/actions/gate.go`.

## Purpose

Control which actions are allowed, require confirmation, or are denied, based on the runtime mode and the risk level of the action.

## Risk levels

```txt
low      read-only or safe operations
medium   actions that may affect the user (copy, display)
high     actions that send messages or perform destructive operations
```

## Runtime modes

```txt
dev      everything is allowed
local    low=allowed, medium=confirm, high=denied
strict   low=confirm, medium/high=denied
```

This is the simplest form of policy. More complex rules can be added later as Policy Plugins.
