# Burrow Engine — V1 Design

**Date:** 2026-09-07
**Status:** Design approved; not implemented.
**Supersedes:** the stack paragraph in [README.md](../../../README.md) ("TypeScript and Bun for … the first local runtime") and the `bun test` engine rule in [AGENTS.md](../../../AGENTS.md).
**Source material:** [braindump/BURROW_HANDOFF.md](../../../braindump/BURROW_HANDOFF.md) §8–12, §23–35, §41–44.

---

## 1. Scope

This spec covers the **engine**: the local process that connects to Gmail, copies messages onto storage you control, and restores them. It does not cover the web UI, hosted runtime, or any source other than Gmail.

The handoff's §23 "V1 must have" list is six independent subsystems. It is decomposed here into nine milestones (§9). **This spec is implemented by M0 and M1 only.** M2 onward get their own specs.

**Acceptance test for the engine, per [AGENTS.md](../../../AGENTS.md):** a message deleted from Gmail is restored from Burrow, and the restored bytes hash-match the stored bytes. Not a completed job. Not a green icon.

---

## 2. Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | **Go** for the engine; TypeScript stays in `packages/web` | Single static binary, no runtime to install on a NAS. Resolves handoff §41, which left language open. |
| D2 | **Hexagonal / ports-and-adapters**, layer rules written into AGENTS.md | `domain/` holds models + interfaces with zero external deps; adapters implement them. Layer table format adopted from FlexPrice. |
| D3 | **ent** ORM + **Atlas versioned migrations**; never `Schema.Create` auto-migrate | The SQLite schema is a published portability contract (§10 of the handoff — the archive bundle carries `schema-version`). Auto-migrate makes the schema an unversioned side effect of Go structs. |
| D4 | **modernc.org/sqlite**, registered under the `sqlite3` driver name | Pure Go, FTS5 included, cross-compiles from macOS to Linux/ARM. A cgo driver forfeits the single-binary story. Needs a ~30-line wrapper because mattn registers as `sqlite3` and modernc as `sqlite`. |
| D5 | **uber fx** for DI, composed from **per-package `fx.Module`s** | `fx.Lifecycle` gives ordered start/stop for background workers. Per-package modules keep `main.go` at ~40 lines — FlexPrice's rule of registering everything in `main.go` produced a 734-line file. |
| D6 | **The blob is the raw `.eml`**; everything else is derived index | `messages.insert` takes raw RFC822 with IMAP-APPEND semantics, so we restore exactly the bytes we stored and byte-equality is directly assertable. A `.eml` also opens in any mail client with no Burrow installed — the portability property. |
| D7 | **Raw-only blobs at M1**; attachments extracted as separate blobs at M4 | Deferred dedup costs ~2.4× on attachment bytes. Shredding the `.eml` and recomposing on restore must be byte-exact (base64 wrapping, MIME boundaries, header folding) — that risk is not taken before a passing restore test exists to catch it. |
| D8 | **Prefixed k-sortable ULIDs** (`obj_`, `ver_`, `blob_`, `rep_`, `src_`, `job_`) | Makes the four-identity discipline (§4) visible at a glance; a `rep_` cannot be silently passed where an `obj_` belongs. Pattern adopted from FlexPrice. |
| D9 | **99designs/keyring** for OAuth tokens, with explicitly pinned backends | zalando/go-keyring is Secret-Service-only and hard-fails on a headless NAS — the deployment the README leads with. 99designs falls back to an encrypted file backend. |
| D10 | **Profile = data directory**, not a domain concept | `TenantID` is a security boundary enforced per query; a profile is an organizational one with no security claim, since one OS user reads every profile's files anyway. A scoping column would advertise isolation the process cannot enforce. |
| D11 | **No** Kafka, Temporal, Redis, or ClickHouse | Handoff §24 and §39 explicitly reject distributed infrastructure in V1. FlexPrice carries ~60 direct dependencies because it is a multi-tenant SaaS API; M1 has ~6. |

### Rejected

- **Pub/Sub push for Gmail** — §29/§30: push is a trigger, not truth. The reconciliation path is mandatory anyway (see §6), so build that first; push is a later latency optimization.
- **An external job-queue library** (goqite, backlite, River) — a `jobs` table in the *same* SQLite file gives transactional enqueue, which is what makes "a job can run twice without corrupting the archive" (§28) provable. A library puts job state in a second place.
- **FlexPrice's `BaseModel`** (`TenantID`, `CreatedBy`, `UpdatedBy`, `Status`) — multi-tenant SaaS concerns. Burrow's `BaseMixin` is `CreatedAt`/`UpdatedAt`. Its `Status` soft-delete is also the wrong shape: deleted-at-source is emphatically **not** deleted-from-archive, and that distinction is the product.
- **restic / Kopia as libraries** — they are programs with their own repository formats, not embeddable stores.

---

## 3. Layout

```text
go.mod  go.sum                        the only Go files at the repo root
packages/engine/
  engine.go  migrate.go               the entire public surface
  cmd/burrow/                        the daemon
  cmd/migrate/                        schema migrations, with --dry-run
  ent/  ent/schema/  ent/schema/mixin/
  migrations/versioned/               Atlas
  internal/
    types/ errors/ config/ sqlited/ validator/
    domain/{object,source,blob}/      models + interfaces, zero third-party imports
    repository/ent/                   implements domain interfaces
    storage/  storage/localfs/        driven adapter
    source/gmail/                     driven adapter
    service/                          use cases: pull · restore · verify
    testutil/                         FakeSource, FakeStore, temp-file SQLite
packages/web/                         TypeScript, bun workspace (M9)
scripts/check-layers.sh               enforces both layer rules below
```

`cmd/` for binaries and `internal/` for implementation is the Go convention —
restic, syncthing and rclone all use it. It sits under `packages/` so the repo
keeps a single top-level source directory shared with the bun workspace, whose
glob is `packages/*`.

**One consequence, deliberately accepted.** Go's `internal` rule is relative to
the enclosing directory, so `packages/engine/cmd/...` *can* import
`packages/engine/internal/...`; the compiler no longer enforces the facade
boundary as it did when the binaries lived outside. The facade is still the
intended API — it is what makes the engine embeddable for an AGPL core others
may run in-process — so the rule moved from the compiler to
`scripts/check-layers.sh`, which fails `make lint` if anything under `cmd/`
imports `internal/`. Verified by breaking it on purpose.

`packages/engine/internal/` is enforced by the compiler, not convention:
`packages/engine/cmd/` importing it fails `make lint`. The facade in `engine.go` is
therefore the real driving port, and the engine is embeddable — which
matters for an AGPL core others may run in-process.

### Layer rules (never violate)

- `internal/domain/` — models and interfaces only. Zero third-party imports beyond `types`/`errors`. Cannot reach the network or the disk.
- Domain interfaces are implemented in `internal/repository/` (persistence) or in adapter packages (`source/gmail`, `storage/localfs`).
- Generated `ent.*` types **never escape `internal/repository/ent/`**. Each domain package owns `FromEnt` / `FromEntList` converters.
- `internal/service/` holds all orchestration. Adapters call nothing upward.
- Every dependency is provided by its own package's `fx.Module`; `main.go` composes modules and provides nothing itself.
- **fx does not appear in tests.** Tests construct dependencies directly.

---

## 4. Data model

### Four identities, never conflated

| Identity | Answers | Form |
|---|---|---|
| `ObjectID` | which thing in my archive | `obj_<ulid>` — ours, stable forever |
| `ContentHash` | which bytes | `sha256:<hex>` — the dedup key |
| `VersionID` | which state, when | `ver_<ulid>` — k-sortable |
| `ReplicaID` | which physical copy, where | `rep_<ulid>` — per (blob, backend) |

Gmail's message ID is none of these. It is `external_id`: a provider fact we record and never build identity on.

### Entities

**Truth** — `Source`, `Object`, `ObjectAlias`, `ObjectVersion`, `Blob`, `Backend`, `Replica`, `Job`. Never reconstructible; loss is data loss.
**Derived** — `MessageIndex`, the FTS5 table. Rebuildable from the blobs alone; safe to drop and regenerate offline.
**Operational** — `Label`, `ObjectLabel`, `GmailSyncState.history_id`. Rebuildable **only by asking the provider**, at a cost.

Chain: `Object → ObjectVersion → Blob → Replica[]`.

`Blob` holds `(content_hash, size_bytes)` and knows nothing about backends. `Replica` holds `(blob, backend, key, state, last_verified_at)`. That separation *is* handoff §8's "one logical copy, many physical replicas" — it is what allows adding, repairing, or migrating a backend without touching restore.

### The truth/derived rule

Derived tables **must be droppable and rebuildable from the blobs alone**, offline, with no network. That makes the archive bundle (§10 of the handoff) definable as "ship truth, rebuild derived on import".

Gmail label IDs and `historyId` are **not** in the RFC 2822 bytes — they are provider-side metadata carried alongside the message — so they can never be rebuilt from a blob. They are therefore *operational*, not derived: losing them costs a metadata re-fetch (labels, via `format=METADATA`) or a forced full resync (`historyId`), but never archived content. The archive bundle ships truth, rebuilds derived, and re-fetches operational state on first sync after import.

If something cannot be rebuilt from blobs **or** re-fetched from the provider, it belongs in truth.

### Two model decisions with teeth

**Label changes do not create versions.** Labels are mutable derived state. Versions are created only when content bytes change. Otherwise a busy mailbox generates thousands of versions all pointing at one blob.

**`ObjectVersion` is unvalidated by the first connector.** Gmail RAW bytes are immutable, so every Gmail message has exactly one version forever. The table ships in a schema that is a portability contract and is exercised only at N=1 until a mutable source arrives. It is retained because retrofitting versioning into a published schema is worse than carrying it unused — but it is explicitly untested, and the first mutable connector must be treated as validating it for the first time.

---

## 5. Ports

```go
// internal/domain/source
type Source interface {
    ID() types.SourceID
    FullList(ctx context.Context, cur Cursor) (refs []ObjectRef, next Cursor, err error)
    Changes(ctx context.Context, cp Checkpoint) (ch []Change, next Checkpoint, err error)
    Fetch(ctx context.Context, ref ObjectRef) (io.ReadCloser, ObjectMeta, error)
    Health(ctx context.Context) error
}

// Separate interface: read and write-back are different capabilities,
// and a future connector may be read-only.
type Restorer interface {
    Restore(ctx context.Context, r io.Reader, opts RestoreOpts) (externalID string, err error)
}
```

`Changes` returns a typed `ErrCheckpointExpired`. Gmail's 404-on-stale-`historyId` is a documented, roughly weekly event, not an edge case — a typed error forces every caller to carry a full-resync path. API reality belongs in the contract.

```go
// internal/storage
type Store interface {
    Put(ctx context.Context, key string, r io.Reader, size int64) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Stat(ctx context.Context, key string) (Stat, error)
    Exists(ctx context.Context, key string) (bool, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) iter.Seq2[string, error]
    Capabilities() Caps
}
```

**`Put`'s postcondition is durability, not acceptance.** When `Put` returns nil the bytes must be complete and durable at `key` — for `localfs` that means `fsync` on the temp file, `rename`, then `fsync` on the parent directory, because the rename's directory entry is not durable otherwise. The whole blob-before-row ordering (§8) rests on this, so a backend that cannot promise it must fail `Put` rather than return early.

`Capabilities()` is handoff §27's capability matrix: backends differ in multipart, range reads, and atomic-write semantics, and the design does not pretend otherwise. `AtomicRename: false` means the service must stage, verify, then commit; a backend offering neither atomic publish nor verify-after-write is rejected at configuration time rather than silently trusted.

Blob keys are content-addressed with fan-out over the **hex digest with the algorithm prefix stripped** — `sha256:abcd1234…` becomes `objects/ab/cd/abcd1234…`. Slicing the canonical value instead would put every object under `sh/a2` and defeat the fan-out entirely.

---

## 6. Gmail specifics

### Fetch and restore

- Read: `users.messages.get` with `format=RAW` → base64url RFC 2822 → decode → those bytes are the blob.
- Restore: `users.messages.insert` (**not** `import` — import re-runs delivery scanning and classification, which can reclassify or drop the message, making restore lossy).

### Restore creates a duplicate unless designed against

`messages.insert` mints a **new** Gmail message ID. The next incremental sync sees an unknown ID and would ingest it as a new `Object`; restore twice and the archive forks.

**Mitigation: restore records the alias, ingest never infers one.** Identical bytes do not mean the same logical object — two genuinely distinct messages can share content, and merging them on a hash match would silently lose one and its lifecycle. Only restore knows the lineage, because it wrote those exact bytes and received that `external_id` back, so restore records the `ObjectAlias` directly.

Ingest therefore resolves `external_id` (following aliases), and on a miss creates a new `Object` and `Version` that **reference the existing `Blob` row** when the hash already exists. Deduplication stays at the blob layer, where it belongs; identity stays with the provider's object.

`ObjectAlias` is unique on `(source_id, external_id)` — Gmail IDs are unique only within a mailbox, and one profile holds many sources. An alias may not shadow an identity an `Object` already claims; `AddAlias` enforces that in the same transaction as the insert. Restore additionally records `restored_from_version_id` provenance.

**Restore is not idempotent, and the archive cannot make it so.** `messages.insert` creates a new message on every call and never deduplicates on content, so a retry after an ambiguous timeout leaves two identical messages in the mailbox. Aliasing prevents the *archive* forking; it does nothing about the *mailbox*. V1 therefore treats restore as an explicit, operator-initiated act: it is never retried automatically, a timeout is reported rather than retried, and the operator is told a duplicate may exist. Automatic retry needs a source-side marker — a Burrow label, or an `X-Burrow-Restore-Id` header searched before inserting — which is deferred with the rest of restore hardening.

### OAuth scopes — a product decision, not a detail

`gmail.readonly` **cannot** restore. `messages.insert` requires one of `gmail.insert`, `gmail.modify`, or `https://mail.google.com/`.

**Decision: two scopes, requested incrementally.** `gmail.readonly` at `burrow connect`; `gmail.insert` (the narrowest — insert and import only; no read, no delete, no send) requested only when the user first attempts a restore. A backup tool that cannot read data back is useless; one holding unused write access is a liability. Incremental consent resolves both.

Both are Google *restricted* scopes, and owning the Cloud project is **not** by itself an exemption. An unverified app is limited to test users on the consent screen (100), and shows the unverified-app warning. That covers the self-host-your-own-project case — which is the V1 distribution model — but any deployment serving external users at scale still needs verification and, for restricted scopes, a security assessment. The spec claims the former, not the latter.

### Rate limits are a product constraint

Google's published quota is **6,000 units/minute/user/project**, with `messages.get` at **20** units and `messages.list` at **5**. That is roughly **300 messages/minute** — plus an undocumented ~50-concurrent-request-per-mailbox ceiling that returns 429 well below the unit quota.

A 500k-message mailbox is therefore **~28 hours of pure API time at the theoretical best**, before any retry or backoff.

**Record the quota cohort per source.** Projects that used the API between November 2025 and April 2026 keep their older, more generous quotas; projects created on or after 1 May 2026 get the numbers above. The engine must not hard-code either: M2 adapts to observed 429s rather than to a compile-time constant, and the cohort is recorded so a projection can be explained.

An earlier draft of this spec cited 15,000 units/min with `get` at 5 — roughly 3,000 msg/min and a 2.8-hour projection. That was wrong by an order of magnitude. It is corrected here because M2 and M3 are sized against it: at 300 msg/min, initial sync is a multi-day background job, not an afternoon.

---

## 7. Credentials and profiles

### Profiles

`--profile <name>` or `BURROW_PROFILE` resolves `~/.burrow/profiles/<name>/{state.db, objects/, config.toml}`. Default profile is `default`, so single-profile users never type the flag.

Multiple Gmail accounts **within** one profile are simply multiple `Source` rows — cross-account search works, and a message sent to two of your addresses dedupes to one blob.

### Credentials

OAuth refresh tokens live in `99designs/keyring`, service `burrow`, key `<profile>/<source_id>`.

- **`AllowedBackends` is pinned explicitly per platform.** Left on auto-select, the library's backend choice is non-deterministic and a token can appear to vanish between runs.
- **Verify at M0 that the darwin keychain backend does not reintroduce cgo.** Build tags should keep it out of Linux/ARM cross-builds, but "should" is insufficient against a hard constraint. Fallback if it does: shell out to `/usr/bin/security`.
- **Honest limitation:** on a headless box the encrypted file backend needs a passphrase, supplied by env var or file for an unattended daemon. That is not materially stronger than a `0600` file. Keyring is a real improvement on macOS and desktop Linux and roughly a wash on a NAS. Documentation must say so.
- **`burrow profile delete` purges that profile's keyring entries.** Otherwise deleting a profile silently orphans live refresh tokens in the OS keychain forever.
- Secrets never enter the export bundle. Import re-authenticates. (Handoff §9.)

---

## 8. Failure model

### Handled at M1

| Failure | Handling |
|---|---|
| Crash mid-blob-write | Write to temp file → `fsync` → atomic rename. |
| Crash between blob and row | **Order is mandatory: fsync blob, then commit row.** The reverse leaves a dangling reference, which is corruption. This order leaves an orphan blob, which is harmless and GC-able precisely because it is content-addressed. There is no transaction spanning ent and the blob store. |
| Expired OAuth token | Refresh via `oauth2` token source; `ErrCredentialExpired` if refresh fails. |
| Same message pulled twice | Same content hash → no second blob, no second version, no second object. |
| Concurrent writers | SQLite in WAL mode with `busy_timeout`; single-writer discipline for background workers. modernc's behaviour under write contention is verified at M1, not assumed. |

### Deferred, explicitly

Provider outage backoff · partial multipart upload · corrupted DB recovery · reordered history events · schema migration failure · stale checkpoint beyond full-resync · storage backend outage.

### Error taxonomy

Sentinel errors with a builder (`ierr.NewError(...).WithHint(...).Mark(...)`), mechanism adopted from FlexPrice but with operational rather than HTTP-shaped codes: `ErrCheckpointExpired`, `ErrChecksumMismatch`, `ErrBlobMissing`, `ErrReplicaCorrupt`, `ErrCredentialExpired`, `ErrSourceUnavailable`. This list is handoff §43-C's failure model expressed as types.

---

## 9. Milestones

### M0 — throwaway spike

One file, no ent, no fx, no layers. **Explicitly labeled throwaway; the code is not kept.**

```text
OAuth → messages.get RAW → sha256 → write to disk
      → messages.insert → messages.get the new one → compare hashes
```

Also answers: does the darwin keyring backend pull in cgo?

**Rationale.** M1 is roughly 2,000 lines of structure standing on untested assumptions. The three most serious findings in this design — restore-creates-duplicates, the scope decision, and the rate-limit ceiling — are all things a hundred lines of throwaway code surfaces in an afternoon. Half a day of spike either validates the thesis end-to-end or surfaces the next surprise while the design is still free to change.

**Output is an answer, not code.**

### M1 — walking skeleton

```text
burrow connect gmail        OAuth loopback; token → keyring
burrow pull --limit 1       fetch RAW → sha256 → blob on disk → rows
burrow restore <obj_id>     read blob → messages.insert
burrow profile list|create|delete
```

Carries the scaffolding: ent schemas, the modernc wrapper driver, the first Atlas migration, the fx module graph.

**Done means:** `sha256(stored bytes) == sha256(bytes Gmail returns for the newly inserted message)`.

### M2–M9

M2 job engine (queue, checkpoints, retry, rate limiting) · M3 incremental sync + reconciliation · M4 attachments + dedup · M5 verify · M6 S3 replica · M7 search (FTS5) · M8 export bundle · M9 web UI.

Each gets its own spec.

---

## 10. Testing

Per [AGENTS.md](../../../AGENTS.md), every production function starts as a failing `go test`.

- **`FakeSource` and `FakeStore`** in `internal/testutil` — in-memory implementations so CI runs with **no Google credentials**. Without this, every future test is hostage to a live mailbox.
- **Real SQLite in tests, not a mocked ent client** (FlexPrice's rule; cheaper for us — a temp file, no testcontainers).
- **Gmail integration tests behind a build tag**, run locally against a dedicated throwaway account.
- **Atlas migration round-trip test.** SQLite cannot `ALTER COLUMN`, so Atlas performs table-rebuild migrations. This must be proven to round-trip cleanly *before* the schema is a portability contract carrying real data.
- Table-driven tests throughout.

**Contributor cost, stated plainly:** running the real Gmail path requires a Google Cloud project, an OAuth consent screen, and a throwaway account to insert into. [CONTRIBUTING.md](../../../CONTRIBUTING.md) names the Gmail loop as *the* contribution; the fakes cover most work, but this is a genuine barrier and should be documented in the contributor guide.

---

## 11. Repo debt this creates

To be paid in the M1 PR, not deferred:

- [AGENTS.md](../../../AGENTS.md) — **Done** — `go test` for the engine and binaries, `bun test` for `packages/web`; layer table added.
- [README.md](../../../README.md) — the "TypeScript and Bun for local dev and the first local runtime" paragraph and the `packages/` tree in "Layout and stack".
- [Makefile](../../../Makefile) / [mise.toml](../../../mise.toml) — pin Go; `test` and `lint` fan out to both toolchains; add `generate-ent` and `generate-migration`.
- [.gitignore](../../../.gitignore) — `~/.burrow` is outside the tree, but ensure no credential path can land in it.

---

## 12. Open decisions

Deliberately unsettled; each blocks a later milestone, none blocks M0 or M1.

### Dependencies deferred to their milestone

`go mod tidy` removes anything nothing imports, so these are recorded here rather than added early. Each is what restic, kopia or syncthing already use for the same job.

| Dependency | Milestone | Why |
|---|---|---|
| `golang.org/x/sync` (`errgroup`, `semaphore`) | M2/M3 | Gmail caps concurrent requests per mailbox at roughly 50 and returns 429 well below the unit quota. `semaphore.Weighted` bounds the fetch fan-out; `errgroup` gives first-error-wins cancellation. Used by restic. |
| `github.com/cenkalti/backoff/v4` | M2 | 429 and 5xx retry. Exponential backoff with jitter is easy to get subtly wrong by hand. Used by restic. |
| `github.com/minio/minio-go/v7` | M6 | S3-compatible object storage. **Both restic and kopia chose it over `aws-sdk-go-v2`** for the same S3-compatible-first requirement, which settles the open question below. |

### Open questions

| Question | Blocks |
|---|---|
| S3 SDK: `minio-go` vs `aws-sdk-go-v2` — evidence above favours `minio-go` | M6 |
| Archive bundle format specifics (handoff §10) | M8 |
| Encryption and key ownership model (handoff §34) | post-V1 |
| Whether `ObjectVersion` survives contact with a mutable source | first non-Gmail connector |
| Pub/Sub push as a latency optimization | post-M3 |
