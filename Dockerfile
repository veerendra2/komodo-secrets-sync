FROM golang:1.25.6 AS builder
WORKDIR /app
RUN curl -sL https://taskfile.dev/install.sh | sh \
  && apt update && apt install -y musl-tools
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN /app/bin/task build

FROM debian:13.3-slim
RUN apt update \
    && apt install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
COPY --from=builder /app/dist/komodo-secrets-sync .
USER nobody
ENTRYPOINT ["/komodo-secrets-sync"]
