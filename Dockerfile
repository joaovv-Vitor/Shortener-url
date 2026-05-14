# --- Stage 1: Build ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /shorner ./cmd/server/

# --- Stage 2: Runtime ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /shorner /shorner

EXPOSE 8080

ENTRYPOINT ["/shorner"]
