FROM rust:1.85-slim-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential curl pkg-config libssl-dev && \
    rm -rf /var/lib/apt/lists/*

# Install Noir
RUN curl -L https://raw.githubusercontent.com/noir-lang/noirup/refs/heads/main/install | bash
ENV PATH="/root/.nargo/bin:${PATH}"
RUN noirup

WORKDIR /build
COPY Cargo.toml Cargo.lock ./
COPY src/ ./src/
COPY circuits/ ./circuits/
COPY risc0/ ./risc0/

RUN cargo build --release --bin zk-prover

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates libssl3 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/target/release/zk-prover /zk-prover
COPY --from=builder /build/circuits /circuits
COPY --from=builder /build/risc0 /risc0

EXPOSE 3002

CMD ["/zk-prover"]
