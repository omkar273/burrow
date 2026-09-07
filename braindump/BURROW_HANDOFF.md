# BURROW - Product + Engineering Handoff

**Purpose:** This document is the source of truth for continuing the Burrow product discussion in a fresh ChatGPT, Codex, or engineering session without re-deriving the prior reasoning.

**Status:** Product thesis is defined. Naming direction is provisionally **Burrow**. Architecture is conceptually defined. V1 scope is defined. Product thesis PDF and pitch deck have been produced and visually refined. The next phase is to convert the thesis into an implementation-ready technical specification and then build the V1 recovery loop.

**Current date:** September 2026

---

# 1. ONE-PARAGRAPH PRODUCT DEFINITION

**Burrow is a local-first, cloud-optional SaaS data ownership and recovery layer that continuously copies critical business data into storage the customer controls, verifies that the copy remains recoverable, and provides a unified interface for search, restore, and portability.**

The product should make a sophisticated backup/recovery system feel simple:

> **Connect once. Stay protected. Know you can recover.**

Core positioning:

> **Your SaaS data. Your storage. Your copy.**

Alternative brand line:

> **SaaS data, independently kept.**

Core product promise:

> **Keep a copy of what matters.**

---

# 2. WHY BURROW EXISTS

Businesses increasingly depend on SaaS as their system of record:

- Google Workspace / Gmail
- Google Drive
- Microsoft 365
- Slack
- GitHub
- Notion
- CRMs
- other operational SaaS systems

But relying on the provider's native recovery is not equivalent to having an independent backup.

There are three gaps:

## Recoverability gap

Provider recovery is provider-specific and bounded. A customer can have a copy but still discover after an incident that the record they need is outside the available recovery window, difficult to reconstruct, or no longer accessible.

## Operations gap

DIY systems are flexible but expose users to:

- credentials
- checkpoints
- queues
- retries
- API quirks
- storage failures
- state management
- indexing
- reconciliation
- restore testing

The desired experience is "set and forget", not "operate a backup platform."

## Ownership gap

Customers may want managed convenience without giving up control of:

- storage
- encryption keys
- portability
- retention
- recovery destination

Burrow treats ownership and portability as first-class product properties.

---

# 3. THE PRODUCT THESIS

The important conceptual shift:

> **Burrow is not primarily a storage product. It is a recoverability product.**

A backup is an action.

Recovery is a property.

Burrow should therefore move through this trust loop:

```text
CAPTURE
   |
   v
VERIFY
   |
   v
DRILL
   |
   v
RECOVER
```

A useful operational definition is:

```text
Protected =
    fresh
    + intact
    + replicated
    + recoverable
```

The dashboard should ultimately be able to justify that state rather than merely display "last sync successful."

---

# 4. CORE MENTAL MODEL

The architecture revolves around five primitives:

```text
Source
Object
Storage
Runtime
Consumption
```

## Source

Knows how to read data from a provider and understand provider-native semantics.

Examples:

- Gmail
- Google Drive
- Microsoft 365
- Slack
- GitHub

## Object

The stable, canonical logical representation of data inside Burrow.

The logical object should not be tightly coupled to one storage backend.

An object may have:

- stable object ID
- source
- source-native ID
- timestamps
- type
- metadata
- content/blob reference
- version information
- integrity information
- lineage
- replica references

The object is the logical unit of truth.

## Storage

A physical persistence backend.

Examples:

- local filesystem
- S3-compatible object storage
- Cloudflare R2
- NAS
- SFTP
- other adapters where technically sensible

Storage is replaceable.

Do not design the product around one cloud storage provider.

## Runtime

Where Burrow's engine executes.

Two modes:

### Local runtime

A lightweight agent/daemon running on the customer's machine or server.

Expected building blocks:

- local state
- SQLite
- local queue/job execution
- customer-controlled storage
- optional local web UI

### Cloud runtime

Managed execution provided by Burrow.

Expected building blocks:

- workers
- queue
- durable state
- monitoring
- customer-selected or managed storage

Important architectural rule:

> **Runtime placement must not change the logical data model.**

## Consumption

How users interact with the archive.

Examples:

- search
- browse
- preview
- versions
- restore
- export
- health
- recovery history
- audit

---

# 5. HIGH-LEVEL ARCHITECTURE

Target flow:

```text
                 SaaS Providers
          ┌──────────┬──────────┐
          │ Gmail    │ Drive    │ M365 ...
          └────┬─────┴────┬─────┘
               │          │
               v          v
           Source adapters
                 |
                 v
          Ingestion / Sync
                 |
                 v
       Canonical Logical Objects
                 |
        ┌────────┴─────────┐
        │                  │
        v                  v
    Replica A           Replica B
        │                  │
        v                  v
 Local / S3 / R2      NAS / S3 / ...
        \                  /
         \                /
          └──────┬───────┘
                 v
     Search / Restore / Export
```

Another view:

```text
                  +----------------+
                  |    SOURCES     |
                  | Gmail / Drive  |
                  | M365 / Slack   |
                  +-------+--------+
                          |
                          v
                  +----------------+
                  |    INGEST      |
                  | sync/reconcile |
                  +-------+--------+
                          |
                          v
                  +----------------+
                  |     BURROW     |
                  | logical objects|
                  | state + jobs   |
                  +---+--------+---+
                      |        |
              +-------+        +-------+
              v                        v
       +-------------+          +-------------+
       |   Replica   |          |   Replica   |
       | local / R2  |          |   S3 / NAS  |
       +-------------+          +-------------+
               \                  /
                \                /
                 v              v
              +--------------------+
              | Search / Restore   |
              | Export / Health    |
              +--------------------+
```

---

# 6. LOCAL-FIRST + CLOUD-OPTIONAL STRATEGY

The product should support both modes without maintaining two conceptual products.

## Local mode

Target experience:

```text
Install agent
    ↓
Connect Gmail
    ↓
Choose local folder / S3 / NAS
    ↓
Sync
    ↓
Protected
```

Benefits:

- inexpensive
- customer-controlled
- works with local storage
- strong portability
- can be open/self-hostable
- does not require Burrow cloud to exist

Trade-off:

- execution pauses if the machine/server is unavailable
- customer manages the runtime

## Cloud mode

Target experience:

```text
Connect SaaS
    ↓
Choose storage
    ↓
Burrow runs continuously
    ↓
Protected
```

Benefits:

- always-on
- no local runtime maintenance
- central monitoring
- team access
- easier managed experience

Trade-off:

- infrastructure cost
- additional trust relationship with Burrow

Important:

> **Local mode and cloud mode should share the same core engine/contracts.**

---

# 7. STORAGE STRATEGY

Storage is a pluggable subsystem.

Initial targets:

1. Local filesystem
2. S3-compatible storage
3. Cloudflare R2

Later:

- NAS
- SFTP
- additional object stores
- other unusual backends if useful

Potentially interesting but explicitly non-core:

- Telegram

Telegram was considered because it could be used as a cheap unusual storage backend for certain small payloads. It should NOT become the product or the canonical storage model.

Telegram has practical file-size/API constraints and is not appropriate as the sole general-purpose archive backend.

The abstraction should therefore be:

```ts
interface Storage {
  put(...)
  get(...)
  exists(...)
  delete(...)
  list(...)
  stat(...)
  verify(...)
}
```

Exact interface still needs to be designed.

---

# 8. DATA MODEL PRINCIPLE: ONE LOGICAL COPY, MANY PHYSICAL REPLICAS

The user should think:

> "This is my Burrow archive."

Not:

> "This folder on R2 contains Gmail exports."

Internally, something like:

```text
LogicalObject
    |
    +-- metadata
    +-- source lineage
    +-- versions
    +-- integrity
    +-- replica refs
```

Then:

```text
Replica
    |
    +-- storage backend
    +-- storage key
    +-- state
    +-- checksum
    +-- last verified
```

This makes it possible to:

- change storage
- add a second replica
- repair a replica
- verify copies independently
- migrate storage
- restore without coupling restore logic to a backend

---

# 9. STATE, SQLITE, AND PORTABILITY

SQLite is the preferred initial state store in local mode and a strong candidate for the core data model.

State includes:

- source configuration metadata
- checkpoints
- ingestion jobs
- retry state
- object metadata/indexes
- replica state
- health state
- verification history
- audit/history
- export/import metadata

Do not blindly export secrets.

Secrets should either:

- be re-authenticated during import
- or be encrypted using customer-controlled key material

---

# 10. PORTABLE ARCHIVE BUNDLE

Docker is important, but the central portability abstraction should NOT be Docker itself.

Docker packages the runtime.

The portable state should be an application-level archive bundle.

Conceptually:

```text
burrow-export/
├── manifest.json
├── schema-version
├── state.sqlite
├── objects/
├── indexes/
├── config/
├── checksums/
└── encryption-metadata/
```

The exact structure is not finalized.

The important invariant:

> **A Burrow export should allow another Burrow runtime to reconstruct the logical application state without relying on the original machine, Docker volume, or infrastructure provider.**

This is one of the strongest product properties.

---

# 11. BACKUP OF BURROW ITSELF

Burrow's own application state is data too.

Expected snapshot strategy:

- hourly snapshots
- daily snapshots
- weekly snapshots
- optional monthly retention

Initial retention hypothesis:

```text
Hourly: 24-48
Daily: 30 days
Weekly: 12 weeks
Monthly: 12 months
```

The exact policy is configurable later.

More important than the schedule:

> **Perform restore drills.**

A database snapshot that has never been restored is an assumption, not proof.

---

# 12. RECOVERY MODEL

Recovery should be treated as a first-class product engine.

Potential restore targets:

1. original source
2. alternate source/account
3. local download
4. complete archive export
5. future structured reconstruction

V1 should support at least:

- search
- preview
- restore
- export

Recovery needs explicit semantics for:

- overwrite
- restore as new
- version selection
- conflicts
- deleted source records
- metadata restoration
- attachments/content
- partial restore failures

These semantics are important design work for the next phase.

---

# 13. UX PRINCIPLES

The implementation should be complex underneath and calm on the surface.

The dashboard should answer three questions immediately:

### Am I protected?

Show:

- freshness
- integrity
- replica health
- recovery drill status

### Where is my copy?

Show:

- storage locations
- replica count
- last successful write
- last verification

### Can I recover?

Show:

- latest recovery drill
- restore capability
- export capability
- warnings

The user should not need to understand:

- queues
- checkpoint leases
- Pub/Sub renewal
- MIME parsing
- provider event semantics
- retry internals
- indexing internals

unless they explicitly open technical diagnostics.

---

# 14. UX FLOW

## First run

```text
Connect source
    ↓
Choose storage
    ↓
Choose replica(s)
    ↓
Initial sync
    ↓
Protected
```

## Everyday state

```text
Protected

Gmail
Last sync: 7 min ago
Replica 1: Healthy
Replica 2: Healthy
Last recovery drill: 2 days ago
```

## Incident state

```text
Find
  ↓
Inspect
  ↓
Choose version
  ↓
Restore
  ↓
Verify
```

---

# 15. INITIAL ICP

Initial target:

> **3-50 person businesses with important business operations living in Google Workspace or Microsoft 365, but without a dedicated backup administrator.**

Likely verticals:

- agencies
- accounting / CA firms
- law firms
- consultancies
- startups
- e-commerce businesses
- professional practices
- nonprofits

Potential buyer:

- founder
- owner
- COO
- operations lead
- technical lead
- office administrator
- MSP / IT partner

Important distinction:

The buyer is often not the engineer.

The product therefore needs technical depth with a non-technical primary experience.

---

# 16. BUYER JOB-TO-BE-DONE

The emotional job is:

> "I want to know that we won't lose something important because our SaaS provider had a problem or because someone deleted something six months ago."

The functional job:

> "Continuously maintain an independent copy of critical SaaS data somewhere we control and make recovery straightforward."

The operational job:

> "Do not make me maintain a backup system."

---

# 17. POSITIONING

Current strongest positioning:

> **Burrow**
>
> **Your SaaS data. Your storage. Your copy.**

Alternative:

> **Burrow**
>
> **SaaS data, independently kept.**

Underlying promise:

> **Keep a copy of what matters.**

Another possible line:

> **An independent home for your SaaS data.**

The metaphor should remain subtle.

Do NOT over-index on rabbits, burrows, carrots, underground jokes, etc.

Burrow should feel:

- calm
- private
- trustworthy
- technical
- slightly warm
- not enterprise-stiff
- not childish

---

# 18. WHY THE NAME BURROW WAS CHOSEN

The word has strong semantic fit:

- a protected place
- private
- durable
- independent
- somewhere important things can live
- can contain multiple things
- implies an owned home

It maps naturally to:

```text
SaaS → Burrow → customer-controlled storage
```

Useful product language:

- Create your Burrow
- Your Burrow is protected
- Add a storage to your Burrow
- Restore from Burrow
- Export your Burrow

But the metaphor should not dominate the interface.

## Naming risk

Burrow is not uncontested.

There are existing technical projects/products with the name, including:

- LinkedIn Kafka Burrow
- a Mac/Windows utility at burrow.computer
- other developer-oriented Burrow projects
- other consumer/software uses

Therefore:

> **Burrow is provisionally selected, not legally finalized.**

Before registration/launch, perform:

- trademark search
- domain search
- GitHub organization/name search
- package namespace search
- Google category collision search
- confusion analysis for SaaS backup/data protection

The decision is not "is Burrow unused?"

The decision is:

> **Can Burrow be owned in our category despite other unrelated uses?**

---

# 19. COMPETITIVE LANDSCAPE

Conceptual categories:

## DIY tools

Examples:

- restic
- Kopia
- rclone
- custom scripts
- GYB
- IMAP/email archive systems

Strength:

- ownership
- flexibility

Weakness:

- operations
- setup
- recovery confidence
- maintenance

Burrow angle:

> Package the reliable recovery workflow and UX.

## Enterprise SaaS backup

Examples researched:

- Afi
- Backupify
- Keepit
- SysCloud

Strength:

- mature recovery
- managed

Weakness:

- cost
- complexity
- less customer-controlled architecture

Burrow angle:

> Simpler, more portable, more ownership-oriented.

## Automation platforms

Example:

- n8n

Strength:

- breadth
- extensibility

Weakness:

- not recovery-first
- correctness is up to the workflow builder

Burrow angle:

> Purpose-built synchronization and recovery.

## Email archive projects

Example:

- Open Archiver

Strength:

- archive/search
- self-hosting
- email expertise

Weakness:

- archive != recovery platform

Burrow angle:

> Expand from archival into independent recoverability across SaaS.

---

# 20. WHAT IS NOT THE MOAT

Do not claim these as moats:

- Gmail connector
- S3 support
- Telegram support
- SQLite
- generic retries
- basic encryption

These are implementation features.

---

# 21. POTENTIAL MOATS

The strongest long-term defensibility candidates:

## 1. Connector correctness

Provider APIs have subtle failure modes.

Examples:

- incremental sync
- pagination
- deleted objects
- edits
- attachment edge cases
- permission changes
- history gaps
- webhook/watch expiry
- reconciliation

Over time, knowing how to correctly synchronize each provider can become meaningful operational IP.

## 2. Canonical data model

A durable logical archive model across providers is harder to reproduce than a single connector.

## 3. Recovery engine

The actual semantics of:

- restore
- versioning
- conflict handling
- partial recovery
- alternate restore
- reconstruction
- recovery drills

can become a major product differentiator.

## 4. Storage sovereignty

The more completely Burrow separates data ownership from Burrow infrastructure, the harder it is for competitors to offer equivalent portability without rebuilding the model.

## 5. Operational reliability history

Over time Burrow can accumulate knowledge around:

- provider behavior
- sync anomalies
- verification
- recovery success
- repair patterns

That can become real product advantage.

## 6. Consumption layer

Search/history/lineage/governance over the archive can make Burrow useful even when the user is not in a disaster.

---

# 22. BUSINESS MODEL HYPOTHESIS

The business should monetize:

- convenience
- always-on execution
- monitoring
- team access
- advanced recovery
- governance
- managed infrastructure

Do not build the business primarily around markup on commodity object storage.

Potential packaging:

## Free / self-hosted

- local runtime
- bring your own storage
- core restore
- export
- basic health

## Managed

- always-on runtime
- monitoring
- team access
- advanced recovery
- recovery history
- alerts
- operational convenience

## Expansion

- more sources
- retention
- compliance
- governance
- audit
- MSP capabilities

Pricing should likely be:

> workspace/source-based

rather than purely per-user.

Exact pricing is not finalized and should be validated through pilots.

---

# 23. V1 PRODUCT SCOPE

Do not build a huge connector platform first.

Build one complete recovery loop.

## V1 must have

### Source

- Google Workspace / Gmail
- attachments

### Sync

- incremental sync
- reconciliation
- retry
- checkpointing

### State

- SQLite
- durable jobs
- source state
- replica state
- health state

### Storage

- local filesystem
- S3-compatible storage

### Integrity

- checksums
- verification
- dedupe

### Reliability

- retries
- idempotency
- recovery from interrupted runs

### Application

- web UI
- search
- object inspection
- restore
- export
- health dashboard

### Portability

- archive bundle
- import
- export
- schema version

### Burrow state backup

- DB snapshot
- recovery procedure

---

# 24. V1 NON-GOALS

Do not let V1 expand into:

- 20 SaaS connectors
- full eDiscovery
- enterprise compliance platform
- SSO everywhere
- multi-tenant MSP platform
- AI search as the primary differentiator
- generic workflow automation
- consumer file storage
- full desktop GUI
- Kubernetes-first architecture
- distributed event infrastructure without need
- Telegram as the main product

---

# 25. INITIAL COMPONENT BOUNDARIES

A likely structure:

```text
core/
  object-model
  source-contract
  storage-contract
  state-store
  queue
  sync
  retry
  verification
  archive-bundle
  recovery

adapters/
  gmail
  local-fs
  s3

runtime/
  local
  cloud

apps/
  web
  api
  agent
```

This is conceptual, not a mandated repository layout.

The key is clean boundaries, not directory naming.

---

# 26. SOURCE CONTRACT

The exact API needs to be designed.

The source abstraction should support concepts such as:

- initial listing
- incremental changes
- object fetch
- content fetch
- metadata
- deletion
- checkpoint
- reconciliation
- provider health

Avoid making the abstraction so generic that it destroys provider-specific semantics.

The connector can expose normalized behavior while retaining provider-specific logic internally.

---

# 27. STORAGE CONTRACT

The storage abstraction should support at minimum:

```ts
put(object)
get(key)
exists(key)
delete(key)
list(prefix)
stat(key)
verify(key)
```

Likely additional concepts:

- multipart upload
- streaming
- content hash
- capabilities
- transactional metadata if available
- range reads
- atomic write semantics

Need a capability matrix rather than pretending every storage backend is identical.

---

# 28. INGESTION / JOB MODEL

The ingestion engine needs to be:

- idempotent
- restartable
- checkpointed
- observable
- retryable
- deterministic where possible

Core principle:

> A job can run twice without corrupting the logical archive.

Potential job types:

- source discovery
- incremental sync
- object fetch
- blob persist
- metadata persist
- replica copy
- verification
- reconciliation
- recovery
- snapshot
- export

The queue implementation should be intentionally simple in local V1.

---

# 29. GMAIL-SPECIFIC NOTES

The Gmail/Workspace connector has provider-specific operational complexity.

Google Gmail push notifications involve Pub/Sub and mailbox watch renewal.

A watch must be renewed periodically and operationally should be treated as ephemeral infrastructure rather than trusted long-term state.

Design implications:

- checkpoint state must be durable
- missing notification windows must be recoverable
- periodic reconciliation is still necessary
- event-driven ingestion must not be treated as the sole source of truth

V1 should have:

```text
event-driven when available
+
periodic reconciliation
```

rather than relying exclusively on push.

---

# 30. BACKUP VS RECONCILIATION

An important architectural point:

**Push events are triggers, not truth.**

The authoritative source remains the SaaS provider.

Therefore Burrow should periodically reconcile:

```text
last known state
        vs
provider state
```

This helps detect:

- missed events
- provider inconsistencies
- local state corruption
- deletions
- changes that occurred outside the expected event stream

---

# 31. EMAIL DATA MODEL

For Gmail V1, likely logical object families:

```text
Message
Attachment
Thread
Label / metadata
```

Need to define:

- stable internal ID
- provider ID
- thread ID
- sender
- recipients
- subject
- timestamps
- labels
- headers
- body
- attachment references
- content checksum
- version lineage

Avoid locking the canonical model too tightly to Gmail MIME structures.

---

# 32. SEARCH

V1 search should be useful without introducing a massive search platform.

Likely search fields:

- sender
- recipient
- subject
- date
- attachment name
- labels
- source
- object type

Full-text attachment extraction can be later if it materially improves the wedge.

Search should abstract physical storage.

The user should never have to say:

> "Search replica B in R2."

They search Burrow.

---

# 33. DUPLICATION + CONTENT ADDRESSING

Dedupe should be considered at the object/blob layer.

Candidate strategy:

```text
content hash
      ↓
blob identity
      ↓
multiple logical references
```

Need to carefully distinguish:

- logical object identity
- content identity
- version identity
- replica identity

Do not use one ID for all four concepts.

---

# 34. ENCRYPTION

Customer ownership is central.

Need to separate:

- encryption at rest
- object encryption
- key ownership
- credentials
- exported state
- recovery keys

Important product principle:

> A customer's portable archive should not become useless because its encryption key only exists inside Burrow.

The exact key-management model is not finalized.

Potential future directions:

- customer-managed encryption key
- locally-held master key
- envelope encryption
- re-authentication on import

---

# 35. DOCKER STRATEGY

Docker should package:

- runtime
- dependencies
- configuration defaults

Docker should NOT be the only way to preserve:

- data
- state
- object history
- identity
- configuration

In other words:

> **Containerize compute. Externalize / export state.**

A user should be able to replace:

```text
machine
container
runtime host
storage backend
```

without losing the logical archive.

---

# 36. SUCCESS METRICS

Primary:

> **Percentage of protected objects that can be successfully recovered when needed.**

Secondary:

- source freshness
- replica health
- verification coverage
- recovery-drill success
- mean time to recovery
- restore failure rate
- sync gap duration
- export/import success
- onboarding completion
- retention
- managed conversion

Avoid optimizing V1 around:

- total bytes stored
- number of connectors
- raw API calls
- jobs executed

Those can become vanity metrics.

---

# 37. THE MOST IMPORTANT DEMO

The strongest product demo is:

```text
CONNECT
   ↓
PROTECT
   ↓
DELETE
   ↓
RECOVER
```

Concrete version:

1. Connect a real Google Workspace.
2. Choose customer-controlled storage.
3. Sync a real invoice email.
4. Search it inside Burrow.
5. Delete it from Gmail.
6. Restore it from Burrow.
7. Verify the restored result.
8. Export the Burrow state.
9. Reconstruct it elsewhere.

The moment to optimize for:

> "I deleted it from SaaS and Burrow still had it."

That makes the thesis tangible.

---

# 38. IMPORTANT PRODUCT LANGUAGE

Prefer:

- independent copy
- recoverable
- customer-controlled
- protected
- replica
- archive
- restore
- portable
- another home

Avoid overusing:

- backup server
- infrastructure
- pipeline
- DAG
- workflow engine
- event processor
- object store

Technical language belongs in diagnostics/docs, not primary UX.

---

# 39. REJECTED OR DE-PRIORITIZED DIRECTIONS

## Telegram-first

Rejected as the product concept.

Telegram may be one possible backend adapter in narrow cases, but not the architecture.

## General automation platform

Do not become n8n.

Automation is a means to achieve reliable recovery, not the product.

## Storage company

Do not become "Dropbox for businesses".

The core value is recoverability and ownership, not commodity storage.

## Email-only product

Email is the wedge.

The product should be architected for broader SaaS data.

## Heavy distributed infrastructure in V1

Do not start with Kafka/Kubernetes/etc.

Build the smallest reliable core.

---

# 40. RESEARCHED TOOLING

Previously evaluated:

## GYB

Got Your Back.

Open-source Gmail backup/restore CLI.

Useful reference for Gmail backup mechanics.

## Open Archiver

Open-source email archiving/eDiscovery.

Relevant because it demonstrates demand for self-hosted email archive/search and customer-controlled storage.

## restic

Strong reference for:

- dedupe
- encrypted backup
- incremental data
- repository concepts

Useful as inspiration, but Burrow is not merely a wrapper around restic.

## Kopia

Useful reference for:

- snapshots
- dedupe
- encryption
- multiple storage backends
- recovery-oriented design

## rclone

Useful reference for:

- storage backend abstraction
- transfer reliability
- hash verification
- cloud/local targets

## n8n

Useful market/reference point.

Do not use as Burrow's core architecture unless there is a compelling isolated use.

---

# 41. ARCHITECTURE BUY-VS-BUILD PRINCIPLE

Before implementing major infrastructure, evaluate existing tools.

Potential language/runtime candidates:

- TypeScript / Bun
- Go
- Rust
- Python

Do not decide based purely on language preference.

Evaluate:

- Gmail libraries
- cloud SDK quality
- S3 compatibility
- filesystem support
- concurrency model
- packaging
- local daemon story
- cross-platform support
- developer velocity
- long-term operational reliability

The user explicitly does not want the architecture locked to one language before this evaluation.

---

# 42. TECHNICAL QUALITY BAR

Burrow is a trust product.

Therefore correctness beats cleverness.

Engineering standards should include:

- idempotency
- deterministic object identity
- checksums
- explicit state transitions
- durable checkpoints
- resumable jobs
- failure injection
- recovery tests
- reconciliation tests
- corrupt-state tests
- interrupted-upload tests
- partial-replica tests
- expired-credential tests
- provider API drift handling
- import/export round-trip tests

A successful sync is not sufficient.

A successful restore is the ultimate test.

---

# 43. WHAT THE NEXT ENGINEERING SESSION SHOULD DO

Do NOT start by coding the Gmail connector immediately.

First produce these artifacts:

## A. V1 technical specification

Define:

- object model
- source contract
- storage contract
- state schema
- job model
- checkpoint model
- replica model
- verification model
- archive bundle format
- recovery semantics

## B. Build-vs-buy matrix

Evaluate:

- Gmail libraries
- Pub/Sub
- S3 SDKs
- local storage libraries
- SQLite
- search/indexing
- encryption
- archive formats
- existing sync engines
- backup engines such as restic/Kopia

## C. Failure model

Explicitly enumerate:

- provider outage
- notification loss
- process crash
- machine shutdown
- partial upload
- corrupted DB
- deleted object
- stale checkpoint
- invalid credentials
- storage outage
- duplicate events
- reordered events
- schema migration failure

## D. Minimal end-to-end spike

Build:

```text
Gmail
  ↓
ingest
  ↓
SQLite
  ↓
local filesystem
  ↓
search
  ↓
restore
```

Then add:

```text
local filesystem
  +
S3-compatible replica
```

Then test:

```text
delete from Gmail
      ↓
recover from Burrow
```

Do this before broad connector work.

---

# 44. OPEN DECISIONS

These are intentionally NOT settled:

## Product

- exact free vs paid boundary
- exact pricing
- initial web UI technology
- self-hosting policy
- open-source extent
- managed storage offering
- multi-user/team model

## Architecture

- exact programming language
- exact queue technology
- exact search technology
- exact object encoding
- exact archive bundle format
- exact encryption/key management
- exact cloud orchestration

## Source model

- exact canonical email schema
- provider version semantics
- attachment handling
- thread representation
- deletion semantics

## Recovery

- restore-as-new vs overwrite defaults
- conflict semantics
- alternate account flows
- partial restore behavior

## Business

- exact ICP vertical order
- acquisition strategy
- partner/MSP strategy
- enterprise expansion

## Brand

- trademark
- final domain
- GitHub organization
- package namespaces
- visual brand identity

Do not treat these as already decided.

---

# 45. ARTIFACTS ALREADY CREATED

Current files:

```text
/mnt/data/burrow_product_thesis_v2.pdf
/mnt/data/burrow_pitch_deck_v2.pptx
```

The PDF is the refined product thesis.

The PPTX is the refined pitch deck.

The prior source thesis also exists:

```text
/mnt/data/saas_data_ownership_recovery_product_thesis.docx
```

The v2 thesis and deck were specifically refined for:

- typography
- hierarchy
- diagrams
- spacing
- narrative flow
- clearer architecture diagrams
- cleaner product language
- stronger recovery story
- more coherent Burrow branding

The newest files should be treated as the current presentation artifacts.

---

# 46. CURRENT PRODUCT NARRATIVE

The narrative should flow:

```text
SaaS dependency
      ↓
independent recovery gap
      ↓
Burrow thesis
      ↓
logical archive
      ↓
customer-controlled storage
      ↓
local/cloud runtime
      ↓
verification + recovery
      ↓
simple user experience
      ↓
focused ICP
      ↓
competitive wedge
      ↓
moat
      ↓
business model
      ↓
V1
```

Avoid starting with implementation details.

Lead with the customer problem and recovery outcome.

---

# 47. FRESH-SESSION INSTRUCTIONS

Paste the following into a new session along with this document:

> You are continuing work on **Burrow**, a local-first, cloud-optional SaaS data ownership and recovery product.
>
> Treat the attached Burrow Handoff as the source of truth for decisions already made.
>
> Do not re-litigate the basic product thesis unless new evidence requires it.
>
> The core product definition is:
>
> **Burrow continuously copies critical SaaS data into customer-controlled storage, verifies that the copy is recoverable, and provides a unified interface for search, restore, and portability.**
>
> Core positioning:
>
> **Your SaaS data. Your storage. Your copy.**
>
> Core primitives:
>
> **Source → Object → Storage → Runtime → Consumption**
>
> Runtime modes:
>
> **Local-first + cloud-optional**
>
> Initial wedge:
>
> **Google Workspace / Gmail + attachments**
>
> V1 success criterion:
>
> **Connect → protect → delete → recover**
>
> Do not expand into a giant connector catalog.
>
> Do not turn Burrow into a general automation platform.
>
> Do not make Telegram the core architecture.
>
> Do not introduce heavy distributed infrastructure before workload justifies it.
>
> The immediate next step is to produce an implementation-grade V1 technical specification covering:
>
> 1. canonical object model
> 2. source contract
> 3. storage contract
> 4. SQLite schema
> 5. queue/job model
> 6. checkpoint/reconciliation model
> 7. replica model
> 8. integrity/verification model
> 9. archive bundle/export-import format
> 10. recovery semantics
> 11. Gmail connector boundary
> 12. local runtime architecture
> 13. test strategy and failure injection
> 14. build-vs-buy recommendations
>
> For every architectural decision, explain:
>
> - why
> - alternatives considered
> - tradeoffs
> - what is reversible
> - what becomes expensive to change later
>
> Optimize for correctness, portability, recoverability, and simplicity.

---

# 48. CODEx BOOTSTRAP PROMPT

For a Codex implementation session, use:

```text
Read BURROW_HANDOFF.md completely before changing code.

You are implementing Burrow V1.

Do not invent a new product direction.
Do not expand scope without explicit justification.
Do not begin with UI polish.
Do not add infrastructure merely because it is familiar.

First inspect the repository and existing tooling.

Then produce:
1. architecture map
2. build-vs-buy matrix
3. proposed V1 repository structure
4. canonical data model
5. state schema
6. source/storage/runtime interfaces
7. failure model
8. implementation sequence

Only after that begin implementation.

Primary end-to-end goal:

Gmail
  -> incremental ingestion
  -> canonical objects
  -> SQLite state
  -> local filesystem
  -> S3-compatible replica
  -> search
  -> restore

Then prove:

Gmail object exists
  -> Burrow captures it
  -> object is replicated
  -> source object is deleted
  -> Burrow still finds it
  -> Burrow restores it
  -> restore is verified

Every important operation must be:
- idempotent
- resumable
- observable
- testable

Treat recovery correctness as the primary engineering metric.
```

---

# 49. FINAL PRINCIPLES

Keep these visible throughout the project:

### Principle 1

> **Recovery is the product.**

### Principle 2

> **Storage is replaceable.**

### Principle 3

> **Runtime is replaceable.**

### Principle 4

> **Logical data must remain portable.**

### Principle 5

> **Push events are triggers, not truth.**

### Principle 6

> **Capture once, persist locally, replicate independently.**

### Principle 7

> **A backup that has never been restored is an assumption.**

### Principle 8

> **V1 should prove one recovery loop exceptionally well.**

### Principle 9

> **Customer-controlled storage is a feature, not an implementation detail.**

### Principle 10

> **Complex underneath. Calm on top.**

---

# 50. ONE-SCREEN SUMMARY

```text
                    BURROW

             Keep a copy of what matters.

                     SaaS
                      |
                  ingestion
                      |
                canonical data
                      |
                   BURROW
                  /       \
             replica A   replica B
                /           \
             local        R2 / S3 / NAS
                  \       /
                   \     /
                 recovery
                 /      \
             search    restore
                    |
                  export

Runtime:
  LOCAL  <------ same core ------>  CLOUD

Product:
  local-first
  cloud-optional
  customer-controlled
  portable
  recovery-first

V1:
  Gmail + attachments
  incremental sync
  reconciliation
  SQLite
  local filesystem
  S3-compatible storage
  checksums
  dedupe
  retries
  health
  search
  restore
  export

Demo:
  CONNECT → PROTECT → DELETE → RECOVER

North star:
  % of protected objects successfully recovered
```

---

## HANDOFF END

The next session should pick up at **implementation-ready architecture**, not at product ideation.