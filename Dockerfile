# syntax=docker/dockerfile:1

# ---- Build stage ----
# CGO is required by github.com/mattn/go-sqlite3, so we build with a C toolchain
# and run on a glibc-based image (not scratch/distroless-static).
FROM golang:1.20-bookworm AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
        -o /out/mindbalancer ./cmd/mindbalancer && \
    go build -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
        -o /out/mindsql ./cmd/mindsql

# ---- Runtime stage ----
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    useradd --system --uid 10001 --home /var/lib/mindbalancer mindbalancer && \
    mkdir -p /var/lib/mindbalancer /etc/mindbalancer && \
    chown -R mindbalancer:mindbalancer /var/lib/mindbalancer

COPY --from=builder /out/mindbalancer /usr/local/bin/mindbalancer
COPY --from=builder /out/mindsql /usr/local/bin/mindsql
COPY configs/mindbalancer.example.cnf /etc/mindbalancer/mindbalancer.cnf

USER mindbalancer
WORKDIR /var/lib/mindbalancer

# admin MySQL (6032), admin HTTP (6033), proxy (6034), metrics (9090)
EXPOSE 6032 6033 6034 9090

ENTRYPOINT ["mindbalancer"]
CMD ["-config", "/etc/mindbalancer/mindbalancer.cnf"]
