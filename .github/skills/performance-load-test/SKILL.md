---
name: performance-load-test
description: Guide HTTP load testing and performance analysis for adaptive concurrency limits. Use when creating k6 scenarios or interpreting throughput, latency, concurrency, queueing, rejection, saturation, and recovery.
---

# Performance Load Test

Use this skill for load-test scenarios and performance reports in this repository.

## Test objectives

Separate these hypotheses:

- fixed concurrency protects the service;
- adaptive concurrency responds to latency and queue formation;
- the gateway sheds excess load before it reaches the upstream;
- capacity or a dependency becomes a bottleneck;
- the system recovers after load decreases.

Do not combine every hypothesis into one scenario without recording the transitions.

## Minimum scenario stages

For each implementation, run:

1. **Baseline**: low, stable load with normal latency.
2. **Ramp-up**: increase RPS gradually.
3. **Saturation**: exceed accepted capacity.
4. **Overload**: maintain load above capacity.
5. **Recovery**: reduce load and observe whether latency and rejection rates return to baseline.

Keep payload and endpoint behavior stable between fixed and adaptive runs.

## Metrics to collect

Collect both application and HTTP behavior:

- throughput;
- requests received;
- requests accepted;
- requests rejected;
- active concurrency;
- current concurrency limit;
- p50, p95, and p99 latency;
- HTTP status distributions;
- upstream latency;
- CPU and memory;
- queue or timeout counts;
- retries, if any.

Use consistent units and clock sources.

## k6 guidance

- Define scenarios explicitly in `options.scenarios`.
- Use `ramping-arrival-rate` for RPS-focused experiments.
- Set representative thresholds, but explain that thresholds should not replace visual or statistical analysis.
- Warm up the service before collecting the measured interval.
- Avoid retries that transform load into a retry storm.
- Make rejection responses observable with a clear HTTP status.

## Comparison method

For fixed and adaptive limiters:

- use the same duration;
- use the same arrival pattern;
- use the same upstream behavior;
- use the same test machine and dependency version;
- record configuration and commit SHA;
- run repeated trials;
- report distributions rather than only averages.

Do not claim that a result generalizes beyond the tested topology.

## Interpretation

Use Little's Law as a diagnostic relationship:

```text
concurrency = throughput * latency
```

Interpret it as an average in steady state.

Watch for these transitions:

- throughput rises while latency remains stable;
- throughput reaches a plateau;
- latency and active concurrency rise together;
- rejection increases while useful throughput stays stable;
- useful throughput falls due to queues, retries, or timeouts;
- latency remains high after load decreases.

A lower request count with lower latency is not necessarily failure. It may represent successful load shedding.

## Acceptance criteria

A useful load-test report should answer:

- did the limiter prevent latency from growing uncontrollably;
- did it shed excess load before upstream saturation;
- did useful throughput stabilize;
- did it recover after overload;
- did it introduce unacceptable latency or rejection;
- how did fixed and adaptive limits differ.

Do not report a pass only because the process stayed running.
