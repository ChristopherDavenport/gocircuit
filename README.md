# GoCircuit

This library aims to create a generic approach to circuit breaking such that implementations can be swapped out easily. This should make it so software can not worry about the mechanics of the internals of their circuit breaker where they are leveraging it.

## Implemenations

### GoBreaker

When researching this pattern this was the standard that was communicated to me. As such this is the only in-memory implementation referenced.

### Redis

#### Realtime

This implementation evaluates off a realtime trailing window with shared state in Redis. Each evaluation of state clears elements outsides the trailing window so they are not included in the determination calculation.

### Noop

This implementation does nothing.