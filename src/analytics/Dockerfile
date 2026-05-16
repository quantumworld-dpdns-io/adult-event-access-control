FROM julia:1.11-bookworm

WORKDIR /app

COPY Project.toml Manifest.toml* ./
RUN julia -e 'import Pkg; Pkg.instantiate()' 2>/dev/null || true

COPY src/ ./src/

EXPOSE 8090

CMD ["julia", "src/server.jl"]
