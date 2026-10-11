# ⚡ AgentForge

> **Production-grade, multi-tenant AI Agent Platform engineered for enterprise workflows, autonomous execution pipelines, and robust LLM orchestration.**

[![Java](https://img.shields.io/badge/Java-21-orange.svg)](https://openjdk.org/projects/jdk/21/)
[![Spring Boot](https://img.shields.io/badge/Spring%20Boot-4.1.1-brightgreen.svg)](https://spring.io/projects/spring-boot)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue.svg)](https://www.postgresql.org/)
[![Kafka](https://img.shields.io/badge/Apache%20Kafka-3.7-black.svg)](https://kafka.apache.org/)
[![Organization](https://img.shields.io/badge/GitHub-agentforge--hq-blueviolet.svg)](https://github.com/agentforge-hq)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

---

## 🎯 Overview

**AgentForge** is an enterprise AI Agent runtime and builder platform. It enables organizations to define, version, deploy, and monitor autonomous AI agents with hardened tenant isolation, transactional safety, and observable streaming execution.

### Key Capabilities

* 🏢 **True Multi-Tenancy:** Hardened tenant isolation enforced at the context holder, database queries, and messaging boundaries.
* 📜 **Immutable Agent Versions:** Published agent configurations are frozen and reproducible—never mutating in place.
* 📡 **Event-Driven Architecture:** Asynchronous event pipelines adhering to CloudEvents envelopes over Apache Kafka.
* 🛡️ **Zero-Trust Security & Tool Guardrails:** Server-side authorization, SSRF protection for HTTP tools, and defense against prompt injection.
* 🔍 **RAG & Knowledge Integration:** Modular search and vector indexing abstractions.
* ⚡ **Streaming Execution:** Real-time token and step telemetry delivered via Server-Sent Events (SSE).

---

## 🏗️ Architecture & Modules

The platform is organized into clean, decoupled bounded contexts:

```text
src/
├── agentforge-parent/          # Root BOM and build configuration
├── agentforge-common/          # Shared kernel: TenantContext, CloudEvents, errors & filters
├── agentforge-identity/        # Organizations, Users, Memberships, API Keys, and Auth
├── agentforge-agent/           # Agent definitions, versions, and prompt templates
├── agentforge-execution/       # Workflow execution runtime & state machines
└── agentforge-knowledge/       # Document ingestion and vector search pipelines
```

---

## 🚀 Quick Start (Local Development)

### Prerequisites
* **Java 21** or later
* **Maven 3.9+**
* **Docker & Docker Compose**

### 1. Start Infrastructure
```bash
docker compose up -d
```
Starts PostgreSQL, Redis, and Apache Kafka locally.

### 2. Build and Run Tests
```bash
mvn clean test
```

---

## 📚 Source of Truth & Documentation

AgentForge enforces strict architectural governance. The documentation hierarchy defines system invariants:

1. **Constitutional Invariants:** [`docs/00-constitution/PROJECT_CONSTITUTION.md`](docs/00-constitution/PROJECT_CONSTITUTION.md)
2. **Engineering Principles:** [`docs/00-constitution/ENGINEERING_PRINCIPLES.md`](docs/00-constitution/ENGINEERING_PRINCIPLES.md)
3. **Security Architecture:** [`docs/02-architecture/SECURITY_ARCHITECTURE.md`](docs/02-architecture/SECURITY_ARCHITECTURE.md)
4. **REST API Contracts:** [`docs/04-contracts/REST_API.md`](docs/04-contracts/REST_API.md)
5. **Event Contracts:** [`docs/04-contracts/EVENTS.md`](docs/04-contracts/EVENTS.md)
6. **Error Codes:** [`docs/04-contracts/ERROR_CODES.md`](docs/04-contracts/ERROR_CODES.md)
7. **Task Tracking Roadmap:** [`tasks/README.md`](tasks/README.md)

---

## 🌐 Ecosystem Roadmap (`agentforge-hq`)

* **`agentforge`** (Current): Core platform backend and orchestration engine.
* **`agentforge-studio`**: Visual Web UI for agent design, prompt tuning, and trace inspection.
* **`agentforge-python`**: Python SDK for data scientists and AI pipelines.
* **`agentforge-node`**: TypeScript / JavaScript client library.
* **`agentforge-helm`**: Kubernetes deployment charts and Terraform infrastructure.
