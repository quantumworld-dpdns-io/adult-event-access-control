# aeV — Adult Event Access Control

> Privacy-respecting event access with zero-knowledge age proofs and anti-transfer credentials.

```
                                             ┌─────────────┐
                                             │ Next.js     │
                                             │ Frontend    │
                                             └──────┬──────┘
                                                    │ HTTP
                                               ┌────▼──────┐
                                               │ Go API    │──Wasmtime──┐
                                               │ Server    │            │
                                               └────┬──────┘            │
                                          gRPC/REST │                  │
                                               ┌────▼──────┐    ┌──────▼──────┐
                                               │ ZK Prover │    │ Wasmtime    │
                                               │ (Rust)    │    │ Verifier    │
                                               │ Noir+RISC0│    │ (embedded)  │
                                               └────┬──────┘    └─────────────┘
                                                    │ SQL
                                               ┌────▼──────┐    ┌─────────────┐
                                               │ Postgres  │◄───│ Julia       │
                                               │           │    │ Analytics   │
                                               └───────────┘    └─────────────┘
```

## Tech Stack

| Layer | Technology | Role |
|-------|-----------|------|
| Frontend | **Next.js** | Event discovery, ticket purchase, check-in UI |
| API | **Go** | REST API, multi-auth, Wasmtime-embedded ZK verification |
| ZK Prover | **Rust + Noir + RISC Zero** | Zero-knowledge proof generation |
| Analytics | **Julia** | ML fraud detection, DuckDB reporting |
| Database | **PostgreSQL** | Events, tickets, proofs, audit log |
| Smart Contracts | **Solidity** | On-chain verifier + Soulbound Tickets |
| Cache | **Redis** | Real-time seat maps, waitlist, rate limiting |
| Vector Search | **Qdrant + pgvector** | Semantic search for events, attendee matching |
| Edge Runtime | **Fermyon Spin** | Wasm-compiled check-in microservice |
| AI Agents | **MCP Server** | LangGraph/CrewAI integration via Model Context Protocol |
| Post-Quantum | **ML-KEM/ML-DSA** | Future-proof identity commitments |

## Authentication (Multi-Auth)

- **World ID (Primary)** — Proof of personhood via [World.org](https://world.org) Orb verification
- **OAuth** — Google, GitHub (fallback)
- **SIWE** — Sign-In with Ethereum (EIP-4361) (fallback)

## Zero-Knowledge Proof System

This project uses **two complementary ZK systems**:

### 1. Noir (Primary — Age Proof)

[Noir](https://noir-lang.org) is a domain-specific language for zero-knowledge proofs. It compiles to ACIR and uses the Barretenberg proving backend.

**Circuit: Age Verification**
```
Private inputs:  birth_year
Public inputs:   min_age, is_over_age
Constraint:      current_year - birth_year >= min_age
```

### 2. RISC Zero (Fallback — Anti-Transfer Credentials)

[RISC Zero](https://risczero.com) is a zkVM that proves correct execution of arbitrary Rust programs using zk-STARKs with recursive composition.

**Guest Program:** Binds credentials to a World ID nullifier within transfer limits.

```
Private inputs:  user_nullifier_hash, credential_commitment
Public outputs:  is_verified (journal)
```

### Proof Flow

```
         User                    Go API                  ZK Prover
          │                        │                        │
          │── age_proof(params)───►│── prove("noir",...)───►│
          │                        │◄─── proof + witness ──│
          │◄── ticket + qr_code ──│                        │
          │                        │                        │
          │── checkin(qr_code)───►│── verify(proof) ──────►│
          │                        │   (Wasmtime embedded)  │
          │◄── verified ──────────│◄─── verified ──────────│
```

## Quick Start

```bash
# Prerequisites: Docker, Docker Compose

# 1. Start all services
docker compose -f deploy/docker-compose.yml up -d

# 2. Frontend (standalone)
cd src/frontend
npm install
npm run dev

# 3. Open http://localhost:3000
```

## Smart Contracts

- **AgeVerifier.sol** — Verifies Noir age proofs on-chain (generated from Barretenberg)
- **SoulboundTicket.sol** — ERC-5192 non-transferable event ticket NFT
- **DeployScript.s.sol** — Foundry deployment script for testnets

Deploy via Foundry:
```bash
cd src/zk-prover/solidity
forge script DeployScript --rpc-url <testnet> --broadcast
```

## Expansion Priority

### Phase P0 — Foundation (Deployment + Infra)
| # | Task | Files |
|---|------|-------|
| 1 | Choreo deployment config (endpoints + workload) | `.choreo/endpoints.yaml`, `.choreo/workload.yaml` |
| 2 | CI/CD deploy pipeline to Choreo | `.github/workflows/deploy.yml` |
| 3 | Secrets management (Choreo + alwaysdata) | `deploy/.env.example`, `deploy/docker-compose.yml` |

### Phase P1 — Infrastructure (Caching + Search)
| # | Task | Files |
|---|------|-------|
| 1 | Redis cache layer (seat maps, waitlist, rate limiting) | `src/backend/internal/cache/redis.go` |
| 2 | Qdrant vector DB client + semantic event search | `src/backend/internal/search/qdrant.go`, `internal/handlers/search.go` |
| 3 | pgvector migration for PostgreSQL vector search | `src/db/migrations/006_pgvector.sql` |

### Phase P2a — Market & Pricing
| # | Task | Files |
|---|------|-------|
| 1 | Dynamic surge pricing engine | `src/backend/internal/handlers/pricing.go`, `src/analytics/src/pricing.jl` |
| 2 | Secondary resale marketplace with ZK transfer proofs | `src/backend/internal/handlers/resale.go`, `src/db/migrations/003_marketplace.sql` |

### Phase P2b — Real-time Venue Features
| # | Task | Files |
|---|------|-------|
| 1 | WebSocket pub/sub for live capacity heatmap | `src/backend/internal/handlers/ws.go`, `internal/handlers/heatmap.go` |
| 2 | Emergency alerts (LISTEN/NOTIFY → WS broadcast) | `src/backend/internal/handlers/alerts.go` |
| 3 | Venue live dashboard | `src/frontend/src/app/venue/[id]/page.tsx` |

### Phase P2c — AI/ML Features
| # | Task | Files |
|---|------|-------|
| 1 | Event-attendee matchmaker (Julia embeddings) | `src/analytics/src/matchmaker.jl`, `internal/handlers/recommend.go` |
| 2 | No-show prediction model | `src/analytics/src/noshow.jl` |
| 3 | ML feature pipeline DB migrations | `src/db/migrations/005_ml.sql` |

### Phase P3a — Edge Wasm (Fermyon Spin)
| # | Task | Files |
|---|------|-------|
| 1 | Spin check-in microservice (Wasm-compiled) | `src/edge-checkin/` (Spin project with Rust SDK) |
| 2 | Edge-optimized QR scanner + proof verification | `src/edge-checkin/src/lib.rs` |

### Phase P3b — AI Agent MCP Server
| # | Task | Files |
|---|------|-------|
| 1 | Model Context Protocol server for LangGraph/CrewAI | `src/mcp-server/` (Go MCP project) |
| 2 | Expose all event/ticket tools via MCP | `src/mcp-server/src/tools.rs` |

### Phase P3c — Post-Quantum Cryptography
| # | Task | Files |
|---|------|-------|
| 1 | ML-KEM/ML-DSA identity commitments in Rust prover | `src/zk-prover/src/pqc.rs` |
| 2 | Go PQC verification module (Wasmtime-embedded) | `src/backend/internal/pqc/pqc.go` |
| 3 | PQC-enhanced proving endpoint | `src/backend/internal/handlers/pqc_handler.go` |

## Deployment Architecture

```
  Choreo.dev                          alwaysdata.com
  ┌─────────────────────┐             ┌──────────────┐
  │ Frontend (Next.js)  │             │ PostgreSQL   │
  │  port 3000          │◄───────────►│  port 5432   │
  ├─────────────────────┤  via pg     └──────────────┘
  │ Backend (Go+Rust)   │
  │  port 8080 (API)    │◄─── gRPC ───┐
  │  port 3002 (Prover) │             │
  ├─────────────────────┤             │
  │ Julia Analytics     │─────────────┘
  │  port 8090          │
  ├─────────────────────┤
  │ Redis               │
  │  port 6379          │
  ├─────────────────────┤
  │ Qdrant              │
  │  port 6333          │
  ├─────────────────────┤
  │ MCP Server          │
  │  port 3100          │
  └─────────────────────┘
```

## Project Structure

```
├── src/
│   ├── frontend/            Next.js app
│   ├── backend/             Go API server (Wasmtime verifier)
│   │   ├── cmd/server/      main.go
│   │   ├── internal/
│   │   │   ├── auth/        multi-auth (World ID, OAuth, SIWE)
│   │   │   ├── cache/       Redis client
│   │   │   ├── db/          PostgreSQL connection
│   │   │   ├── handlers/    REST handlers
│   │   │   ├── middleware/  CORS, logging, JWT auth
│   │   │   ├── pqc/         Post-quantum verification
│   │   │   └── search/      Qdrant vector search client
│   │   └── Dockerfile
│   ├── zk-prover/           Rust (Noir circuits + RISC Zero guests)
│   │   ├── src/             main.rs, noir_prover, risc0_prover, pqc
│   │   ├── circuits/        Noir age proof circuit
│   │   ├── risc0/           RISC Zero guest programs
│   │   └── solidity/        Solidity verifier + SoulboundTicket
│   ├── analytics/           Julia ML analytics
│   │   └── src/             server.jl, fraud_detection, pricing, matchmaker, noshow
│   ├── mcp-server/          MCP AI agent server (Rust)
│   ├── edge-checkin/        Fermyon Spin edge check-in (Rust)
│   └── db/                  PostgreSQL migrations
├── deploy/                  Docker Compose + env
├── docs/                    Architecture & runbooks
└── .github/workflows/       CI/CD pipelines
```

## License

MIT — see [LICENSE](LICENSE)
