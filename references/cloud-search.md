# Task-based skill discovery

The `search` command calls the AgentForge discovery service at
https://agentforge-five-sandy.vercel.app/api/v1/search. It sends the explicit
query, optional harness name, and result limit. No API key is needed. It does
not read or upload a transcript or repository, install a package, or execute
returned content. A configured endpoint override is an explicit operator choice.

The public service forwards to a Pixeltable Cloud catalog with keyword and
embedding retrieval. It returns source-pinned metadata. `lexical_only` means
semantic retrieval was unavailable. Empty results are a supported outcome.
Compatibility can be unknown, and indexing is not a security review.

The embedding model runs on the backend. The application does not intentionally
store query bodies. Hosting providers process technical request information.
Users should omit secrets from queries. An installed skill can be selected
implicitly, but automatic invocation depends on the harness and is not guaranteed
on every turn.

Review requirements and upstream licensing. To adopt, fetch the exact repository
revision and skill directory returned by search, inspect all package contents,
and use the existing fingerprint-reviewed promotion workflow. No match or service
failure should prevent the agent from continuing the original task.

`discover search` retains its existing GitHub repository search behavior. A Cloud
search failure never silently falls back to another provider.
