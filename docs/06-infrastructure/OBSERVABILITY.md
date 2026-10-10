---
document: observability
version: 1.0
status: target
authority: architecture
depends_on:
  - SYSTEM_ARCHITECTURE
  - AGENT_RUNTIME
---

# Observability

## Stack

- Spring Actuator
- Micrometer
- Prometheus
- OpenTelemetry
- Grafana
- optionally Loki
- optionally Tempo

## Logs

Structured logs include, where safe:

- timestamp
- service
- level
- correlationId
- traceId
- spanId
- tenant identifier in controlled form
- executionId
- operation
- outcome

Never log secrets, raw access tokens, or uncontrolled prompt/tool content.

## Metrics

Platform metrics include:

- HTTP request latency/error rate
- execution duration
- execution success/failure
- model latency
- tool latency
- token usage
- estimated model cost
- Kafka consumer lag
- document processing latency
- database connection health
- cache hit/miss where useful

## Traces

An execution trace should connect:

```text
HTTP request
  -> execution creation
  -> Kafka publish
  -> worker consume
  -> model call
  -> tool call
  -> retrieval
  -> final response
```

Trace context propagates through HTTP and Kafka.

## Alerting candidates

- sustained execution failure rate
- dependency outage
- Kafka lag
- high model latency
- tool failure spikes
- database saturation
- abnormal tenant error rates
