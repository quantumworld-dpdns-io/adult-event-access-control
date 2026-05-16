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

## Authentication (Multi-Auth)

- **World ID (Primary)** — Proof of personhood via [World.org](https://world.org) Orb verification
- **OAuth** — Google, GitHub (fallback)
- **SIWE** — Sign-In with Ethereum (EIP-4361) (fallback)

## Zero-Knowledge Proof System

This project uses **two complementary ZK systems**:

### 1. Noir (Primary — Age Proof)

[Noir](https://noir-lang.org) is a domain-specific language for zero-knowledge proofs. It compiles to ACIR (Abstract Circuit Intermediate Representation) and uses the Barretenberg proving backend.

**Circuit: Age Verification**

```
Private inputs:  birth_year
Public inputs:   min_age, is_over_age
Constraint:      current_year - birth_year >= min_age
```

The circuit proves an attendee is over a minimum age without revealing their exact birthdate. The proof is verified via:

- **Off-chain**: Wasmtime-embedded Barretenberg verifier in Go
- **On-chain**: Exported Solidity verifier contract for on-chain ticket verification

**Why Noir?** — Purpose-built for constraint circuits, simple Rust-like syntax, Solidity verifier export, mature toolchain (`nargo`).

### 2. RISC Zero (Fallback — Anti-Transfer Credentials)

[RISC Zero](https://risczero.com) is a zero-knowledge virtual machine (zkVM) that proves correct execution of arbitrary Rust programs. It uses zk-STARKs with recursive proof composition.

**Guest Program: Anti-Transfer Credential Binding**

```
Private inputs:  user_nullifier_hash, credential_commitment
Public outputs:  is_verified (journal)
Proof:           The credential is bound to a specific World ID nullifier
                 and has not exceeded transfer limits
```

The zkVM proves that a ticket credential is:
1. Bound to a specific World ID identity (anti-Sybil)
2. Within transfer limits (anti-fraud)
3. Age-verified via the Noir proof commitment

**Why RISC Zero?** — General-purpose computation proofs, ideal for complex multi-condition verification, recursive proofs for batch verification, Rust-native development.

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

## Project Structure

```
├── src/
│   ├── frontend/          Next.js app
│   ├── backend/           Go API server (Wasmtime verifier)
│   ├── zk-prover/         Rust (Noir circuits + RISC Zero guests)
│   ├── analytics/         Julia ML analytics
│   └── db/                PostgreSQL migrations
├── deploy/                Docker Compose + K8s
├── docs/                  Architecture & runbooks
└── .github/workflows/     CI/CD pipelines
```

## License

MIT — see [LICENSE](LICENSE)
