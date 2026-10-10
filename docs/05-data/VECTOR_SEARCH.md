---
document: vector-search
version: 1.0
status: active
authority: architecture
depends_on:
  - KNOWLEDGE
  - DATA_OWNERSHIP
  - SECURITY_ARCHITECTURE
---

# Vector Search

## Abstraction

Knowledge domain code calls a Search Service abstraction rather than a vendor-specific vector database directly.

## Stored metadata

A vector record includes:

- chunkId
- tenantId
- knowledgeBaseId
- documentId
- embeddingModel/version
- source metadata
- vector

## Query

Every query applies mandatory metadata filters before/alongside similarity retrieval:

- tenantId
- authorized knowledgeBaseIds

## Hybrid retrieval

The target architecture supports:

1. keyword retrieval
2. vector similarity
3. result fusion
4. optional reranking

The MVP may begin with one retrieval strategy while keeping the abstraction stable.

## Index lifecycle

Embedding model changes must not silently reinterpret old vectors. Index records identify embedding model/version.

Re-indexing creates a controlled lifecycle rather than mutating semantics invisibly.

## Vendor neutrality

The domain contract does not expose vendor-specific query types.
