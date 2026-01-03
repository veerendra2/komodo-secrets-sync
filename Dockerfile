FROM golang:1.25.5 AS builder
WORKDIR /app
RUN curl -sL https://taskfile.dev/install.sh | sh \
  && apt update && apt install -y musl-tools build-essential gcc musl-dev
COPY . .
RUN go mod download
RUN /app/bin/task build

FROM alpine:3.23.2
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /app/dist/komodo-secrets-injector .
USER nobody
ENTRYPOINT ["/komodo-secrets-injector"]