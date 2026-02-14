FROM golang:1.24-bookworm AS builder

WORKDIR /src

# Install SQLite dev headers for CGO
RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-dev gcc libc6-dev && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /bin/sync-server .

# --- runtime ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-0 ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/sync-server /usr/local/bin/sync-server

RUN mkdir -p /app/data

ENV HOST=0.0.0.0
ENV PORT=8080
ENV DATA_DIR=/app/data
ENV STORE_BACKEND=json

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["sync-server"]
