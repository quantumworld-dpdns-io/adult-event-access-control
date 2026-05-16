# aeV Architecture

> Privacy-respecting event access with zero-knowledge age proofs and anti-transfer credentials.

## System Overview

```
                                    ┌──────────────────────────────────────┐
                                    │           Users / Attendees         │
                                    │  (Browser, Mobile Wallet, QR Scan)  │
                                    └────────┬────────────┬───────────────┘
                                             │            │
                                  ┌──────────▼──┐  ┌──────▼──────────┐
                                  │  World ID    │  │  OAuth / SIWE   │
                                  │  (Orb Verify)│  │  (Fallback)     │
                                  └──────┬───────┘  └──────┬──────────┘
                                         │                 │
                                    ┌────▼─────────────────▼───────────┐
                                    │        Next.js Frontend         │
                                    │   (Event discovery, ticket UI,  │
                                    │    check-in scan, proof request)│
                                    └───────────────┬─────────────────┘
                                                    │ HTTP (REST)
                                    ┌───────────────▼─────────────────┐
                                    │       Go API Server             │
                                    │  ┌──────────────────────────┐   │
                                    │  │  Auth (multi-provider)    │   │
                                    │  │  Events CRUD              │   │
                                    │  │  Tickets + QR             │   │
                                    │  │  Check-in                 │   │
                                    │  │  Admin / Stats            │   │
                                    │  └──────────┬───────────────┘   │
                                    │             │ Wasmtime           │
                                    │  ┌──────────▼───────────────┐   │
                                    │  │  ZK Verifier (embedded)   │   │
                                    │  │  Barretenberg via Wasm    │   │
                                    │  └──────────────────────────┘   │
                                    └──────┬────────────────┬─────────┘
                                           │ gRPC/REST      │ SQL
                              ┌────────────▼───┐   ┌───────▼──────────┐
                              │  ZK Prover     │   │   PostgreSQL     │
                              │  (Rust)        │   │                  │
                              │  ┌──────────┐  │   │ ┌────────────┐  │
                              │  │ Noir      │  │   │ │ Users      │  │
                              │  │ Age Proof  │──┼───┼>│ Events     │  │
                              │  └──────────┘  │   │ │ Tickets    │  │
                              │  ┌──────────┐  │   │ │ Check-ins  │  │
                              │  │ RISC Zero │  │   │ │ Transfers  │  │
                              │  │ Anti-Xfer │  │   │ │ Proofs     │  │
                              │  └──────────┘  │   │ └────────────┘  │
                              └────────────────┘   └──────────────────┘
                                                         │ SQL
                                                    ┌────▼──────────────┐
                                                    │  Julia Analytics  │
                                                    │  ┌────────────┐   │
                                                    │  │ Fraud      │   │
                                                    │  │ Detection  │   │
                                                    │  │ (Isolation │   │
                                                    │  │  Forest)   │   │
                                                    │  │ Sybil Score│   │
                                                    │  └────────────┘   │
                                                    └───────────────────┘
                                    ┌───────────────────────────────────┐
                                    │     Smart Contracts (EVM)         │
                                    │  ┌────────────┐ ┌──────────────┐ │
                                    │  │AgeVerifier │ │SoulboundTicket│ │
                                    │  │ (Noir ZK)  │ │ (ERC-5192)   │ │
                                    │  └────────────┘ └──────────────┘ │
                                    └───────────────────────────────────┘
```

## Component Descriptions

### 1. Next.js Frontend

The frontend is a Next.js 15 application providing:

- **Event discovery** — Browse published events with filtering
- **Multi-auth UI** — World ID Orb Integration, OAuth buttons, SIWE modal
- **Ticket management** — View owned tickets, QR codes for check-in
- **ZK proof flow** — Triggers age proof generation before ticket issuance
- **Check-in scanning** — QR code scanner for venue entry

Technology choices:

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Framework | Next.js 15 (App Router) | SSR, RSC, file-based routing |
| Auth UI | `@worldcoin/idkit`, `siwe`, `ethers` | First-party World ID + wallet support |
| API client | Native `fetch` via `src/lib/api.ts` | No extra dependencies |

### 2. Go API Server

The backend is a single Go binary serving REST endpoints. It embeds a Wasmtime-powered Barretenberg verifier for on-the-fly ZK proof verification.

**Modules:**

| Package | Responsibility | Routes |
|---------|---------------|--------|
| `internal/auth` | Multi-provider authentication | `POST /api/auth/{oauth,siwe,world-id}`, `GET /api/auth/me` |
| `internal/handlers/events.go` | Event CRUD | `GET/POST /api/events`, `GET/PUT/DELETE /api/events/{id}` |
| `internal/handlers/tickets.go` | Ticket management | `POST /api/events/{eid}/tickets`, `GET /api/tickets/{id}`, `GET /api/my/tickets` |
| `internal/handlers/checkin.go` | Entry verification | `POST /api/checkin`, `GET /api/events/{eid}/attendees` |
| `internal/handlers/zk_prover.go` | ZK proof proxy | `POST /api/zk/generate-proof`, `POST /api/zk/verify` |
| `internal/handlers/admin.go` | Admin dashboard | `GET /api/admin/stats`, `GET /api/admin/events/{id}/analytics` |
| `internal/middleware` | Request pipeline | Auth (JWT), CORS, request logging |

Technology choices:

| Concern | Choice | Rationale |
|---------|--------|-----------|
| HTTP router | `net/http` (Go 1.22+) | Built-in pattern matching, no external dependency |
| Database | `database/sql` + `lib/pq` | Standard library, minimal abstraction |
| Auth | `golang-jwt/jwt/v5` | Industry standard for JWT |
| ZK verification | Wasmtime (planned) | Embed Barretenberg verifier without subprocess |

### 3. ZK Prover (Rust)

An Actix-web HTTP service that manages ZK proof generation. It wraps two proving backends:

**Noir (Primary) — Age Proof Circuit**

```
Private inputs:  birth_year
Public inputs:   current_year, min_age
Output:          1 if age >= min_age, else proof generation fails
```

The circuit (`src/zk-prover/circuits/src/main.nr`) constrains that `current_year - birth_year >= min_age`. The proof is generated via `nargo` + Barretenberg.

**RISC Zero (Fallback) — Anti-Transfer Credential**

The zkVM guest (`src/zk-prover/risc0/guest/src/main.rs`) proves:
1. The credential is bound to a specific World ID nullifier (anti-Sybil)
2. The credential has not exceeded transfer limits
3. The age proof commitment is valid

Technology choices:

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Runtime | Actix-web 4 | Async, high performance |
| Proving | Noir + Barretenberg | Domain-specific, Solidity verifier export |
| Fallback | RISC Zero zkVM | General computation proofs |
| Verification | Wasmtime (Go side) | Embed verifier without subprocess |

### 4. PostgreSQL Database

The schema uses UUID primary keys, foreign key relationships, and materialized views for analytics.

**Key tables:**

| Table | Purpose |
|-------|---------|
| `users` | User profiles (agnostic to auth provider) |
| `auth_providers` | Links users to OAuth/SIWE/World ID identities |
| `events` | Event details, capacity, min age |
| `tickets` | Off-chain ticket records with ZK commitment |
| `check_in_log` | Audit trail for venue entry |
| `transfer_log` | Ticket transfer history for fraud analysis |
| `world_id_proofs` | World ID verification audit |

**Materialized view:** `ticket_transfer_stats` — pre-computed transfer ratios per event for fraud detection.

### 5. Julia Analytics Service

A lightweight HTTP server (via `HTTP.jl`) providing:

- **`GET /api/analytics/overview`** — Aggregate stats + anomaly detection
- **`GET /api/analytics/sybil/:user_id`** — Per-user Sybil resistance score
- **`GET /api/analytics/anomalies`** — Transfer ratio outliers, rapid check-in detection

The `FraudDetection` module implements an Isolation Forest for unsupervised anomaly detection on ticket transfer patterns.

Technology choices:

| Concern | Choice | Rationale |
|---------|--------|-----------|
| HTTP | `HTTP.jl` | Built-in, lightweight |
| Data | `DataFrames.jl` | Tabular processing |
| ML | Custom Isolation Forest | Lightweight, no external ML deps |
| DB | `LibPQ.jl` | Native PostgreSQL binding |

### 6. Smart Contracts (Solidity)

Foundry-managed EVM contracts:

- **`AgeVerifier.sol`** — On-chain verifier for Noir age proofs. In production, the verification key is generated by the Barretenberg `bb contract` command.
- **`SoulboundTicket.sol`** — ERC-5192 non-transferable NFT for on-chain ticket representation. Minting requires a valid ZK age proof.

## Data Flow Diagrams

### Authentication Flow

```
User                    Frontend                Go API             Auth Provider
 │                        │                       │                     │
 │  Choose auth method    │                       │                     │
 │───────────────────────>│                       │                     │
 │                        │                       │                     │
 │  ─── OAuth ───         │                       │                     │
 │  Redirect to provider  │                       │                     │
 │<───────────────────────│                       │                     │
 │─────────────────────────────────────────────────────────────────────>│
 │  Auth code callback    │                       │                     │
 │<─────────────────────────────────────────────────────────────────────│
 │───────────────────────>│  POST /api/auth/oauth  │                     │
 │                        │  {provider, code}      │                     │
 │                        │───────────────────────>│                     │
 │                        │                       │  Verify code         │
 │                        │                       │─────────────────────>│
 │                        │                       │  User info           │
 │                        │                       │<─────────────────────│
 │                        │                       │                     │
 │                        │                       │  Upsert user + JWT  │
 │                        │<── {token, user_id} ──│                     │
 │<── store token ────────│                       │                     │
 │                        │                       │                     │
 │  ─── SIWE ───          │                       │                     │
 │  Sign message (EIP-4361)│                      │                     │
 │───────────────────────>│  POST /api/auth/siwe  │                     │
 │                        │  {message, signature} │                     │
 │                        │───────────────────────>│                     │
 │                        │                       │  Recover address     │
 │                        │                       │  Upsert user + JWT  │
 │                        │<── {token, user_id} ──│                     │
 │                        │                       │                     │
 │  ─── World ID ───     │                       │                     │
 │  Orb verification      │                       │                     │
 │───────────────────────>│                       │                     │
 │                        │  POST /api/auth/world-id                    │
 │                        │  {nullifier_hash, proof}                    │
 │                        │───────────────────────>│                     │
 │                        │                       │  Verify with World  │
 │                        │                       │  ID cloud API       │
 │                        │                       │─────────────────────>│
 │                        │                       │<─────────────────────│
 │                        │                       │  Upsert user + JWT  │
 │                        │<── {token, user_id} ──│                     │
```

### ZK Age Proof Flow

```
User                    Frontend                Go API              ZK Prover (Rust)
 │                        │                       │                       │
 │  Enter birthdate       │                       │                       │
 │───────────────────────>│                       │                       │
 │                        │  POST /api/zk/generate-age-proof              │
 │                        │  {birthdate, min_age} │                       │
 │                        │───────────────────────>│                       │
 │                        │                       │  POST /api/prove-age  │
 │                        │                       │  (proxies to Rust)    │
 │                        │                       │──────────────────────>│
 │                        │                       │                       │──► Noir circuit
 │                        │                       │                       │    main(birth_year,
 │                        │                       │  ◄── proof ──────────│    current_year,
 │                        │                       │                       │    min_age)
 │                        │  ◄── proof, public ───│                       │
 │                        │       outputs         │                       │
 │  ◄── proof commitment ─│                       │                       │
 │                        │                       │                       │
 │  Issue ticket          │                       │                       │
 │───────────────────────>│  POST /api/events/{id}/tickets               │
 │                        │  {zk_proof_commitment, ticket_type}           │
 │                        │───────────────────────>│                       │
 │                        │                       │  Verify proof         │
 │                        │                       │  (Wasmtime-embedded   │
 │                        │                       │   Barretenberg)       │
 │                        │                       │                       │
 │                        │                       │  INSERT ticket (DB)   │
 │                        │  ◄── ticket + QR ─────│                       │
 │  ◄── show QR code ────│                       │                       │
```

### Check-in Flow

```
Attendee              Venue Staff / Scanner      Go API               DB
 │                        │                       │                    │
 │  Show QR code          │                       │                    │
 │───────────────────────>│                       │                    │
 │                        │  POST /api/checkin    │                    │
 │                        │  {qr_secret, method}  │                    │
 │                        │───────────────────────>│                    │
 │                        │                       │  Lookup ticket     │
 │                        │                       │───────────────────>│
 │                        │                       │<── ticket data ────│
 │                        │                       │                    │
 │                        │                       │  Verify status     │
 │                        │                       │  (not checked in)  │
 │                        │                       │                    │
 │                        │                       │  UPDATE status     │
 │                        │                       │  = 'checked_in'    │
 │                        │                       │───────────────────>│
 │                        │                       │  INSERT check_in_log│
 │                        │                       │───────────────────>│
 │                        │                       │                    │
 │                        │  ◄── verified ───────│                    │
 │  ◄── entry granted ───│                       │                    │
```

### Ticket Transfer Flow

```
Sender                Go API                    DB              RISC Zero ZKVM
 │                       │                       │                    │
 │  Request transfer     │                       │                    │
 │──────────────────────>│                       │                    │
 │  POST /api/tickets/{id}/transfer              │                    │
 │  {to_user_id,         │                       │                    │
 │   zk_proof_from,      │                       │                    │
 │   zk_proof_to}        │                       │                    │
 │                       │                       │                    │
 │                       │  Check sender owns    │                    │
 │                       │──────────────────────>│                    │
 │                       │<── confirmed ─────────│                    │
 │                       │                       │                    │
 │                       │  Verify anti-transfer │                    │
 │                       │  proof (zkVM guest)   │                    │
 │                       │──────────────────────────────────────────>│
 │                       │<── is_verified ───────│────────────────────│
 │                       │                       │                    │
 │                       │  INSERT transfer_log  │                    │
 │                       │──────────────────────>│                    │
 │                       │  UPDATE owner_id      │                    │
 │                       │──────────────────────>│                    │
 │  ◄── transferred ────│                       │                    │
```

## Sequence Diagram: Full Auth + ZK Proof Flow

```
┌─────────┐   ┌──────────┐   ┌──────────┐   ┌───────────┐   ┌──────┐   ┌──────────┐
│ Browser │   │ Frontend │   │ Go API   │   │ ZK Prover│   │ DB   │   │World ID  │
│         │   │ (Next.js)│   │ (Server) │   │ (Rust)   │   │      │   │(Cloud)   │
└────┬────┘   └────┬─────┘   └────┬─────┘   └─────┬─────┘   └──┬───┘   └────┬─────┘
     │             │              │                │           │            │
     │  1. Auth with World ID    │                │           │            │
     │────────────>│             │                │           │            │
     │             │  2. POST /api/auth/world-id  │           │            │
     │             │  {nullifier_hash, proof}     │           │            │
     │             │────────────>│                │           │            │
     │             │             │  3. Verify proof───────────│───────────>│
     │             │             │<── success ────────────────│────────────│
     │             │             │                │           │            │
     │             │             │  4. Upsert user            │            │
     │             │             │───────────────│───────────>│            │
     │             │<── {jwt} ───│                │           │            │
     │<── store ───│             │                │           │            │
     │             │             │                │           │            │
     │  5. Browse events        │                │           │            │
     │────────────>│             │                │           │            │
     │             │  6. GET /api/events          │           │            │
     │             │────────────>│                │           │            │
     │             │             │  7. Query events            │            │
     │             │             │───────────────│───────────>│            │
     │             │<── events ──│<──────────────│────────────│            │
     │<── list ────│             │                │           │            │
     │             │             │                │           │            │
     │  8. Trigger age proof   │                │           │            │
     │────────────>│             │                │           │            │
     │             │  9. POST /api/zk/generate-age-proof     │            │
     │             │  {birthdate: "1990-01-01", min_age: 18} │            │
     │             │────────────>│                │           │            │
     │             │             │ 10. /api/prove-age────────>│            │
     │             │             │                │           │            │
     │             │             │                │ 11. Run Noir circuit  │
     │             │             │                │  main(birth_year,    │
     │             │             │                │       current_year,  │
     │             │             │                │       min_age)       │
     │             │             │                │  => proof            │
     │             │             │<── proof ──────│           │            │
     │             │<── proof ───│                │           │            │
     │<── proof ───│             │                │           │            │
     │             │             │                │           │            │
     │ 12. Issue ticket         │                │           │            │
     │────────────>│             │                │           │            │
     │             │ 13. POST /api/events/{id}/tickets      │            │
     │             │ {zk_proof_commitment, ticket_type}     │            │
     │             │────────────>│                │           │            │
     │             │             │ 14. Verify proof (Wasmtime)           │
     │             │             │    (embedded Barretenberg)             │
     │             │             │                │           │            │
     │             │             │ 15. INSERT ticket          │            │
     │             │             │───────────────│───────────>│            │
     │             │<── {ticket} │                │           │            │
     │<── show QR ─│             │                │           │            │
     │             │             │                │           │            │
     │ 16. Check-in at venue   │                │           │            │
     │────────────>│             │                │           │            │
     │             │ 17. POST /api/checkin       │           │            │
     │             │ {qr_secret, method: "qr"}  │           │            │
     │             │────────────>│                │           │            │
     │             │             │ 18. Lookup ticket          │            │
     │             │             │───────────────│───────────>│            │
     │             │             │ 19. Verify status + update│            │
     │             │             │───────────────│───────────>│            │
     │             │             │ 20. INSERT check_in_log   │            │
     │             │             │───────────────│───────────>│            │
     │             │<── verified │                │           │            │
     │<── entry ───│             │                │           │            │
```

## Security Considerations

### Zero-Knowledge Proofs

- **Age proofs** reveal only whether the user is over the minimum age — never the exact birthdate
- **Anti-transfer proofs** prove credential binding without revealing the World ID nullifier
- **Wasmtime sandbox** isolates the Barretenberg verifier from the Go API process
- **On-chain verification** allows trustless ticket minting via SoulboundTicket

### Authentication

- **World ID** provides proof of personhood (anti-Sybil at registration)
- **JWT tokens** expire after 24 hours; refresh mechanism is per-session
- **OAuth/SIWE** are fallback auth modes with reduced Sybil resistance

### Anti-Fraud

- **Nullifier deduplication** prevents single World ID identity from holding multiple tickets to the same event
- **Transfer ratio monitoring** flags events with abnormally high transfer rates
- **Rapid check-in detection** identifies automated scanning
- **Soulbound tickets (ERC-5192)** prevent secondary market resale
