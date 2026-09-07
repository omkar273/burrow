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
| D3 | **ent** ORM with **ent's own auto-migration** (`Schema.Create`), foreign keys on and dropping off. Version in SQLite's `PRAGMA user_version` | **Reversed 2026-09-08.** This decision originally mandated Atlas versioned migrations and explicitly forbade `Schema.Create`, on the grounds that the schema is a portability contract. Atlas was dropped because it made `ent/schema` and a directory of `.sql` files two sources of truth kept in step by a copy step, and a toolchain pin, for a single-file local database. `ent/schema` is now the only source of truth. What was preserved: the version identity (`SchemaVersion` constant → `user_version`), reviewable SQL before applying (`Schema.WriteTo` behind `make migrate-dry-run`), and the foreign keys. What was given up: per-change migration history, and a hand-written escape hatch for destructive changes. `WithDropColumn(false)` means a removed field leaves its column behind — divergence has to be handled deliberately when it first arises. |
| D4 | **modernc.org/sqlite**, registered under the `sqlite3` driver name | Pure Go, FTS5 included, cross-compiles from macOS to Linux/ARM. A cgo driver forfeits the single-binary story. Needs a ~30-line wrapper because mattn registers as `sqlite3` and modernc as `sqlite`. |
| D5 | **uber fx** for DI, composed from **per-package `fx.Module`s** | `fx.Lifecycle` gives ordered start/stop for background workers. Per-package modules keep `main.go` at ~40 lines — FlexPrice's rule of registering everything in `main.go` produced a 734-line file. |
| D6 | **The blob is the raw `.eml`**; everything else is derived index | IMAP `APPEND` takes raw RFC822, so we restore exactly the bytes we stored and byte-equality is directly assertable. A `.eml` also opens in any mail client with no Burrow installed — the portability property. |
| D7 | **Raw-only blobs at M1**; attachments extracted as separate blobs at M4 | Deferred dedup costs ~2.4× on attachment bytes. Shredding the `.eml` and recomposing on restore must be byte-exact (base64 wrapping, MIME boundaries, header folding) — that risk is not taken before a passing restore test exists to catch it. |
| D8 | **Prefixed k-sortable ULIDs** (`obj_`, `ver_`, `blob_`, `rep_`, `src_`, `job_`) | Makes the four-identity discipline (§4) visible at a glance; a `rep_` cannot be silently passed where an `obj_` belongs. Pattern adopted from FlexPrice. |
| D9 | **99designs/keyring** for provider credentials, with explicitly pinned backends | zalando/go-keyring is Secret-Service-only and hard-fails on a headless NAS — the deployment the README leads with. 99designs falls back to an encrypted file backend. |
| D10 | **Profile = data directory**, not a domain concept | `TenantID` is a security boundary enforced per query; a profile is an organizational one with no security claim, since one OS user reads every profile's files anyway. A scoping column would advertise isolation the process cannot enforce. |
| D12 | **IMAP** (`emersion/go-imap/v2`) rather than the Gmail API, authenticated by app password in V1 with XOAUTH2 pluggable later | One connector serves every mail provider, `APPEND` is the restore primitive the API only emulated, and no Cloud project or OAuth verification is needed to start. Costs the scoped credential; see §6. |
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
  internal/
    types/ errors/ config/ validator/
    sqlite/                          handle, WithTx, Atlas migrations (embedded)
    domain/{object,source,blob}/      models + interfaces, zero third-party imports
    repository/ent/                   implements domain interfaces via Querier(ctx)
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

### Schema version

The archive's schema version is the `SchemaVersion` constant, written to
SQLite's `user_version` header field. There is no bookkeeping table and no
migration directory.

It cannot disagree with the schema, because there is no second record to drift.
Any tool that opens a SQLite file can read it. And opening an archive whose
version exceeds this build's is refused rather than half-read — that archive was
written by a newer Burrow.

The constant is bumped by hand when `ent/schema` changes in a way another
Burrow would need to understand. That is one visible line in a diff, which is
the point: a schema change that matters should be an explicit act.

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

## 6. Mail access: IMAP, not the Gmail API

**Decided 2026-09-08, replacing the Gmail API.** The spec previously specified
`users.messages.get?format=RAW` and `users.messages.insert` behind OAuth. It now
specifies IMAP.

### Why

Gmail's IMAP exposes what the connector needs through the documented
`X-GM-EXT-1` extensions — confirmed advertised by `imap.gmail.com`:

| Need | IMAP |
|---|---|
| Provider message id (`external_id`) | `X-GM-MSGID` |
| Thread id | `X-GM-THRID` |
| Labels, readable and settable | `X-GM-LABELS` |
| Canonical RFC822 bytes | `FETCH BODY.PEEK[]` |
| Restore | `APPEND` |

`APPEND` is the decisive one. `messages.insert` was chosen because it has
IMAP-APPEND semantics; IMAP simply *is* that primitive, without the indirection.

And one connector reaches every provider — Fastmail, Proton Bridge, Zoho,
self-hosted — rather than Gmail alone. For a product positioned as "your SaaS
data, independently kept", that is a materially larger claim than a Gmail
backup tool.

### The cost, stated plainly

**An app password cannot be scoped.** It grants full mailbox access including
delete, does not expire, and is not per-capability. That is a worse trust
posture than the OAuth design it replaces, where read access was requested at
connect and write only at first restore. It sits uneasily against this repo's
own "not a credential proxy" line, and it is accepted knowingly rather than
overlooked.

**The mitigation is that the credential and the protocol are independent.**
`imap.gmail.com` advertises `AUTH=XOAUTH2`, so the same IMAP connector can
authenticate with an OAuth token instead of an app password. The connector must
therefore treat auth as pluggable from the start: app password for V1 because
it needs no Cloud project, XOAUTH2 later for users who want scoped, revocable
credentials. Choosing IMAP does not forfeit that.

**Throughput is latency-bound, not bandwidth-bound — measured 2026-09-08.** Gmail
caps IMAP at 2,500 MB/day down and 500 MB/day up, but a real 19,033-message
mailbox is only ~223 MB, or 9% of one day's allowance. The binding constraint is
per-message round trips: a 12 KB fetch took 0.37 s, which is latency, not
transfer. **M2 pipelines and batches FETCH rather than throttling on bytes.**
Bandwidth only binds above roughly 2 GB of mail.

### Two Gmail connectors, not one

Gmail gets two connector implementations behind the same ports:

| | `gmailimap` (V1) | `gmailapi` (later) |
|---|---|---|
| Transport | IMAP | Gmail REST API |
| Credential | app password, or XOAUTH2 | OAuth, scoped and incremental |
| Setup cost | 2FA + an app password | Cloud project + consent screen |
| Limit shape | 2,500 MB/day, bandwidth-bound | ~300 msg/min, request-bound |
| Reach | every IMAP provider | Gmail only |

Both implement `source.Connector` and `source.Restorer` unchanged. This is what
the ports were for: the transport is an adapter detail, and the archive does not
know which one filled it.

**They should produce the same `external_id`.** Gmail's API message id appears to
be the hexadecimal form of the IMAP `X-GM-MSGID` integer. If that holds, an
archive ingested over IMAP is compatible with one ingested over the API, and a
user can switch transports without re-ingesting a mailbox. `external_id` is
therefore normalised to the hex form regardless of connector. **This is M0
question 9** — the whole two-connector story rests on it, and it is unverified.

`gmailapi` is not scheduled. It exists in this spec so the IMAP connector is not
written in a way that forecloses it: nothing Gmail-transport-specific may leak
past the adapter boundary.

### A gap in the library

`emersion/go-imap/v2` has **no `X-GM-*` support** — verified by inspecting
`v2.0.0-beta.8`. Labels, message ids and thread ids therefore need raw FETCH
items regardless of which client library is used, and whether the library
permits arbitrary fetch items is M0 question 10.

### Checkpoints

`UIDVALIDITY` plus `UIDNEXT` per mailbox replaces `historyId`, and the failure
mode is identical: when `UIDVALIDITY` changes, every stored UID is meaningless
and a full resync is required. That maps onto the existing typed
`ErrCheckpointExpired` without changing the port.

**Gmail offers neither `CONDSTORE` nor `QRESYNC`** — confirmed both before and
after authentication on 2026-09-08. There is no MODSEQ-based "what changed since
*n*" query.

M3 therefore fetches UIDs above the stored `UIDNEXT` for additions, and detects
label changes and deletions by reconciling over the UID range on a schedule.
Reconciliation is mandatory rather than a safety net — which is what handoff §30
argued for on principle, now forced by the protocol.

`UIDPLUS` is not advertised, yet `APPEND` returns `[APPENDUID <validity> <uid>]`
anyway. Restore uses it to learn the new UID directly, but as a best-effort
optimisation with a fallback: an undeclared capability can be withdrawn without
notice.

### Verified end to end

`FETCH BODY.PEEK[]` → sha256 → `APPEND` → `FETCH` returned **byte-identical**
content against a real mailbox on 2026-09-08 (12,309 bytes, matching sha256).
The acceptance criterion stands as written: byte equality, not a normalised
comparison. `X-GM-MSGID`, `X-GM-THRID` and `X-GM-LABELS` all behave as
documented. See the [M0 findings](../plans/2026-09-07-m0-findings.md).

### Restore is still not idempotent

`APPEND` creates a new message with a new UID on every call, exactly as
`messages.insert` did. The reasoning and the V1 stance are unchanged: restore
is an explicit operator-initiated act, never retried automatically, and a
timeout is reported rather than retried.

### Library

`github.com/emersion/go-imap/v2`, which supports the extensions above.

**It is at `v2.0.0-beta.8`.** Taking a beta dependency for the connector of a
data-integrity product is a real risk and is taken deliberately: v1 is stable
but has an older API, and the alternative is hand-rolling IMAP. The mitigation
is that the blob is raw RFC822 bytes whose hash we verify ourselves, so a
library defect corrupts a fetch loudly rather than the archive silently.

---

## 7. Credentials and profiles

### Profiles

`--profile <name>` or `BURROW_PROFILE` resolves `~/.burrow/profiles/<name>/{state.db, objects/, config.toml}`. Default profile is `default`, so single-profile users never type the flag.

Multiple Gmail accounts **within** one profile are simply multiple `Source` rows — cross-account search works, and a message sent to two of your addresses dedupes to one blob.

### Credentials

The provider credential — a Gmail app password in V1, an OAuth refresh token if
XOAUTH2 is added — lives in `99designs/keyring`, service `burrow`, key
`<profile>/<source_id>`.

An app password is unscoped and does not expire, which makes where it lives
matter more than it would for a revocable token: anything that reads it holds
full mailbox access, including delete. It is never logged, never written to the
archive, and never included in an export.

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
| Rejected credential | IMAP `AUTHENTICATE` fails; `ErrCredentialExpired`. An app password does not expire, so this means it was revoked, 2FA was turned off, or a Workspace admin disabled app passwords. |
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
IMAP login (app password) → FETCH BODY.PEEK[] → sha256 → write to disk
      → APPEND → FETCH the new UID → compare hashes
```

Also answers: does the darwin keyring backend pull in cgo?

**Rationale.** M1 is roughly 2,000 lines of structure standing on untested assumptions. The three most serious findings in this design — restore-creates-duplicates, the scope decision, and the rate-limit ceiling — are all things a hundred lines of throwaway code surfaces in an afternoon. Half a day of spike either validates the thesis end-to-end or surfaces the next surprise while the design is still free to change.

**Output is an answer, not code.**

### M1 — walking skeleton

```text
burrow connect gmail        prompt for an app password; store in keyring
burrow pull --limit 1       FETCH → sha256 → blob on disk → rows
burrow restore <obj_id>     read blob → IMAP APPEND
burrow profile list|create|delete
```

Carries the scaffolding: ent schemas, the modernc wrapper driver, the first Atlas migration, the fx module graph.

**Done means:** `sha256(stored bytes) == sha256(bytes Gmail returns for the newly appended message)`.

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
