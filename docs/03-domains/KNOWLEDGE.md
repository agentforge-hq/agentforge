---
document: knowledge-domain
version: 1.0
status: active
authority: domain
depends_on:
  - PROJECT_CONSTITUTION
  - IDENTITY
related:
  - VECTOR_SEARCH
  - EVENTS
---

# Knowledge Domain

## Entities

### KnowledgeBase

Tenant-owned logical collection.

### Document

Metadata and processing state for an uploaded source.

### DocumentChunk

Normalized chunk metadata and content reference.

### Embedding

Vector representation associated with a chunk and embedding model/version.

## Document lifecycle

```text
UPLOADED
  -> PROCESSING
  -> TEXT_EXTRACTED
  -> CHUNKED
  -> EMBEDDING
  -> INDEXED

Any processing state -> FAILED
```

Retries are bounded and state-aware.

## Query pipeline

```text
User Query
 -> query processing
 -> hybrid retrieval
 -> tenant + knowledge filtering
 -> reranking
 -> context construction
 -> model
 -> answer + citations
```

## Tenant isolation

Tenant filters are mandatory at every retrieval layer. A vector similarity score must never override tenant filtering.

## Citation

Retrieved chunks receive stable source identifiers so final answers can reference source documents/chunks.
