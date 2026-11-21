---
name: go-concurrency-limits
description: Guide implementation and review of Go concurrency limits. Use when designing or changing admission control, concurrency counters, goroutines, channels, cancellation, synchronization, or race-safe limiter state.
---

# Go Concurrency Limits

Use this skill for the concurrency-limit implementation in this repository.

## Core objectives

- Protect the service with bounded concurrency.
- Keep accepted operations observable and releasable.
- Make cancellation and completion paths symmetric.
- Prefer clear synchronization to clever concurrency patterns.
- Do not implement worker pools or queues unless they solve a measured requirement.

## Limiter state

Treat these values as limiter state:

- `currentLimit`: maximum admitted operations.
- `currentInFlight`: admitted operations not yet finished.
- `minimumLatency`: observed baseline latency.
- `currentLatency`: recently observed latency.

Use the smallest concurrency primitive that correctly protects state.

- Use a mutex for a small, bounded limiter state shared by handlers.
- Use channels to transfer events or results when ownership is already event-based.
- Do not use channels only to replace a counter or mutex.
- Do not create a goroutine for every waiting request.

## Admission lifecycle

For every request:

1. Try to acquire a permit.
2. If no permit is available, reject immediately.
3. If admitted, increment `currentInFlight`.
4. Execute or forward the operation.
5. On success, timeout, error, or cancellation, measure the completion if the result is valid.
6. Decrement `currentInFlight`.
7. Update the limiter based on the configured policy.

Do not let a timeout, panic, or early return skip permit release.

## Adaptive algorithm

Use the distinction between accepted work and received traffic:

- received RPS is external arrival rate;
- `currentInFlight` is admitted work still active;
- RPS must not be used as the sole input to concurrency admission.

When implementing the gradient policy, represent the concept clearly:

```text
gradient = minimumLatency / currentLatency
newLimit = currentLimit * gradient + sqrt(currentLimit)
```

The square root is a chosen queue allowance, not a result from Little's Law.

## Go conventions

- Pass `context.Context` through HTTP handlers and upstream calls.
- Respect cancellation in dependent calls.
- Avoid goroutine leaks: every goroutine needs a bounded lifetime and a completion path.
- Use `defer` for guaranteed permit release when a permit has been acquired.
- Use `sync.RWMutex` only if reads dominate and benchmarks justify it; start with `sync.Mutex`.
- Prevent races on counters, limits, latency baselines, and configuration.

## Review checklist

Before implementation is complete, verify:

- rejected requests never increment `currentInFlight`;
- every admitted request releases exactly once;
- timeout and cancellation paths release the permit;
- the limiter remains thread-safe;
- latency updates cannot corrupt limiter state;
- fixed and adaptive limiters satisfy the same interface;
- tests cover success, rejection, timeout, cancellation, and concurrent access;
- `go test -race ./...` passes.

## Validation style

Prefer small table-driven tests and focused concurrency tests.

Do not validate concurrency behavior with timing-only assertions. Coordinate test execution with channels or synchronization primitives and use timeouts only as failures.
