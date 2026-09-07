<p align="center">
  <img src="logo.png" width="96" alt="Burrow mascot" />
</p>

<h1 align="center">Burrow</h1>

<p align="center">
  <strong>Your SaaS data.<br/>Your storage.<br/>Your copy.</strong>
</p>

<p align="center">
  Keep an independent, verifiable copy<br/>
  of the data your business depends on.
</p>

<p align="center">
  <a href="#get-your-burrow-running"><strong>Get started</strong></a>
  &nbsp;·&nbsp;
  <a href="#under-the-burrow">Architecture</a>
  &nbsp;·&nbsp;
  <a href="#build-with-us">Contribute</a>
</p>

<p align="center">Open source · Self-hostable · Cloud-optional</p>

<p align="center">
  <img src="docs/assets/burrow-hero.png" alt="Mascot beside a burrow, abstract data objects arriving" width="920" />
</p>

---

<p align="center">
  <img src="docs/assets/burrow-problem.png" alt="Mascot looking up at scattered, unowned data" width="720" />
</p>

```text
 Gmail      Drive      Slack      GitHub
   ↓          ↓          ↓           ↓
        Your business lives here.
                    ?
        Where's your independent copy?
```

<p align="center"><strong>Your business runs on SaaS. Your independent copy shouldn't have to.</strong></p>

---

<p align="center">
  <img src="docs/assets/aha.svg" alt="Access and sync checked; recovery still a question" />
</p>

<details>
<summary>Access is not a copy</summary>

Being able to open Gmail does not mean you have an independent copy.

</details>

<details>
<summary>Sync is not recovery</summary>

A completed job does not prove you can get the object back.

</details>

<details>
<summary>Recovery is the product</summary>

A backup isn't a backup until you know you can recover from it.

</details>

---

## Meet Burrow

<p align="center">
  <img src="docs/assets/burrow-meet.png" alt="Sources flowing into Burrow, then out to storage you control" width="920" />
</p>

<p align="center"><strong>Burrow gives your data another home.</strong></p>

<p align="center">
  Local-first. Cloud-optional. Quietly in the background.<br/>
  Backup is the mechanism. Ownership and recovery are the point.
</p>

---

## The loop

```mermaid
flowchart TD
  A[CONNECT] --> B[COPY]
  B --> C[VERIFY]
  C --> D[SEARCH]
  D --> E[RECOVER]
  E --> F[EXPORT]
```

<details>
<summary>CONNECT</summary>

Link a source. Gmail is the first one we are building.

</details>

<details>
<summary>COPY</summary>

Ingest into a canonical archive, then onto storage you control.

</details>

<details>
<summary>VERIFY</summary>

Checksums, completeness, consistency — not a green job icon.

</details>

<details>
<summary>SEARCH</summary>

Find it in *your* copy, not only in the provider UI.

</details>

<details>
<summary>RECOVER</summary>

Original account, another account, download, or a portable archive.

</details>

<details>
<summary>EXPORT</summary>

Leave with the archive. Secrets stay out of the plaintext dump.

</details>

---

## Your storage. Your choice.

<p align="center">
  <img src="docs/assets/burrow-storage.png" alt="Mascot in front of three burrow openings — local, object storage, NAS" width="920" />
</p>

```text
     Local FS          S3 / R2            NAS
        └───────────────┬───────────────┘
                        │
                     Burrow
```

<p align="center"><strong>Burrow doesn't need to become the owner of your backup.</strong></p>

---

<p align="center">
  <img src="docs/assets/features.svg" alt="Keep a copy, verify it, find it, get it back" />
</p>

---

## A green sync icon isn't enough

<p align="center">
  <img src="docs/assets/burrow-recovery.png" alt="Mascot emerging from the burrow carrying a data box" width="560" />
</p>

<p align="center"><strong>Recovery is the point.</strong></p>

<p align="center">
  Burrow treats recoverability as a first-class product property.<br/>
  Intact objects. Matching checksums. A restore you have actually done.
</p>

```text
Typical path                         Burrow

Your data                            Your data
    ↓                                    ↓
Provider                             Burrow
    ↓                                    ↓
Provider storage                     Your storage
    ↓                                    ↓
Provider recovery                    Your independent copy
```

Burrow should never become the next place you're locked into.

---

## Under the burrow

```mermaid
flowchart TD
  S[Sources] --> I[Ingestion]
  I --> O[Canonical objects]
  O --> ST[Storage]
  ST --> V[Verify]
  ST --> X[Index]
  V --> R[Recover]
  X --> F[Search]
```

Same engine locally (SQLite + disk) or hosted (Postgres + queue). Placement changes. The archive does not.

Ingestion is at-least-once. Writes are idempotent. Push events are triggers, not truth.

<details>
<summary>Want to see what's underneath?</summary>

Monorepo. Modular core. Several entry points — not microservices.

```text
apps/        web · api · agent · worker · cli
packages/    domain · core · connectors · storage · state · queue
             archive · recovery · search · crypto · observability
             contracts · ui
```

`apps/` run Burrow. `packages/` *are* Burrow.

Current implementation direction: TypeScript, Bun for local dev and the first local runtime, SQLite at home, Postgres when hosted, S3-compatible object storage. Choices — not the identity of the product.

</details>

### Why it's built this way

| | |
| --- | --- |
| **Ownership** | Storage is yours and replaceable. |
| **Portability** | Export reconstructs logical state, not a Docker volume. |
| **Recoverability** | Restore is the acceptance test. |
| **Reliability** | Checkpoints, reconcile, retries. Boring on purpose. |

---

## Who it's for

| Founder | Engineer | Self-hoster |
| --- | --- | --- |
| Don't let one SaaS account become a single point of failure. | A recovery engine around idempotency, verification, and portability. | Put the data where you want it. |

| Security | Ops | Contributor |
| --- | --- | --- |
| Credentials, storage, and keys under your control. | Know what's protected before you need it. | Infrastructure for data independence. |

---

## Open at the core

<p align="center">
  <img src="docs/assets/open-core.svg" alt="AGPL core plus optional cloud and enterprise" />
</p>

<p align="center"><strong>Everything required to own and recover your data is open.</strong></p>

<p align="center">
  AGPLv3 for own / copy / verify / recover / export.<br/>
  Cloud and enterprise for convenience and organizational scale — planned, not a catalog.
</p>

AGPL is copyleft, not “non-commercial,” and it does not forbid forks. **Burrow**, the logo, and the mascot are trademarks. Fork the code; don’t impersonate the burrow.

---

## Where we are

The idea is sharp. The first recovery loop is **not shipped**.

| Available | In progress | Planned |
| --- | --- | --- |
| This repo, the thesis, the architecture | Local runtime, Gmail + attachments, verify, search, restore, export, UI | S3 replica, more sources, customer-held keys, hosted cloud |

Drive, Microsoft 365, Slack, GitHub, Notion are **roadmap names**, not connectors.

---

## Get your burrow running

<a id="get-your-burrow-running"></a>

Nothing production-ready to install yet. Intended developer flow:

```bash
git clone https://github.com/<org>/burrow.git   # TODO: canonical remote
cd burrow
bun install                                     # TODO
bun dev                                         # TODO
```

<!-- TODO: docs, Docker, CI badge -->

---

## Build with us

<a id="build-with-us"></a>

Useful work: the Gmail loop, storage adapters, tests that fail when restore would fail.

```bash
bun test          # TODO
bun run lint      # TODO
```

Open an issue before large design changes.

<!-- TODO: CONTRIBUTING.md -->

<p align="center">
  <img src="logo.png" width="72" alt="" />
</p>

<p align="center"><strong>Burrow</strong><br/>Your SaaS data. Your copy.</p>
