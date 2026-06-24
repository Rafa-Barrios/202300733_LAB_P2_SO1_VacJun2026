# Stage 1 — compilación
FROM rust:1.96-slim AS builder

WORKDIR /app

# Instalamos dependencias del sistema necesarias para compilar
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Copiamos archivos del proyecto
COPY Cargo.toml Cargo.lock* ./
COPY src ./src

# Compilamos en modo release
RUN cargo build --release

# Stage 2 — imagen final liviana
FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

# Copiamos solo el binario compilado
COPY --from=builder /app/target/release/rust-api .

EXPOSE 8080

ENV PORT=8080
ENV RUST_LOG=info

CMD ["./rust-api"]