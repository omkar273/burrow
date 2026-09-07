<p align="center">
  <img src="logo.png" alt="Burrow — a protected place for your data" width="220" />
</p>

<h1 align="center">Burrow</h1>

<p align="center"><strong>Your SaaS data. Your storage. Your copy.</strong></p>

<p align="center">
  Burrow keeps an independent, verifiable copy of the data your business<br />
  depends on — in storage you control.
</p>

<p align="center">
  <a href="#try-it"><strong>Get started</strong></a>
  &nbsp;·&nbsp;
  <a href="#how-it-works">See how it works</a>
</p>

<p align="center">
  Local-first · Cloud-optional · Open at the core
</p>

<br />

Your email lives in Google.

Your documents live in Drive.

Your conversations live in Slack.

Your code lives in GitHub.

Your business lives across all of them.

**Your copy doesn’t.**

SaaS makes data easy to use. That is not the same as owning it. An account can be deleted, compromised, misconfigured, overwritten, held behind a retention window, or simply unreachable when you need it most.

Keep another copy. Make it yours.

A backup isn’t a backup until you know you can recover from it.

---

## What Burrow is

Burrow is a local-first, cloud-optional **data ownership and recovery layer** for SaaS.

It copies the records that matter out of the tools you already use, puts them in storage you choose, checks that the copy can actually come back, and gives you one place to search, restore, and leave with everything.

Backup is the mechanism. **Ownership, recoverability, and portability are the product.**

Burrow gives your data another home — quietly, independently, and without becoming the next place you’re trapped.

```text
             SaaS
              │
              ▼
        ┌─────────────┐
        │   Burrow    │
        │             │
        │  Sync       │
        │  Verify     │
        │  Index      │
        │  Recover    │
        └──────┬──────┘
               │
       ┌───────┴────────┐
       ▼                ▼
   Your storage     Another replica
```

Run it on your machine. Or let a hosted runtime do the watching. Convenience is optional. The copy is not.

---

## What you get

**Keep a copy.** Continuously pull important data from the SaaS tools your business already depends on.

**Put it where you want.** A folder on disk. S3-compatible storage. R2. NAS as adapters land. Your bucket, your disk — not ours.

**Know that it actually worked.** “Sync completed” is not the finish line. Burrow cares whether objects are intact, consistent, and restorable.

**Find what you need.** Search the independent copy instead of hunting through five products when something goes missing.

**Get it back.** Restore to the original account, another account, a local download, or a portable archive.

**Leave whenever you want.** Export in a form that still makes sense if Burrow isn’t there tomorrow.

That last one is load-bearing. Burrow should never become the next place you’re locked into.

---

## How it works

The product is a loop, not a dashboard widget.

```text
CONNECT → COPY → VERIFY → SEARCH → RECOVER → EXPORT
```

| | |
| --- | --- |
| **Connect** | Link a source. Gmail is the first one we are building. |
| **Copy** | Ingest into a canonical archive, then onto storage you control. |
| **Verify** | Checksums, completeness, consistency — not just a successful job. |
| **Search** | Find a message, a file, a version in *your* copy. |
| **Recover** | Put it back where it belongs, or somewhere else. |
| **Export** | Walk away with the archive. Secrets stay out of the plaintext dump. |

The loop we are aiming at, in one sitting:

Connect Gmail → choose your storage → let it copy → delete a message on purpose → restore it from Burrow → confirm it matches.

If that last step has never worked, you are not protected. You have a log line.

---

## Your data should remain yours

```text
Typical SaaS                         Burrow

Your data                            Your data
    ↓                                    ↓
Provider                             Burrow
    ↓                                    ↓
Provider storage                     Your storage
    ↓                                    ↓
Provider recovery                    Your independent copy
```

Burrow is in the path. It is not supposed to *be* the destination.

Storage is a replaceable layer: local filesystem and S3-compatible APIs first. Docker packages the process; it is not the archive. A real export reconstructs logical state — manifest, schema version, snapshot, objects, checksums — on another machine, without the original volume.

> Everything required to own and recover your data is open.

---

## A green sync icon isn’t enough

A job that finished is an event. Recoverability is a **property**.

Burrow is being designed so “healthy” means more than a checkmark:

- the objects you think you have are actually there
- metadata still makes sense
- checksums match
- storage isn’t quietly corrupt
- a restore has been proven, not assumed

**Burrow treats recoverability as a first-class product property.**

Ingestion is at-least-once. Writes are idempotent. Provider push events are triggers, not the source of truth. Reconciliation still has to close the gaps. Boring on purpose.

---

## Where it sits

```text
              ┌─────────────┐
              │ SaaS sources│
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │   Burrow    │
              └──────┬──────┘
                     │
          ┌──────────┼──────────┐
          ▼          ▼          ▼
       Local FS     S3 / R2      NAS
          │          │          │
          └──────────┼──────────┘
                     ▼
              Your independent
                     copy
```

Self-hosting is a first-class design goal, not a community afterthought. A managed cloud, when it exists, is the same idea with the lights left on — you still bring the storage if you want to.

---

## Open core

The software you need to **own, copy, verify, search, restore, and export** is licensed under **[AGPLv3](https://www.gnu.org/licenses/agpl-3.0.html)**.

AGPL is copyleft. It is not “non-commercial,” and it does not forbid forks. If you run a modified Burrow as a network service, you owe people the source.

Organizational-scale operation can be commercial later: SSO, SCIM, tighter RBAC, retention and legal hold, central admin, MSP, support, a managed cloud. That list is **direction**, not a catalog. You should not need a paid edition to keep a copy of your mail.

```text
Burrow Community          Burrow Cloud              Burrow Enterprise
Open source               Managed infrastructure    Controls, compliance
Self-hosted               Pay for convenience       Support
```

This repository is the project. It is not a pricing page.

**Burrow**, the logo, and the mascot are trademarks. Fork the code; don’t pretend to *be* Burrow.

---

## Where we are

The idea is sharp. The first recovery loop is **not shipped**. There is no production release to install today.

| | |
| --- | --- |
| **Available** | This repository, the product thesis, the architecture. |
| **In progress** | Local runtime, Gmail + attachments, verify, search, restore, portable export, web UI. |
| **Planned** | S3-compatible replica, more sources (Drive, Microsoft 365, Slack, GitHub, Notion — **none of these exist yet**), customer-held keys, hosted cloud, enterprise controls. |

No connector is real until it restores. Roadmap names are not features.

---

## Try it

<a id="try-it"></a>

When the local runtime lands, trying Burrow should feel like: install (or run a container), connect one source, point at storage you already own, wait, then recover something on purpose.

Until then, this is the **intended** developer entry — not a verified quickstart. The tree is not runnable yet.

```bash
git clone https://github.com/<org>/burrow.git   # TODO: canonical remote
cd burrow
bun install                                     # TODO: workspace install
bun dev                                         # TODO: dev script
```

<!-- TODO: docs, Docker Compose, CI badge -->

---

## Under the hood

<a id="how-it-works"></a>

Burrow is a **monorepo** with a **modular core** and several ways to start the same engine: local agent, API, web UI, cloud worker, CLI.

It is not microservices. Packages exist so a connector cannot swallow storage, and so recovery does not depend on Gmail remaining in the room.

```text
                    Core engine
                         │
              ┌──────────┴──────────┐
              │                     │
        Local runtime          Cloud runtime
        SQLite + local FS      Postgres + queue
        your machine           managed infrastructure
```

Same objects. Same recovery semantics. Different placement.

Connectors read providers. Canonical objects are the archive. Storage adapters hold bytes. State and a job queue keep work durable. Verification and search sit on top. Runtime is just where that runs.

The current implementation direction is TypeScript, Bun for local development and the first local process, SQLite at home, Postgres when hosted, S3-compatible object storage. Those are choices. They are not the identity of the product.

Push is a hint. Reconciliation is the backstop. A restore test is the acceptance criteria.

### Repository map

This is the layout we are implementing. Folders will move. The split — apps run Burrow, packages *are* Burrow — should not.

```text
apps/        web, api, agent, worker, cli
packages/    domain, core, connectors, storage, state, queue,
             archive, recovery, search, crypto, observability,
             contracts, ui
```

### Develop

```bash
bun test          # TODO
bun run lint      # TODO
```

Keep provider code in connectors, bytes in storage adapters, restore independent of both.

Useful work right now: the Gmail loop, storage adapters, and tests that go red when restore would fail. Open an issue before large design swings.

<!-- TODO: CONTRIBUTING.md -->

---

## License

Core: **[GNU Affero General Public License v3.0](https://www.gnu.org/licenses/agpl-3.0.html)**.

<!-- TODO: add LICENSE to the repo -->

Everything required to own and recover your data stays in that core.

---

Questions, ideas, and the first connectors: GitHub issues.

<!-- TODO: security contact, community space -->
