# Contributing to aeV — Adult Event Access Control

## Table of Contents

- [Development Setup](#development-setup)
- [Running Services](#running-services)
- [Running Tests](#running-tests)
- [Smart Contract Development](#smart-contract-development)
- [Project Structure](#project-structure)
- [Coding Guidelines](#coding-guidelines)
- [PR Guidelines](#pr-guidelines)
- [Release Process](#release-process)

## Development Setup

### Prerequisites

| Tool      | Version  | Purpose                       |
|-----------|----------|-------------------------------|
| Go        | 1.26+    | Backend API server            |
| Rust      | 1.85+    | ZK Prover service             |
| Julia     | 1.11+    | Analytics / ML service        |
| Node.js   | 22+      | Next.js frontend              |
| Docker    | 24+      | Containerized services        |
| Docker Compose | 2.24+ | Multi-service orchestration |
| Foundry   | latest   | Solidity contract development |
| Noir      | 1.0+     | ZK circuit development        |

### Clone & Install

```bash
git clone https://github.com/quantumworld-dpdns-io/adult-event-access-control.git
cd adult-event-access-control

# Backend
cd src/backend
go mod download
cd ../..

# Frontend
cd src/frontend
npm install
cd ../..

# ZK Prover
cd src/zk-prover
cargo build --release --bin zk-prover
cd ../..

# Analytics
cd src/analytics
julia -e 'import Pkg; Pkg.instantiate()'
cd ../..

# Solidity
cd src/zk-prover/solidity
forge install
forge build
cd ../../..
```

### Environment Variables

Copy the example env file and fill in your secrets:

```bash
cp deploy/.env.example .env
```

Required overrides for local dev:
- `JWT_SECRET` — generate with `openssl rand -hex 32`
- `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` — for OAuth testing
- `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` — for OAuth testing
- `DEPLOYER_PRIVATE_KEY` — for contract deployment (never commit)

## Running Services

### Option A: Docker Compose (all services)

```bash
docker compose -f deploy/docker-compose.yml up -d
```

This starts:
- **PostgreSQL** on `:5432`
- **Go Backend** on `:8080`
- **Julia Analytics** on `:8090`
- **Next.js Frontend** on `:3000`

### Option B: Standalone (individual services)

#### PostgreSQL

```bash
docker run -d \
  --name aev-postgres \
  -e POSTGRES_USER=aev \
  -e POSTGRES_PASSWORD=aev_secret \
  -e POSTGRES_DB=aev_events \
  -v $(pwd)/src/db/migrations:/docker-entrypoint-initdb.d \
  -p 5432:5432 \
  postgres:16-alpine
```

#### Go Backend

```bash
cd src/backend
export DB_HOST=localhost DB_PORT=5432 DB_USER=aev DB_PASSWORD=aev_secret DB_NAME=aev_events
export JWT_SECRET=$(openssl rand -hex 32)
go run ./cmd/server
# Listening on :8080
```

#### ZK Prover (Rust)

```bash
cd src/zk-prover
cargo run --release --bin zk-prover
# Listening on :3002
```

#### Julia Analytics

```bash
cd src/analytics
DB_HOST=localhost DB_PORT=5432 DB_USER=aev DB_PASSWORD=aev_secret DB_NAME=aev_events \
  julia src/entrypoint.jl
# Listening on :8090
```

#### Next.js Frontend

```bash
cd src/frontend
NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev
# Listening on :3000
```

## Running Tests

### Go Backend Tests

```bash
cd src/backend
go test ./...
go test -v ./internal/...
```

### Integration Tests

```bash
cd tests/integration
go test -v ./...
```

### E2E Tests

```bash
cd tests/e2e
go test -v ./...
```

### Rust Tests

```bash
cd src/zk-prover
cargo test
```

### Julia Tests

```bash
cd src/analytics
julia -e 'include("test/runtests.jl")'
```

### Foundry / Solidity Tests

```bash
cd src/zk-prover/solidity
forge test
```

## Smart Contract Development

### Compile

```bash
cd src/zk-prover/solidity
forge build
```

### Deploy

```bash
cd src/zk-prover/solidity

# Local Anvil
anvil &
forge script DeployScript --rpc-url http://localhost:8545 --broadcast

# Sepolia testnet
forge script DeployScript \
  --rpc-url https://sepolia.infura.io/v3/$INFURA_KEY \
  --broadcast --verify

# Base Sepolia
forge script DeployScript \
  --rpc-url https://sepolia.base.org \
  --broadcast --verify
```

### Upgrade Verifier

When the Noir circuit changes, regenerate the Solidity verifier:

```bash
cd src/zk-prover
nargo compile --package age_proof
bb contract --output ./solidity/AgeVerifier.sol
forge build
```

## Project Structure

```
├── src/
│   ├── backend/              Go API server
│   │   ├── cmd/server/        Entrypoint
│   │   ├── internal/
│   │   │   ├── auth/          Multi-auth (OAuth, SIWE, World ID)
│   │   │   ├── db/            Database connection
│   │   │   ├── handlers/      HTTP handlers (events, tickets, checkin, ZK, admin)
│   │   │   └── middleware/    Auth, CORS, logging
│   │   └── db/                SQL migrations & seeds
│   ├── frontend/              Next.js app
│   │   └── src/
│   │       ├── app/           Pages (auth, events, tickets)
│   │       ├── components/    UI components
│   │       └── lib/           API client
│   ├── zk-prover/             Rust ZK service
│   │   ├── circuits/          Noir age proof circuit
│   │   ├── risc0/             RISC Zero anti-transfer guest
│   │   ├── solidity/          Foundry project (AgeVerifier, SoulboundTicket)
│   │   └── verifier-wasm/     Wasmtime verifier artifacts
│   ├── analytics/             Julia ML service
│   │   └── src/               Server + fraud detection
│   └── db/                    PostgreSQL migrations
├── deploy/                    Docker Compose + K8s manifests
├── docs/                      Architecture, API spec, runbooks
└── tests/
    ├── unit/                  Unit tests
    ├── integration/           Integration tests (Go httptest)
    └── e2e/                   End-to-end workflow tests
```

## Coding Guidelines

### General

- Follow the conventions of each language's standard style guide
- Mimic existing code patterns within each service
- Never commit secrets, API keys, or `.env` files
- All new features must include tests

### Go

- Use `go fmt` before committing
- Follow standard Go project layout conventions
- Use `database/sql` with prepared statements (no raw string interpolation)
- Handler methods use Go 1.22+ enhanced routing patterns (`METHOD /path/{param}`)
- Export only what's necessary; keep internals in `internal/`

### Rust

- Use `cargo fmt` and `cargo clippy`
- Error types should implement `thiserror::Error`
- Use `actix-web` for HTTP server patterns consistent with existing code

### Solidity

- Target `^0.8.28`
- Use Foundry for compilation and deployment
- Follow existing contract patterns (NatSpec, events, modifiers)
- Keep deploy scripts in `DeployScript.s.sol`

### Julia

- Follow the Julia style guide
- Use `DataFrames` for tabular data processing
- Keep ML models in the `FraudDetection` module

### Commits

- Use conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`
- Keep commits focused on a single logical change
- Write descriptive commit messages explaining *why*, not just *what*

## PR Guidelines

1. **Branch**: Create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Changes**: Make focused, well-tested changes. Keep PRs small.

3. **Commit**: Write clear, descriptive commit messages:
   ```
   feat: add SIWE authentication handler
   
   Implements Sign-In with Ethereum (EIP-4361) verification for
   wallet-based authentication. Parses SIWE messages, recovers
   the signer address, and issues a JWT on success.
   ```

4. **Test**: Ensure all CI checks pass:
   - `go vet ./...` (no warnings)
   - `cargo check` (no errors)
   - `forge build` (compiles)
   - All tests pass

5. **PR Description**: Include:
   - What this PR does
   - Why it's needed (link to issue if applicable)
   - Screenshots for UI changes
   - Breaking changes or migration steps

6. **Review Checklist**:

   - [ ] Code compiles and lints pass
   - [ ] Tests pass (unit + integration + e2e where applicable)
   - [ ] New endpoints have OpenAPI documentation
   - [ ] New env vars are documented in `deploy/.env.example`
   - [ ] No secrets or keys committed
   - [ ] Commit messages are clear and follow conventions

7. **Merge**: Squash-merge into `main` after approval. Delete the feature branch.

## Release Process

1. Create a release branch: `release/vX.Y.Z`
2. Bump versions in `src/backend/go.mod`, `src/zk-prover/Cargo.toml`, `src/frontend/package.json`
3. Run full CI pipeline
4. Tag the release: `git tag vX.Y.Z && git push origin vX.Y.Z`
5. Publish Docker images:
   ```bash
   docker compose -f deploy/docker-compose.yml build
   docker tag aev-backend ghcr.io/org/aev-backend:vX.Y.Z && docker push ...
   ```
6. Deploy to staging, run smoke tests, then promote to production
