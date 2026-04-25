# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go

# Run stage
FROM alpine:latest
WORKDIR /app

# Ensure we have CA certs for connecting to S3 / external DBs securely
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server .
# Copy migrations so the server can run them on startup
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
