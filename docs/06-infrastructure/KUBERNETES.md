---
document: kubernetes
version: 1.0
status: target
authority: architecture
depends_on:
  - SYSTEM_ARCHITECTURE
  - OBSERVABILITY
---

# Kubernetes

## Resources

Target workloads use:

- Deployment
- Service
- ConfigMap
- Secrets
- Ingress
- readiness probe
- liveness probe
- resource requests/limits
- HPA where scaling signals are meaningful
- PodDisruptionBudget where availability requirements justify it

## Readiness

Readiness indicates whether a pod can safely receive traffic. It must not merely mean the JVM process is alive.

## Liveness

Liveness detects unrecoverable process state. It must not restart pods because a temporary database outage occurred.

## Scaling

Execution workers scale based on workload metrics such as queue lag and processing utilization. API services may scale on CPU/latency signals.

## Deployment

Use rolling deployment by default.

Schema changes must support rolling compatibility.

## Secrets

Kubernetes Secrets are delivery mechanisms, not a replacement for a dedicated secret-management strategy in higher environments.
